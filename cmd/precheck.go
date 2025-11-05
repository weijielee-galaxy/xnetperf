package cmd

import (
	"fmt"
	"os"

	"xnetperf/internal/service/precheck"
	v0 "xnetperf/internal/v0"

	"github.com/spf13/cobra"
)

const DESCRIPTION = `
Precheck and verify that all configured InfiniBand HCAs are in proper state.
This command checks both physical state (LinkUp) and logical state (ACTIVE) 
for each HCA on each host configured in the config file.

Only HCAs that are both LinkUp and ACTIVE are considered healthy.

Example:
  xnetperf precheck
`

var precheckCmd = &cobra.Command{
	Use:   "precheck",
	Short: "Precheck InfiniBand HCA status on all configured hosts",
	Long:  DESCRIPTION,
	Run:   runPrecheck,
}

func runPrecheck(cmd *cobra.Command, args []string) {
	cfg := GetConfig()

	if cfg.Version == "v1" {
		checker := precheck.New(cfg)
		checker.Display(checker.DoCheck())
		os.Exit(0)
	}

	success := v0.ExecPrecheckCommand(cfg)
	if !success {
		fmt.Println("\n❌ Precheck failed! Some HCAs are not in healthy state.")
		fmt.Println("Please fix the network issues before running performance tests.")
	} else {
		fmt.Println("\n✅ Precheck passed! All HCAs are healthy.")
	}
}
