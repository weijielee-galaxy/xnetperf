package cmd

import (
	"fmt"
	"os"
	"strings"

	"xnetperf/config"
	"xnetperf/internal/script"
	"xnetperf/internal/service"
	v0 "xnetperf/internal/v0"

	"github.com/spf13/cobra"
)

var executeCmd = &cobra.Command{
	Use:   "execute",
	Short: "Execute complete test workflow: run -> probe -> collect -> analyze",
	Long: `Execute the complete network performance testing workflow:

1. Run network tests (equivalent to 'xnetperf run')
2. Monitor test progress with 5-second intervals (equivalent to 'xnetperf probe')
3. Collect report files and cleanup remote files (equivalent to 'xnetperf collect --cleanup')
4. Analyze results and display performance tables (equivalent to 'xnetperf analyze')

This command automates the entire testing process from start to finish.

Examples:
  # Execute complete workflow with default config
  xnetperf execute

  # Execute with custom config file
  xnetperf execute -c /path/to/config.yaml`,
	Run: runExecute,
}

func runExecute(cmd *cobra.Command, args []string) {
	fmt.Println("🚀 Starting complete xnetperf workflow...")
	fmt.Println(strings.Repeat("=", 60))

	// Load configuration once for all steps
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		fmt.Printf("❌ Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Step 1: Execute run command
	fmt.Println("\n📋 Step 1/4: Running network tests...")
	if !executeRunStep(cfg) {
		fmt.Println("❌ Run step failed. Aborting workflow.")
		os.Exit(1)
	}

	// Special handling for P2P stream type
	// 不需要展示效果
	if cfg.StreamType == config.P2P {
		fmt.Println("⚠️  P2P stream type detected. Skipping probe step as tests are short-lived.")
		os.Exit(0)
	}

	// Step 2: Execute probe command
	fmt.Println("\n🔍 Step 2/4: Monitoring test progress...")
	if !executeProbeStep(cfg) {
		fmt.Println("❌ Probe step failed. Aborting workflow.")
		os.Exit(1)
	}

	// Step 3: Execute collect command
	fmt.Println("\n📥 Step 3/4: Collecting reports...")
	if !executeCollectStep(cfg) {
		fmt.Println("❌ Collect step failed. Aborting workflow.")
		os.Exit(1)
	}

	// Step 4: Execute analyze command
	fmt.Println("\n📊 Step 4/4: Analyzing results...")
	if !executeAnalyzeStep(cfg) {
		fmt.Println("❌ Analyze step failed. Aborting workflow.")
		os.Exit(1)
	}

	fmt.Println("\n🎉 Complete xnetperf workflow finished successfully!")
	fmt.Println(strings.Repeat("=", 60))
}

// executeRunStep runs the network tests
func executeRunStep(cfg *config.Config) bool {
	fmt.Printf("Executing network tests (stream_type: %s)...\n", cfg.StreamType)

	if cfg.Version == "v1" {
		// executor 里面没有precheck 先添加在这里
		fmt.Println("\n🔍 Step 0/5: Performing network card precheck...")
		service.DisplayPrecheckResultsV2(service.Precheck(cfg))
		fmt.Println("✅ Precheck passed! All network cards are healthy. Proceeding with latency tests...")

		executor := script.NewExecutor(cfg, script.TestTypeBandwidth)
		if executor == nil {
			fmt.Println("❌ Unsupported stream type for v1 execute workflow. Aborting.")
			os.Exit(1)
		}
		fmt.Println("\n📋 Step 1/4: Running network tests...")
		err := executor.Execute()
		if err != nil {
			fmt.Printf("❌ Run step failed: %v. Aborting workflow.\n", err)
			os.Exit(1)
		}
	} else {
		v0.ExecRunCommand(cfg)
	}

	fmt.Println("✅ Network tests started successfully")
	return true
}

// executeProbeStep monitors the test progress using probe logic
func executeProbeStep(cfg *config.Config) bool {
	fmt.Println("Monitoring ib_write_bw processes (5-second intervals)...")

	execProbeCommand(cfg)
	return true
}

// executeCollectStep collects report files with cleanup
func executeCollectStep(cfg *config.Config) bool {
	if !cfg.Report.Enable {
		fmt.Println("⚠️  Report generation is disabled in config. Skipping collect step.")
		return true
	}

	// --cleanup=true
	cleanupRemote = true
	fmt.Println("Collecting report files from remote hosts...")
	err := execCollectCommand(cfg)
	if err != nil {
		fmt.Printf("❌ Error during report collection: %v\n", err)
		return false
	}

	fmt.Println("✅ Report collection completed successfully")
	return true
}

// executeAnalyzeStep analyzes the results
func executeAnalyzeStep(cfg *config.Config) bool {
	if !cfg.Report.Enable {
		fmt.Println("⚠️  Report generation is disabled in config. Skipping analyze step.")
		return true
	}

	fmt.Println("Analyzing performance results...")

	reportsDir := "reports"

	// Check if reports directory exists
	if _, err := os.Stat(reportsDir); os.IsNotExist(err) {
		fmt.Printf("❌ Reports directory not found: %s\n", reportsDir)
		return false
	}

	// Handle different stream types with separate analysis functions
	switch cfg.StreamType {
	case config.P2P:
		// P2P analysis
		p2pData, err := collectP2PReportData(reportsDir, cfg.SSH.PrivateKey, cfg.SSH.User)
		if err != nil {
			fmt.Printf("❌ Error collecting P2P report data: %v\n", err)
			return false
		}
		displayP2PResults(p2pData)
	default:
		// Traditional fullmesh/incast analysis
		clientData, serverData, err := collectReportData(reportsDir, cfg)
		if err != nil {
			fmt.Printf("❌ Error collecting report data: %v\n", err)
			return false
		}
		displayResults(clientData, serverData, cfg.Speed)
	}

	fmt.Println("✅ Analysis completed successfully")
	return true
}
