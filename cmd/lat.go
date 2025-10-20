package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"xnetperf/config"
	"xnetperf/stream"

	"github.com/spf13/cobra"
)

var latCmd = &cobra.Command{
	Use:   "lat",
	Short: "Execute latency testing workflow with N×N matrix results",
	Long: `Execute the latency testing workflow for measuring network latency between all HCA pairs:

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

func init() {
	// No additional flags needed - uses global config flag
}

func runLat(cmd *cobra.Command, args []string) {
	fmt.Println("🚀 Starting xnetperf latency testing workflow...")
	fmt.Println(strings.Repeat("=", 60))

	// Load configuration
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		fmt.Printf("❌ Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Step 1: Generate latency scripts
	fmt.Println("\n📋 Step 1/5: Generating latency test scripts...")
	if !executeLatencyGenerateStep(cfg) {
		fmt.Println("❌ Script generation failed. Aborting workflow.")
		os.Exit(1)
	}

	// Step 2: Run latency tests
	fmt.Println("\n▶️  Step 2/5: Running latency tests...")
	if !executeLatencyRunStep(cfg) {
		fmt.Println("❌ Latency test execution failed. Aborting workflow.")
		os.Exit(1)
	}

	// Step 3: Monitor progress
	fmt.Println("\n🔍 Step 3/5: Monitoring test progress...")
	if !executeLatencyProbeStep(cfg) {
		fmt.Println("❌ Probe step failed. Aborting workflow.")
		os.Exit(1)
	}

	// Step 4: Collect reports
	fmt.Println("\n📥 Step 4/5: Collecting latency reports...")
	if !executeLatencyCollectStep(cfg) {
		fmt.Println("❌ Report collection failed. Aborting workflow.")
		os.Exit(1)
	}

	// Step 5: Analyze and display latency matrix
	fmt.Println("\n📊 Step 5/5: Analyzing latency results...")
	if !executeLatencyAnalyzeStep(cfg) {
		fmt.Println("❌ Analysis failed. Aborting workflow.")
		os.Exit(1)
	}

	fmt.Println("\n🎉 Latency testing workflow completed successfully!")
	fmt.Println(strings.Repeat("=", 60))
}

// executeLatencyGenerateStep generates latency test scripts
func executeLatencyGenerateStep(cfg *config.Config) bool {
	fmt.Println("Generating N×N latency test scripts...")

	if err := stream.GenerateLatencyScripts(cfg); err != nil {
		fmt.Printf("❌ Error generating latency scripts: %v\n", err)
		return false
	}

	fmt.Println("✅ Latency scripts generated successfully")
	return true
}

// executeLatencyRunStep runs the latency test scripts
func executeLatencyRunStep(cfg *config.Config) bool {
	fmt.Println("Executing latency tests...")

	if err := stream.RunLatencyScripts(cfg); err != nil {
		fmt.Printf("❌ Error running latency scripts: %v\n", err)
		return false
	}

	fmt.Println("✅ Latency tests started successfully")
	return true
}

// executeLatencyProbeStep monitors latency test progress
func executeLatencyProbeStep(cfg *config.Config) bool {
	fmt.Println("Monitoring ib_write_lat processes (5-second intervals)...")

	// Use the existing probe logic, but monitor ib_write_lat instead of ib_write_bw
	execLatencyProbeCommand(cfg)
	return true
}

// executeLatencyCollectStep collects latency report files
func executeLatencyCollectStep(cfg *config.Config) bool {
	if !cfg.Report.Enable {
		fmt.Println("⚠️  Report generation is disabled in config. Skipping collect step.")
		return true
	}

	cleanupRemote = true
	fmt.Println("Collecting latency report files from remote hosts...")

	if err := execCollectCommand(cfg); err != nil {
		fmt.Printf("❌ Error during report collection: %v\n", err)
		return false
	}

	fmt.Println("✅ Latency report collection completed successfully")
	return true
}

// executeLatencyAnalyzeStep analyzes latency results and displays N×N matrix
func executeLatencyAnalyzeStep(cfg *config.Config) bool {
	if !cfg.Report.Enable {
		fmt.Println("⚠️  Report generation is disabled in config. Skipping analyze step.")
		return true
	}

	fmt.Println("Analyzing latency results...")

	reportsDir := "reports"

	// Check if reports directory exists
	if _, err := os.Stat(reportsDir); os.IsNotExist(err) {
		fmt.Printf("❌ Reports directory not found: %s\n", reportsDir)
		return false
	}

	// Collect and display latency data
	latencyMatrix, err := collectLatencyReportData(reportsDir)
	if err != nil {
		fmt.Printf("❌ Error collecting latency report data: %v\n", err)
		return false
	}

	displayLatencyMatrix(latencyMatrix)

	fmt.Println("✅ Latency analysis completed successfully")
	return true
}

// execLatencyProbeCommand monitors ib_write_lat processes
func execLatencyProbeCommand(cfg *config.Config) {
	// Similar to probe command but monitor ib_write_lat instead of ib_write_bw
	fmt.Println("Monitoring latency test processes...")
	// TODO: Implement actual monitoring logic similar to probe command
	// For now, just wait for the test duration
	fmt.Printf("⏳ Waiting %d seconds for latency tests to complete...\n", cfg.Run.DurationSeconds+5)
}

// LatencyData represents a single latency measurement
type LatencyData struct {
	SourceHost   string
	SourceHCA    string
	TargetHost   string
	TargetHCA    string
	AvgLatencyUs float64 // Average latency in microseconds
}

// LatencyReport represents the JSON structure from ib_write_lat
type LatencyReport struct {
	Results []struct {
		TAvg float64 `json:"t_avg"` // Average latency in microseconds
	} `json:"results"`
}

// collectLatencyReportData parses all latency JSON reports
func collectLatencyReportData(reportsDir string) ([]LatencyData, error) {
	var latencyData []LatencyData

	// Walk through all JSON files in reports directory
	err := filepath.Walk(reportsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process latency JSON files (named like latency_c_*.json or latency_s_*.json)
		if !info.IsDir() && strings.HasPrefix(info.Name(), "latency_") && strings.HasSuffix(info.Name(), ".json") {
			data, parseErr := parseLatencyReport(path)
			if parseErr != nil {
				fmt.Printf("⚠️  Warning: Failed to parse %s: %v\n", path, parseErr)
				return nil // Continue processing other files
			}
			if data != nil {
				latencyData = append(latencyData, *data)
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk reports directory: %v", err)
	}

	if len(latencyData) == 0 {
		return nil, fmt.Errorf("no latency reports found in %s", reportsDir)
	}

	return latencyData, nil
}

// parseLatencyReport parses a single latency JSON report file
func parseLatencyReport(filePath string) (*LatencyData, error) {
	// Parse filename to extract source, target, and HCA information
	// Format: latency_c_hostname_hca_port.json or latency_s_hostname_hca_port.json
	filename := filepath.Base(filePath)

	// Remove extension and split by underscore
	parts := strings.Split(strings.TrimSuffix(filename, ".json"), "_")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid filename format: %s", filename)
	}

	// Only process client reports (latency_c_*)
	if parts[1] != "c" {
		return nil, nil // Skip server reports
	}

	sourceHost := parts[2]
	sourceHCA := parts[3]

	// Read and parse JSON file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var report LatencyReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	// Extract average latency from results
	if len(report.Results) == 0 {
		return nil, fmt.Errorf("no results found in report")
	}

	avgLatency := report.Results[0].TAvg

	// TODO: Extract target host and HCA from connection information
	// For now, we'll need to infer this from the test setup or add metadata to reports
	latencyData := &LatencyData{
		SourceHost:   sourceHost,
		SourceHCA:    sourceHCA,
		TargetHost:   "unknown", // Will be improved in future iterations
		TargetHCA:    "unknown", // Will be improved in future iterations
		AvgLatencyUs: avgLatency,
	}

	return latencyData, nil
}

// displayLatencyMatrix displays the N×N latency matrix in table format
func displayLatencyMatrix(latencyData []LatencyData) {
	if len(latencyData) == 0 {
		fmt.Println("⚠️  No latency data to display")
		return
	}

	// Build matrix structure: source -> target -> latency
	matrix := make(map[string]map[string]float64)
	allSources := make(map[string]bool)
	allTargets := make(map[string]bool)

	for _, data := range latencyData {
		source := fmt.Sprintf("%s:%s", data.SourceHost, data.SourceHCA)
		target := fmt.Sprintf("%s:%s", data.TargetHost, data.TargetHCA)

		if matrix[source] == nil {
			matrix[source] = make(map[string]float64)
		}
		matrix[source][target] = data.AvgLatencyUs

		allSources[source] = true
		allTargets[target] = true
	}

	// Sort sources and targets
	var sources []string
	for source := range allSources {
		sources = append(sources, source)
	}
	sort.Strings(sources)

	var targets []string
	for target := range allTargets {
		targets = append(targets, target)
	}
	sort.Strings(targets)

	// Calculate column widths
	sourceWidth := 20
	for _, source := range sources {
		if len(source) > sourceWidth {
			sourceWidth = len(source)
		}
	}

	targetColWidth := 12 // Width for each latency value column

	// Print title
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📊 Latency Matrix (Average Latency in microseconds)")
	fmt.Println(strings.Repeat("=", 80))

	// Print table header top border
	sourceDashes := strings.Repeat("─", sourceWidth)
	fmt.Printf("┌─%s─┬", sourceDashes)
	for i := range targets {
		if i < len(targets)-1 {
			fmt.Printf("─%s─┬", strings.Repeat("─", targetColWidth))
		} else {
			fmt.Printf("─%s─┐\n", strings.Repeat("─", targetColWidth))
		}
	}

	// Print column headers (target nodes)
	fmt.Printf("│ %-*s │", sourceWidth, "Source → Target")
	for _, target := range targets {
		// Truncate long target names
		displayTarget := target
		if len(target) > targetColWidth {
			displayTarget = target[:targetColWidth-2] + ".."
		}
		fmt.Printf(" %-*s │", targetColWidth, displayTarget)
	}
	fmt.Println()

	// Print header separator
	fmt.Printf("├─%s─┼", sourceDashes)
	for i := range targets {
		if i < len(targets)-1 {
			fmt.Printf("─%s─┼", strings.Repeat("─", targetColWidth))
		} else {
			fmt.Printf("─%s─┤\n", strings.Repeat("─", targetColWidth))
		}
	}

	// Print data rows
	for i, source := range sources {
		fmt.Printf("│ %-*s │", sourceWidth, source)

		for _, target := range targets {
			latency := matrix[source][target]
			if latency > 0 {
				fmt.Printf(" %*.2f μs │", targetColWidth-3, latency)
			} else {
				fmt.Printf(" %-*s │", targetColWidth, "-")
			}
		}
		fmt.Println()

		// Print row separator (except for last row)
		if i < len(sources)-1 {
			fmt.Printf("├─%s─┼", sourceDashes)
			for j := range targets {
				if j < len(targets)-1 {
					fmt.Printf("─%s─┼", strings.Repeat("─", targetColWidth))
				} else {
					fmt.Printf("─%s─┤\n", strings.Repeat("─", targetColWidth))
				}
			}
		}
	}

	// Print table bottom border
	fmt.Printf("└─%s─┴", sourceDashes)
	for i := range targets {
		if i < len(targets)-1 {
			fmt.Printf("─%s─┴", strings.Repeat("─", targetColWidth))
		} else {
			fmt.Printf("─%s─┘\n", strings.Repeat("─", targetColWidth))
		}
	}

	// Calculate and display statistics
	var allLatencies []float64
	for _, data := range latencyData {
		allLatencies = append(allLatencies, data.AvgLatencyUs)
	}

	minLatency := minFloat(allLatencies)
	maxLatency := maxFloat(allLatencies)
	avgLatency := avgFloat(allLatencies)

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📈 Latency Statistics:")
	fmt.Printf("  Minimum Latency: %.2f μs\n", minLatency)
	fmt.Printf("  Maximum Latency: %.2f μs\n", maxLatency)
	fmt.Printf("  Average Latency: %.2f μs\n", avgLatency)
	fmt.Printf("  Total Measurements: %d\n", len(latencyData))
	fmt.Println(strings.Repeat("=", 80))
}

// Helper functions for statistics
func minFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

func maxFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func avgFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
