package stream

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// 假设所有的服务器都有同样的HCA数目
func GenerateFullMeshScript(cfg *Config) {
	// 清空streamScript文件夹内容
	ClearStreamScriptDir(cfg)
	// 生成fullmesh脚本
	totalPort := len(cfg.Client.Hostname) * len(cfg.Client.Hca) * len(cfg.Server.Hostname) * len(cfg.Server.Hca) * 2
	fmt.Println("Total Ports Needed:", totalPort)
	if totalPort > (65535 - cfg.StartPort + 1) {
		fmt.Printf("Error: Not enough ports available. Required: %d, Available: %d\n", totalPort, 65535-cfg.StartPort+1)
		return
	}
	allServerHostName := append(cfg.Server.Hostname, cfg.Client.Hostname...)
	fmt.Println(allServerHostName)

	num := 1
	// port := cfg.StartPort

	for _, Server := range allServerHostName {
		port := cfg.StartPort
		// ip addr show bond0 | grep 'inet ' | awk '{print $2}' | cut -d'/' -f1
		command := fmt.Sprintf("ip addr show %s | grep 'inet ' | awk '{print $2}' | cut -d'/' -f1", "bond0")

		// 2. Create the command to be executed locally: ssh <hostname> "<command>"
		cmd := exec.Command("ssh", Server, command)

		// 3. Run the command and capture the combined output (stdout and stderr).
		output, err := cmd.CombinedOutput()
		if err != nil {
			// If command fails, return the output for debugging and a detailed error.
			fmt.Printf("Error executing command on %s: %v\nOutput: %s\n", Server, err, string(output))
		}
		fmt.Println("Output from", Server, ":", string(output))
		for _, hcaServer := range cfg.Client.Hca {
			var serverScriptContent, clientScriptContent string

			serverScriptFileName := fmt.Sprintf("%s/%s_%s_server_fullmesh.sh", cfg.OutputDir(), Server, hcaServer)
			// serverScriptContent := "#!/bin/bash\n"

			clientScriptFileName := fmt.Sprintf("%s/%s_%s_client_fullmesh.sh", cfg.OutputDir(), Server, hcaServer)
			// clientScriptContent := "#!/bin/bash\n"

			fmt.Println("Generating scripts for Server:", Server, "Client HCA:", hcaServer)
			for _, allHost := range allServerHostName {
				if allHost == Server {
					continue
				}
				for _, hcaClient := range cfg.Server.Hca {
					fmt.Println("num:", num, "Server HCA:", Server, "Server HCA:", hcaClient, port)
					fmt.Println("num:", num, "client HCA:", allHost, "Client HCA:", hcaClient, port, Server)
					port++
					num++
					// Append the command to scriptContent instead of overwriting
					serverScriptContent += fmt.Sprintf("ssh %s 'ib_write_bw -d %s --run_infinitely -m %d -p %d %s &'; %s\n", Server, hcaServer, cfg.MessageSizeBytes, port-1, Ignore, Sleep)
					clientScriptContent += fmt.Sprintf("ssh %s 'ib_write_bw -d %s --run_infinitely -m %d -p %d  %s %s &'; %s\n", allHost, hcaClient, cfg.MessageSizeBytes, port-1, strings.TrimSpace(string(output)), Ignore, Sleep)
				}
			}
			// Write the complete scriptContent to the file after the loops
			err := os.WriteFile(serverScriptFileName, []byte(serverScriptContent), 0755)
			if err != nil {
				fmt.Printf("Error writing script file '%s': %v\n", serverScriptFileName, err)
				return
			}

			// Write the complete scriptContent to the file after the loops
			err = os.WriteFile(clientScriptFileName, []byte(clientScriptContent), 0755)
			if err != nil {
				fmt.Printf("Error writing script file '%s': %v\n", serverScriptFileName, err)
				return
			}
		}
	}
}

func ClearPreviewScript(hosts []string) {
	var wg sync.WaitGroup

	for _, host := range hosts {
		// Increment the WaitGroup counter.
		wg.Add(1)

		// Launch a new goroutine.
		// We pass 'host' as an argument to the goroutine to ensure that
		// each goroutine gets its own copy of the hostname.
		go func(hostname string) {
			// Decrement the counter when the goroutine completes.
			defer wg.Done()

			fmt.Printf("-> Sending command to %s...\n", hostname)

			// Create the command: ssh <hostname> "killall ib_write_bw"
			cmd := exec.Command("ssh", hostname, "killall ib_write_bw")

			// Run the command and capture the combined standard output and standard error.
			output, err := cmd.CombinedOutput()

			// --- Analyze the result ---
			if err != nil {
				// The command failed, but we need to check if it's because the process
				// wasn't running, which is a "successful" outcome for us.
				if strings.Contains(string(output), "no process found") {
					fmt.Printf("✅ On %s: OK (process was not running).\n", hostname)
				} else {
					// This is a genuine error (e.g., SSH connection failed, permission denied).
					fmt.Printf("❌ On %s: FAILED. Error: %v\n", hostname, err)
					fmt.Printf("   └── Output: %s\n", string(output))
				}
			} else {
				// The command executed successfully with exit code 0.
				fmt.Printf("✅ On %s: SUCCESS (process killed).\n", hostname)
				// Print output if there is any (usually killall is silent on success).
				if len(strings.TrimSpace(string(output))) > 0 {
					fmt.Printf("   └── Output: %s\n", string(output))
				}
			}
		}(host)
	}
	wg.Wait()
	fmt.Println("All commands completed.")
}

