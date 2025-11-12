package cmd

import (
	"xnetperf/internal/service/analyze"
	v0 "xnetperf/internal/v0"

	"github.com/spf13/cobra"
)

var generateMD bool
var reportsPath string

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze network performance reports and display results in table format",
	Long: `Analyze JSON report files in the reports directory and display bandwidth
statistics in a formatted table. Separates client (TX) and server (RX) data.
Can optionally generate a Markdown table file.`,
	Run: runAnalyze,
}

func init() {
	analyzeCmd.Flags().BoolVar(&generateMD, "markdown", false, "Generate markdown table file")
	analyzeCmd.Flags().StringVar(&reportsPath, "reports-dir", "reports", "Path to the reports directory")
}

func runAnalyze(cmd *cobra.Command, args []string) {
	cfg := GetConfig()
	switch cfg.Version {
	case "v0":
		v0.ExecAnalyzeCommand(cfg, reportsPath, generateMD)
	default:
		analyzeer := analyze.New(cfg)
		analyzeer.DoAnalyze(reportsPath, generateMD)
	}
}
