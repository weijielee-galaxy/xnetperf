package stream

import (
	"fmt"
	"os"
	"os/exec"
)

// 假设所有的服务器都有同样的HCA数目
func GenerateIncastScripts(cfg *Config) {
	// 清空streamScript文件夹内容
	ClearStreamScriptDir(cfg)
	// 生成incast脚本
	totalPort := len(cfg.Client.Hostname) * len(cfg.Client.Hca) * len(cfg.Server.Hostname) * len(cfg.Server.Hca)
	fmt.Println("Total Ports Needed:", totalPort)
	if totalPort > (65535 - cfg.StartPort + 1) {
		fmt.Printf("Error: Not enough ports available. Required: %d, Available: %d\n", totalPort, 65535-cfg.StartPort+1)
		return
	}

	// 根据配置文件，生成在每个server的每个HCA上监听的脚本
	port := cfg.StartPort
	for _, sHost := range cfg.Server.Hostname {
		command := fmt.Sprintf("ip addr show %s | grep 'inet ' | awk '{print $2}' | cut -d'/' -f1", "bond0")
		// 2. Create the command to be executed locally: ssh <hostname> "<command>"
		cmd := exec.Command("ssh", sHost, command)
		// 3. Run the command and capture the combined output (stdout and stderr).
		hostIP, err := cmd.CombinedOutput()
		if err != nil {
			// If command fails, return the output for debugging and a detailed error.
			fmt.Printf("Error executing command on %s: %v\nOutput: %s\n", sHost, err, string(hostIP))
		}

		for _, sHca := range cfg.Server.Hca { // 对于每一个server的HCA，其要启动多少个监听脚本取决于client的HCA和client的hostname数量
			serverScriptFileName := fmt.Sprintf("%s/%s_%s_server_incast.sh", cfg.OutputDir(), sHost, sHca)
			clientScriptFileName := fmt.Sprintf("%s/%s_%s_client_incast.sh", cfg.OutputDir(), sHost, sHca)
			var serverScriptContent, clientScriptContent string
			for _, cHost := range cfg.Client.Hostname {
				for _, cHca := range cfg.Client.Hca {
					// Append the command to scriptContent instead of overwriting
					serverScriptContent += fmt.Sprintf("ssh %s ib_write_bw -d %s --run_infinitely -m %d -p %d &\n", sHost, sHca, cfg.MessageSizeBytes, port)
					clientScriptContent += fmt.Sprintf("ssh %s ib_write_bw -d %s --run_infinitely -m %d -p %d %s &\n", cHost, cHca, cfg.MessageSizeBytes, port, string(hostIP))
					port++
				}
			}
			err := os.WriteFile(serverScriptFileName, []byte(serverScriptContent), 0755)
			if err != nil {
				fmt.Printf("Error writing script file '%s': %v\n", serverScriptFileName, err)
				return
			}
			err = os.WriteFile(clientScriptFileName, []byte(clientScriptContent), 0755)
			if err != nil {
				fmt.Printf("Error writing script file '%s': %v\n", serverScriptFileName, err)
				return
			}
		}
	}
}