func DistributeAndRunScripts(cfg *Config) {
	// 下发前先清空之前的结果
	fmt.Println("Distributing and running scripts...")
	allServerHostName := append(cfg.Server.Hostname, cfg.Client.Hostname...)
	fmt.Println(allServerHostName)
	ClearPreviewScript(allServerHostName)

	// 这里是分发和启动脚本的逻辑
	fmt.Println("Distributing and running scripts...")
	scriptDir := cfg.OutputDir()

	// Read all entries in the script directory.
	entries, err := os.ReadDir(scriptDir)
	if err != nil {
		log.Fatalf("❌ Failed to read directory '%s': %v", scriptDir, err)
	}

	processedFiles := 0
	sg := &sync.WaitGroup{}
	// 先下发服务端脚本，再下发客户端脚本
	for _, entry := range entries {
		// Skip subdirectories and non-shell files.
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sh") {
			continue
		}

		processedFiles++

		fileName := entry.Name()
		fmt.Printf("\nProcessing file: %s\n", fileName)

		parts := strings.Split(fileName, "_")
		if len(parts) < 1 {
			fmt.Printf("  -> Skipping: Invalid filename format.\n")
			continue
		}
		hostname := parts[0]
		fmt.Printf("  -> Extracted hostname: %s\n", hostname)

		if strings.Contains(fileName, "server") {
			fmt.Printf("server script: %s\n", fileName)
			// --- Read Script Content ---
			filePath := filepath.Join(scriptDir, fileName)
			scriptContent, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("  -> ❌ Error reading file %s: %v\n", filePath, err)
				continue
			}

			sg.Add(1)
			go func(hostname string, scriptContent []byte) {
				defer sg.Done()
				// --- Execute Remotely ---
				err = executeRemoteScript(hostname, scriptContent)
				if err != nil {
					fmt.Printf("  -> ❌ Execution failed: %v\n", err)
				} else {
					fmt.Println("  -> ✅ Execution successful.")
					time.Sleep(2000 * time.Millisecond)
				}
			}(hostname, scriptContent)
		}
	}
	sg.Wait()

	fmt.Printf("\nWaiting %d seconds before starting client scripts...\n", cfg.WaitingTimeSeconds)
	time.Sleep(time.Second * time.Duration(cfg.WaitingTimeSeconds))

	sshSG := &sync.WaitGroup{}
	//启动客户端脚本
	// 先下发服务端脚本，再下发客户端脚本
	for _, entry := range entries {
		// Skip subdirectories and non-shell files.
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sh") {
			continue
		}

		processedFiles++

		fileName := entry.Name()
		fmt.Printf("\nProcessing file: %s\n", fileName)

		parts := strings.Split(fileName, "_")
		if len(parts) < 1 {
			fmt.Printf("  -> Skipping: Invalid filename format.\n")
			continue
		}
		hostname := parts[0]
		fmt.Printf("  -> Extracted hostname: %s\n", hostname)

		if strings.Contains(fileName, "client") {
			fmt.Printf("server script: %s\n", fileName)
			// --- Read Script Content ---
			filePath := filepath.Join(scriptDir, fileName)
			scriptContent, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("  -> ❌ Error reading file %s: %v\n", filePath, err)
				continue
			}

			sshSG.Add(1)
			go func(hostname string, scriptContent []byte) {
				defer sshSG.Done()
				// --- Execute Remotely ---
				err = executeRemoteScript(hostname, scriptContent)
				if err != nil {
					fmt.Printf("  -> ❌ Execution failed: %v\n", err)
				} else {
					fmt.Println("  -> ✅ Execution successful.")
					time.Sleep(2000 * time.Millisecond)
				}
			}(hostname, scriptContent)
		}
	}

	if processedFiles == 0 {
		fmt.Println("\nNo '.sh' scripts found in the 'streamScript' directory.")
	} else {
		fmt.Println("\nAll scripts processed.")
	}
}

func executeRemoteScript(hostname string, scriptContent []byte) error {
	fmt.Printf("  -> Attempting to execute on host: %s\n", hostname)
	fmt.Printf("  -> Script Content:\n%s\n", string(scriptContent))

	// Command: ssh <hostname> "bash -s"
	// "bash -s" tells the remote bash shell to read commands from standard input.
	cmd := exec.Command("bash", "-c", string(scriptContent))

	// Run the command and wait for it to finish.
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute script on %s: %w", hostname, err)
	}
	return nil
}
