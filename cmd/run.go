package cmd

import (
	"fmt"
	"os"
	"xnetperf/config"
	"xnetperf/stream"

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
		switch cfg.StreamType {
		case config.FullMesh:
			stream.GenerateFullMeshScript(cfg)
		case config.InCast:
			stream.GenerateIncastScripts(cfg)
		default:
			fmt.Printf("Invalid stream_type '%s' in config.\n", cfg.StreamType)
			os.Exit(1)
		}
		stream.DistributeAndRunScripts(cfg)
	},
}
