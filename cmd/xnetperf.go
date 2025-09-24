package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"xnetperf/stream"

	"gopkg.in/yaml.v3"
)

func LoadConfig(filePath string) (*stream.Config, error) {
	var cfg stream.Config
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", filePath, err)
	}
	err = yaml.Unmarshal(yamlFile, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML content from '%s': %w", filePath, err)
	}
	return &cfg, nil
}

func handleStopCommand(cfg *stream.Config) {
	commandToStop := "killall ib_write_bw"
	allServerHostName := append(cfg.Server.Hostname, cfg.Client.Hostname...)

	fmt.Printf("[INFO] 'stop' command initiated. Sending '%s' to %d hosts...\n\n", commandToStop, len(allServerHostName))
	var wg sync.WaitGroup

	for _, hostname := range allServerHostName {
		wg.Add(1)
		go func(h string) {
			defer wg.Done()

			fmt.Printf("-> Contacting %s...\n", h)
			cmd := exec.Command("ssh", h, commandToStop)
			output, err := cmd.CombinedOutput()

			if err != nil {
				fmt.Printf("command failed on %s. Error: %v\n", hostname, err)
			}

			if err != nil {
				// Check if the "error" is simply because the process wasn't running.
				if strings.Contains(string(output), "no process found") {
					fmt.Printf("   [OK] ✅ On %s: Process was not running.\n", h)
				} else {
					// A genuine error occurred (e.g., connection failed, permission denied).
					fmt.Printf("   [ERROR] ❌ On %s: %v\n", h, err)
					fmt.Printf("      └── Output: %s\n", string(output))
				}
			} else {
				// The command succeeded.
				fmt.Printf("   [SUCCESS] ✅ On %s: Process killed.\n", h)
			}
		}(hostname)
	}

	wg.Wait()
	fmt.Println("\n[INFO] All 'stop' operations complete.")
}

func main() {
	cfg, err := LoadConfig("./config.yaml")
	if len(os.Args) > 1 {
		subcommand := os.Args[1]
		switch subcommand {
		case "stop":
			handleStopCommand(cfg)
			return
		}
	}

	if err != nil {
		fmt.Printf("Error reading config.yaml: %v\n", err)
		return
	}

	switch cfg.StreamType {
	case stream.FullMesh:
		stream.GenerateFullMeshScript(cfg)
	case stream.InCast:
		stream.GenerateIncastScripts(cfg)
	default:
		fmt.Printf("Invalid stream_type '%s' in config.yaml. Must be 'fullmesh' or 'incast'.\n", cfg.StreamType)
		return
	}

	// 提取streamScript文件夹下面的脚本，分发到对应的机器上启动，先启动server,然后启动client
	stream.DistributeAndRunScripts(cfg)
}
