package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"xnetperf/config"

	"github.com/spf13/cobra"
)

const COMMAND_STOP_LAT = "killall ib_write_lat"

var stopLatCmd = &cobra.Command{
	Use:   "stoplat",
	Short: "Stop all ib_write_lat processes (latency tests)",
	Long: `Stop all running ib_write_lat processes on all configured hosts.
This is useful when latency tests encounter errors or need to be terminated manually.`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig(cfgFile)
		if err != nil {
			fmt.Printf("Error reading config: %v\n", err)
			os.Exit(1)
		}
		handleStopLatCommand(cfg)
	},
}

func init() {
	rootCmd.AddCommand(stopLatCmd)
}

func handleStopLatCommand(cfg *config.Config) {
	commandToStop := COMMAND_STOP_LAT
	allServerHostName := append(cfg.Server.Hostname, cfg.Client.Hostname...)

	fmt.Printf("[INFO] 'stoplat' command initiated. Sending '%s' to %d hosts...\n\n", commandToStop, len(allServerHostName))
	var wg sync.WaitGroup

	for _, hostname := range allServerHostName {
		wg.Add(1)
		go func(h string) {
			defer wg.Done()

			fmt.Printf("-> Contacting %s...\n", h)
			cmd := buildSSHCommand(h, commandToStop, cfg.SSH.PrivateKey)
			output, err := cmd.CombinedOutput()

			if err != nil {
				// Check if the "error" is simply because the process wasn't running.
				if strings.Contains(string(output), "no process found") {
					fmt.Printf("   [OK] ✅ On %s: No ib_write_lat process was running.\n", h)
				} else {
					// A genuine error occurred (e.g., connection failed, permission denied).
					fmt.Printf("   [ERROR] ❌ On %s: %v\n", h, err)
					fmt.Printf("      └── Output: %s\n", string(output))
				}
			} else {
				// The command succeeded.
				fmt.Printf("   [SUCCESS] ✅ On %s: ib_write_lat processes killed.\n", h)
			}
		}(hostname)
	}

	wg.Wait()
	fmt.Println("\n[INFO] All 'stoplat' operations complete.")
}
