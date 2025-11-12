package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"xnetperf/config"
	"xnetperf/pkg/tools"

	"github.com/spf13/cobra"
)

const COMMAND_STOP = "killall ib_write_bw"

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop all ib_write_bw processes",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig(cfgFile)
		if err != nil {
			fmt.Printf("Error reading config: %v\n", err)
			os.Exit(1)
		}
		handleStopCommand(cfg)
	},
}

func handleStopCommand(cfg *config.Config) {
	commandToStop := COMMAND_STOP
	allServerHostName := append(cfg.Server.Hostname, cfg.Client.Hostname...)

	fmt.Printf("[INFO] 'stop' command initiated. Sending '%s' to %d hosts...\n\n", commandToStop, len(allServerHostName))
	var wg sync.WaitGroup

	for _, hostname := range allServerHostName {
		wg.Add(1)
		go func(h string) {
			defer wg.Done()

			fmt.Printf("-> Contacting %s...\n", h)
			cmd := tools.BuildSSHCommand(h, commandToStop, cfg.SSH.PrivateKey, cfg.SSH.User)
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
