package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"xnetperf/config"
	"xnetperf/stream"

	"github.com/spf13/cobra"
)

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

	// Step 0: Precheck - Verify network card status before starting tests
	fmt.Println("\n🔍 Step 0/5: Performing network card precheck...")
	if !execPrecheckCommand(cfg) {
		fmt.Printf("❌ Precheck failed! Network cards are not ready. Please fix the issues before running latency tests.\n")
		os.Exit(1)
	}
	fmt.Println("✅ Precheck passed! All network cards are healthy. Proceeding with latency tests...")

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
	// Get all hosts list
	allHosts := make(map[string]bool)
	for _, host := range cfg.Server.Hostname {
		allHosts[host] = true
	}
	for _, host := range cfg.Client.Hostname {
		allHosts[host] = true
	}

	if len(allHosts) == 0 {
		fmt.Println("No hosts configured in config file")
		return
	}

	fmt.Printf("Probing ib_write_lat processes on %d hosts...\n", len(allHosts))
	fmt.Println("Mode: Continuous monitoring until all processes complete")
	fmt.Println()

	probeIntervalSec := 5

	for {
		results := probeLatencyAllHosts(allHosts, cfg.SSH.PrivateKey)
		displayLatencyProbeResults(results)

		// Check if all processes have completed
		allCompleted := true
		for _, result := range results {
			if result.ProcessCount > 0 {
				allCompleted = false
				break
			}
		}

		if allCompleted {
			fmt.Println("✅ All ib_write_lat processes have completed!")
			break
		}

		// Wait for next probe
		fmt.Printf("Waiting %d seconds for next probe...\n\n", probeIntervalSec)
		time.Sleep(time.Duration(probeIntervalSec) * time.Second)
	}
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
	Results struct {
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
	// Format: latency_c_sourceHost_sourceHCA_to_targetHost_targetHCA_pPORT.json
	// Example: latency_c_host2_mlx5_0_to_host1_mlx5_1_p20000.json
	filename := filepath.Base(filePath)

	// Remove extension
	nameWithoutExt := strings.TrimSuffix(filename, ".json")

	// Only process client reports (latency_c_*)
	if !strings.HasPrefix(nameWithoutExt, "latency_c_") {
		return nil, nil // Skip server reports
	}

	// Remove "latency_c_" prefix
	remaining := strings.TrimPrefix(nameWithoutExt, "latency_c_")

	// Split by "_to_" to separate source and target
	parts := strings.Split(remaining, "_to_")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid filename format (missing _to_): %s", filename)
	}

	// Parse source (format: host_hca)
	sourceParts := strings.SplitN(parts[0], "_", 2)
	if len(sourceParts) != 2 {
		return nil, fmt.Errorf("invalid source format in filename: %s", filename)
	}
	sourceHost := sourceParts[0]
	sourceHCA := sourceParts[1]

	// Parse target (format: host_hca_pPORT)
	// Need to find the last occurrence of _p to separate HCA from port
	targetStr := parts[1]
	pIndex := strings.LastIndex(targetStr, "_p")
	if pIndex == -1 {
		return nil, fmt.Errorf("invalid target format (missing _p prefix for port) in filename: %s", filename)
	}

	// Everything before "_p" is "host_hca"
	hostAndHCA := targetStr[:pIndex]
	targetParts := strings.SplitN(hostAndHCA, "_", 2)
	if len(targetParts) != 2 {
		return nil, fmt.Errorf("invalid target host_hca format in filename: %s", filename)
	}
	targetHost := targetParts[0]
	targetHCA := targetParts[1]

	// Read and parse JSON file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	var report LatencyReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	avgLatency := report.Results.TAvg

	latencyData := &LatencyData{
		SourceHost:   sourceHost,
		SourceHCA:    sourceHCA,
		TargetHost:   targetHost,
		TargetHCA:    targetHCA,
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

// LatencyProbeResult represents the probe result for ib_write_lat processes
type LatencyProbeResult struct {
	Hostname     string
	ProcessCount int
	Processes    []string
	Error        string
	Status       string
}

// probeLatencyAllHosts probes all hosts for ib_write_lat processes
func probeLatencyAllHosts(hosts map[string]bool, sshKeyPath string) []LatencyProbeResult {
	var results []LatencyProbeResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for hostname := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			result := probeLatencyHost(host, sshKeyPath)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(hostname)
	}

	wg.Wait()
	return results
}

// probeLatencyHost probes a single host for ib_write_lat processes
func probeLatencyHost(hostname, sshKeyPath string) LatencyProbeResult {
	result := LatencyProbeResult{
		Hostname: hostname,
	}

	// Use SSH to execute ps command to find ib_write_lat processes
	cmd := buildSSHCommand(hostname, "ps aux | grep ib_write_lat | grep -v grep", sshKeyPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// If no processes found or SSH connection failed
		if strings.Contains(string(output), "") && cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
			// ps command returning 1 usually means no matching processes found
			result.ProcessCount = 0
			result.Status = "COMPLETED"
		} else {
			result.Error = fmt.Sprintf("SSH error: %v", err)
			result.Status = "ERROR"
		}
		return result
	}

	// Parse output
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		result.ProcessCount = 0
		result.Status = "COMPLETED"
		return result
	}

	// Filter and count valid process lines
	var processes []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && strings.Contains(line, "ib_write_lat") {
			processes = append(processes, line)
		}
	}

	result.ProcessCount = len(processes)
	result.Processes = processes

	if result.ProcessCount > 0 {
		result.Status = "RUNNING"
	} else {
		result.Status = "COMPLETED"
	}

	return result
}

// displayLatencyProbeResults displays the probe results for ib_write_lat processes
func displayLatencyProbeResults(results []LatencyProbeResult) {
	fmt.Printf("=== Latency Probe Results (%s) ===\n", time.Now().Format("15:04:05"))
	fmt.Println("┌─────────────────────┬───────────────┬──────────────┬─────────────────┐")
	fmt.Println("│ Hostname            │ Status        │ Process Count│ Details         │")
	fmt.Println("├─────────────────────┼───────────────┼──────────────┼─────────────────┤")

	for _, result := range results {
		details := ""
		statusIcon := ""

		switch result.Status {
		case "RUNNING":
			statusIcon = "🟡 RUNNING"
			if result.ProcessCount > 0 {
				details = fmt.Sprintf("%d process(es)", result.ProcessCount)
			}
		case "COMPLETED":
			statusIcon = "✅ COMPLETED"
			details = "No processes"
		case "ERROR":
			statusIcon = "❌ ERROR"
			details = "Connection failed"
		}

		fmt.Printf("│ %-19s │ %-12s │ %12d │ %-15s │\n",
			result.Hostname, statusIcon, result.ProcessCount, details)

		// If there's an error, display error message on next line
		if result.Error != "" {
			fmt.Printf("│ %-19s │ %-12s │ %12s │ %-15s │\n",
				"", "Error:", "", result.Error)
		}
	}

	fmt.Println("└─────────────────────┴───────────────┴──────────────┴─────────────────┘")

	// Display summary
	running := 0
	completed := 0
	errors := 0
	totalProcesses := 0

	for _, result := range results {
		switch result.Status {
		case "RUNNING":
			running++
			totalProcesses += result.ProcessCount
		case "COMPLETED":
			completed++
		case "ERROR":
			errors++
		}
	}

	fmt.Printf("\nSummary: %d hosts running (%d processes), %d completed, %d errors\n",
		running, totalProcesses, completed, errors)
}
