package cmd

import (
	"fmt"
	"os"

	"xnetperf/internal/service/lat"
	v0 "xnetperf/internal/v0"

	"github.com/spf13/cobra"
)

// ANSI color codes
const (
	colorRed   = "\033[31m"
	colorReset = "\033[0m"
)

// latencyThreshold is the threshold in microseconds for marking latency as high (red)
const latencyThreshold = 4.0

var latCmd = &cobra.Command{
	Use:   "lat",
	Short: "Execute latency testing workflow with N×N matrix results",
	Long: `Execute the latency testing workflow for measuring network latency between all HCA pairs:

0. Precheck - Verify network card status on all hosts
1. Generate latency test scripts using ib_write_lat (instead of ib_write_bw)
2. Run latency tests
3. Monitor test progress
4. Collect latency report files
5. Analyze results and display N×N latency matrix

Note: Latency testing currently only supports fullmesh mode. If your config uses
a different stream_type, a warning will be shown but testing will continue.

Examples:
  # Execute latency test with default config
  xnetperf lat

  # Execute with custom config file
  xnetperf lat -c /path/to/config.yaml`,
	Run: runLat,
}

func runLat(cmd *cobra.Command, args []string) {
	cfg := GetConfig()

	if cfg.Version == "v1" {
		latRunner := lat.New(cfg)
		if err := latRunner.Execute(); err != nil {
			fmt.Printf("❌ Latency test failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		v0.ExecuteLatCommand(cfg)
	}
}
