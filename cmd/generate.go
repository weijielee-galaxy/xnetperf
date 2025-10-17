package cmd

import (
	"fmt"
	"os"
	"xnetperf/config"
	"xnetperf/stream"

	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate test scripts based on configuration file",
	Long: `Generate network test scripts according to the configuration file without executing them.
This is useful for reviewing the generated scripts before running tests.

The generated scripts will be saved to the output directory specified in the config file.

Example:
  xnetperf generate
  xnetperf generate --config custom-config.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig(cfgFile)
		if err != nil {
			fmt.Printf("Error reading config: %v\n", err)
			os.Exit(1)
		}
		execGenerateCommand(cfg)
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}

func execGenerateCommand(cfg *config.Config) {
	fmt.Printf("📝 Generating scripts for stream type: %s\n", cfg.StreamType)
	fmt.Printf("📁 Output directory: %s\n\n", cfg.OutputDir())

	if err := stream.GenerateScripts(cfg); err != nil {
		fmt.Printf("❌ Failed to generate scripts: %v\n", err)
		os.Exit(1)
	}
	// 显示生成的脚本文件列表
	displayGeneratedScripts(cfg)
}

func displayGeneratedScripts(cfg *config.Config) {
	fmt.Println("\n📋 Generated script files:")

	entries, err := os.ReadDir(cfg.OutputDir())
	if err != nil {
		fmt.Printf("Warning: Could not read output directory: %v\n", err)
		return
	}

	serverScripts := []string{}
	clientScripts := []string{}

	for _, entry := range entries {
		if !entry.IsDir() && (entry.Name()[len(entry.Name())-3:] == ".sh") {
			if containsString(entry.Name(), "_server_") {
				serverScripts = append(serverScripts, entry.Name())
			} else if containsString(entry.Name(), "_client_") {
				clientScripts = append(clientScripts, entry.Name())
			}
		}
	}

	if len(serverScripts) > 0 {
		fmt.Println("\n  Server scripts:")
		for _, script := range serverScripts {
			fmt.Printf("    - %s\n", script)
		}
	}

	if len(clientScripts) > 0 {
		fmt.Println("\n  Client scripts:")
		for _, script := range clientScripts {
			fmt.Printf("    - %s\n", script)
		}
	}

	if len(serverScripts) == 0 && len(clientScripts) == 0 {
		fmt.Println("    (No scripts generated)")
	}

	fmt.Printf("\n💡 Tip: You can review the generated scripts in %s before running them.\n", cfg.OutputDir())
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				indexOfSubstring(s, substr) >= 0)))
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
