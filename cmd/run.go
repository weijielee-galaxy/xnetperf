package cmd

import (
	"fmt"
	"os"
	"xnetperf/config"
	v0 "xnetperf/internal/v0"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run network test",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadConfig(cfgFile)
		if err != nil {
			fmt.Printf("Error reading config: %v\n", err)
			os.Exit(1)
		}
		v0.ExecRunCommand(cfg)
	},
}
