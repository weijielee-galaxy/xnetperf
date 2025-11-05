package cmd

import (
	"log/slog"
	"xnetperf/internal/script"
	run "xnetperf/internal/service/runner"
	v0 "xnetperf/internal/v0"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run network test",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := GetConfig()
		if cfg.Version == "v1" {
			runner := run.New(cfg)
			err := runner.Run(script.TestTypeBandwidth)
			if err != nil {
				slog.Error("Run command failed", slog.Any("error", err))
			}
			return
		}
		v0.ExecRunCommand(cfg)
	},
}
