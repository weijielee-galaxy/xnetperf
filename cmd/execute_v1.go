package cmd

import (
	"fmt"
	"os"
	"strings"

	"xnetperf/config"

	"github.com/spf13/cobra"
)

func runExecuteV1(cmd *cobra.Command, args []string) {
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
	if !executeRunStepV1(cfg) {
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
func executeRunStepV1(cfg *config.Config) bool {
	fmt.Printf("Executing network tests (stream_type: %s)...\n", cfg.StreamType)

	execRunCommandV1(cfg)

	fmt.Println("✅ Network tests started successfully")
	return true
}

// executeProbeStep monitors the test progress using probe logic
func executeProbeStepV1(cfg *config.Config) bool {
	fmt.Println("Monitoring ib_write_bw processes (5-second intervals)...")

	execProbeCommand(cfg)
	return true
}

// executeCollectStep collects report files with cleanup
func executeCollectStepV1(cfg *config.Config) bool {
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
func executeAnalyzeStepV1(cfg *config.Config) bool {
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
		p2pData, err := collectP2PReportData(reportsDir)
		if err != nil {
			fmt.Printf("❌ Error collecting P2P report data: %v\n", err)
			return false
		}
		displayP2PResults(p2pData)
	default:
		// Traditional fullmesh/incast analysis
		clientData, serverData, err := collectReportData(reportsDir)
		if err != nil {
			fmt.Printf("❌ Error collecting report data: %v\n", err)
			return false
		}
		displayResults(clientData, serverData, cfg.Speed)
	}

	fmt.Println("✅ Analysis completed successfully")
	return true
}
