package cmd

import (
	"xnetperf/internal/service/probe"
	v0 "xnetperf/internal/v0"

	"github.com/spf13/cobra"
)

var (
	probeInterval int
	oneShot       bool
)

var probeCmd = &cobra.Command{
	Use:   "probe",
	Short: "Probe ib_write_bw processes status on remote hosts",
	Long: `Probe and monitor ib_write_bw processes on all configured hosts.
By default, probes every 5 seconds until all processes complete.
Can be configured to run once or with custom intervals.

Examples:
  # Monitor with default 5s interval until completion
  xnetperf probe

  # Check once and exit
  xnetperf probe --once

  # Monitor with 10s interval
  xnetperf probe --interval 10`,
	Run: runProbe,
}

func init() {
	probeCmd.Flags().IntVar(&probeInterval, "interval", 5, "Probe interval in seconds")
	probeCmd.Flags().BoolVar(&oneShot, "once", false, "Probe once and exit without waiting")
}

func runProbe(cmd *cobra.Command, args []string) {
	cfg := GetConfig()
	if cfg.Version == "v1" {
		prober := probe.New(cfg)
		prober.DoProbeWait(probeInterval)
		return
	}

	v0.ExecProbeCommand(cfg, oneShot, probeInterval)
}
