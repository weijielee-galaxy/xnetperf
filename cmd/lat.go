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

	// Display based on stream type
	if cfg.StreamType == config.InCast {
		displayLatencyMatrixIncast(latencyMatrix, cfg)
	} else {
		// Default to fullmesh display
		displayLatencyMatrix(latencyMatrix)
	}

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
	// Formats:
	//   Fullmesh: latency_fullmesh_c_sourceHost_sourceHCA_to_targetHost_targetHCA_pPORT.json
	//   Incast:   latency_incast_c_sourceHost_sourceHCA_to_targetHost_targetHCA_pPORT.json
	//   Legacy:   latency_c_sourceHost_sourceHCA_to_targetHost_targetHCA_pPORT.json
	filename := filepath.Base(filePath)

	// Remove extension
	nameWithoutExt := strings.TrimSuffix(filename, ".json")

	// Only process client reports (latency_*_c_* or latency_c_*)
	if !strings.Contains(nameWithoutExt, "_c_") {
		return nil, nil // Skip server reports
	}

	// Remove prefix (latency_fullmesh_c_, latency_incast_c_, or latency_c_)
	var remaining string
	if strings.HasPrefix(nameWithoutExt, "latency_fullmesh_c_") {
		remaining = strings.TrimPrefix(nameWithoutExt, "latency_fullmesh_c_")
	} else if strings.HasPrefix(nameWithoutExt, "latency_incast_c_") {
		remaining = strings.TrimPrefix(nameWithoutExt, "latency_incast_c_")
	} else if strings.HasPrefix(nameWithoutExt, "latency_c_") {
		remaining = strings.TrimPrefix(nameWithoutExt, "latency_c_")
	} else {
		return nil, fmt.Errorf("invalid filename format: %s", filename)
	}

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

	// Build matrix structure and collect unique hosts/HCAs
	type HostHCA struct {
		Host string
		HCA  string
	}

	sourceHostHCAs := make(map[string][]string)   // host -> []hca
	targetHostHCAs := make(map[string][]string)   // host -> []hca
	matrix := make(map[string]map[string]float64) // "host:hca" -> "host:hca" -> latency

	for _, data := range latencyData {
		// Track source hosts and HCAs
		if sourceHostHCAs[data.SourceHost] == nil {
			sourceHostHCAs[data.SourceHost] = []string{}
		}
		found := false
		for _, hca := range sourceHostHCAs[data.SourceHost] {
			if hca == data.SourceHCA {
				found = true
				break
			}
		}
		if !found {
			sourceHostHCAs[data.SourceHost] = append(sourceHostHCAs[data.SourceHost], data.SourceHCA)
		}

		// Track target hosts and HCAs
		if targetHostHCAs[data.TargetHost] == nil {
			targetHostHCAs[data.TargetHost] = []string{}
		}
		found = false
		for _, hca := range targetHostHCAs[data.TargetHost] {
			if hca == data.TargetHCA {
				found = true
				break
			}
		}
		if !found {
			targetHostHCAs[data.TargetHost] = append(targetHostHCAs[data.TargetHost], data.TargetHCA)
		}

		// Build matrix
		sourceKey := fmt.Sprintf("%s:%s", data.SourceHost, data.SourceHCA)
		targetKey := fmt.Sprintf("%s:%s", data.TargetHost, data.TargetHCA)
		if matrix[sourceKey] == nil {
			matrix[sourceKey] = make(map[string]float64)
		}
		matrix[sourceKey][targetKey] = data.AvgLatencyUs
	}

	// Sort hosts and their HCAs
	var sourceHosts []string
	for host := range sourceHostHCAs {
		sourceHosts = append(sourceHosts, host)
		sort.Strings(sourceHostHCAs[host])
	}
	sort.Strings(sourceHosts)

	var targetHosts []string
	for host := range targetHostHCAs {
		targetHosts = append(targetHosts, host)
		sort.Strings(targetHostHCAs[host])
	}
	sort.Strings(targetHosts)

	// Calculate column widths
	hostColWidth := 10 // Minimum width for hostname column
	for _, host := range sourceHosts {
		if len(host) > hostColWidth && len(host) <= 20 {
			hostColWidth = len(host)
		} else if len(host) > 20 {
			hostColWidth = 20 // Cap at 20
		}
	}
	for _, host := range targetHosts {
		if len(host) > hostColWidth && len(host) <= 20 {
			hostColWidth = len(host)
		} else if len(host) > 20 {
			hostColWidth = 20
		}
	}

	hcaColWidth := 10 // Minimum width for HCA column
	for _, hcas := range sourceHostHCAs {
		for _, hca := range hcas {
			if len(hca) > hcaColWidth && len(hca) <= 15 {
				hcaColWidth = len(hca)
			} else if len(hca) > 15 {
				hcaColWidth = 15 // Cap at 15
			}
		}
	}

	valueColWidth := 12 // Width for latency values (e.g., "123.45 μs")

	// Print title
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📊 Latency Matrix (Average Latency in microseconds)")
	fmt.Println(strings.Repeat("=", 80))

	// Count total target columns
	totalTargetCols := 0
	for _, host := range targetHosts {
		totalTargetCols += len(targetHostHCAs[host])
	}

	// Print top border
	fmt.Printf("┌%s┬%s┬",
		strings.Repeat("─", hostColWidth+2),
		strings.Repeat("─", hcaColWidth+2))
	for i, targetHost := range targetHosts {
		numHCAs := len(targetHostHCAs[targetHost])
		width := numHCAs*valueColWidth + (numHCAs-1)*3 + 2
		if i < len(targetHosts)-1 {
			fmt.Printf("%s┬", strings.Repeat("─", width))
		} else {
			fmt.Printf("%s┐\n", strings.Repeat("─", width))
		}
	}

	// Print first header row (target hostnames)
	fmt.Printf("│%*s│%*s│",
		hostColWidth+2, " ",
		hcaColWidth+2, " ")
	for i, targetHost := range targetHosts {
		numHCAs := len(targetHostHCAs[targetHost])
		width := numHCAs*valueColWidth + (numHCAs-1)*3
		displayHost := targetHost
		if len(targetHost) > width {
			displayHost = targetHost[:width-2] + ".."
		}
		if i < len(targetHosts)-1 {
			fmt.Printf(" %-*s │", width, displayHost)
		} else {
			fmt.Printf(" %-*s │\n", width, displayHost)
		}
	}

	// Print separator between hostname row and HCA row
	fmt.Printf("│%*s│%*s├",
		hostColWidth+2, " ",
		hcaColWidth+2, " ")
	for i, targetHost := range targetHosts {
		hcas := targetHostHCAs[targetHost]
		for j := range hcas {
			if j < len(hcas)-1 {
				fmt.Printf("%s┬", strings.Repeat("─", valueColWidth+2))
			} else {
				if i < len(targetHosts)-1 {
					fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
				} else {
					fmt.Printf("%s┤\n", strings.Repeat("─", valueColWidth+2))
				}
			}
		}
	}

	// Print second header row (target HCAs)
	fmt.Printf("│%*s│%*s│",
		hostColWidth+2, " ",
		hcaColWidth+2, " ")
	for i, targetHost := range targetHosts {
		hcas := targetHostHCAs[targetHost]
		for j, hca := range hcas {
			displayHCA := hca
			if len(hca) > valueColWidth {
				displayHCA = hca[:valueColWidth-2] + ".."
			}
			if j < len(hcas)-1 || i < len(targetHosts)-1 {
				fmt.Printf(" %-*s │", valueColWidth, displayHCA)
			} else {
				fmt.Printf(" %-*s │\n", valueColWidth, displayHCA)
			}
		}
	}

	// Print header separator
	fmt.Printf("├%s┼%s┼",
		strings.Repeat("─", hostColWidth+2),
		strings.Repeat("─", hcaColWidth+2))
	for i := 0; i < totalTargetCols; i++ {
		if i < totalTargetCols-1 {
			fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
		} else {
			fmt.Printf("%s┤\n", strings.Repeat("─", valueColWidth+2))
		}
	}

	// Print data rows
	rowIdx := 0
	for _, sourceHost := range sourceHosts {
		sourceHCAs := sourceHostHCAs[sourceHost]
		for hcaIdx, sourceHCA := range sourceHCAs {
			// Print hostname in first column (only for first HCA of this host)
			if hcaIdx == 0 {
				displayHost := sourceHost
				if len(sourceHost) > hostColWidth {
					displayHost = sourceHost[:hostColWidth-2] + ".."
				}
				fmt.Printf("│ %-*s │", hostColWidth, displayHost)
			} else {
				fmt.Printf("│%*s│", hostColWidth+2, " ")
			}

			// Print HCA in second column
			displayHCA := sourceHCA
			if len(sourceHCA) > hcaColWidth {
				displayHCA = sourceHCA[:hcaColWidth-2] + ".."
			}
			fmt.Printf(" %-*s │", hcaColWidth, displayHCA)

			// Print latency values
			for _, targetHost := range targetHosts {
				targetHCAs := targetHostHCAs[targetHost]
				for _, targetHCA := range targetHCAs {
					sourceKey := fmt.Sprintf("%s:%s", sourceHost, sourceHCA)
					targetKey := fmt.Sprintf("%s:%s", targetHost, targetHCA)
					latency := matrix[sourceKey][targetKey]

					if latency > 0 {
						valueStr := fmt.Sprintf("%.2f μs", latency)
						fmt.Printf(" %*s │", valueColWidth, valueStr)
					} else {
						fmt.Printf(" %*s │", valueColWidth, "-")
					}
				}
			}
			fmt.Println()

			rowIdx++

			// Print row separator
			needsSeparator := false
			isLastHCAOfHost := hcaIdx == len(sourceHCAs)-1
			isLastHost := sourceHost == sourceHosts[len(sourceHosts)-1]

			if !isLastHost || !isLastHCAOfHost {
				if isLastHCAOfHost {
					// Separator between different hosts (with left border crossing hostname column)
					fmt.Printf("├%s┼%s┼",
						strings.Repeat("─", hostColWidth+2),
						strings.Repeat("─", hcaColWidth+2))
					needsSeparator = true
				} else {
					// Separator within same host (hostname column stays empty)
					fmt.Printf("│%*s├%s┼",
						hostColWidth+2, " ",
						strings.Repeat("─", hcaColWidth+2))
					needsSeparator = true
				}

				if needsSeparator {
					for i := 0; i < totalTargetCols; i++ {
						if i < totalTargetCols-1 {
							fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
						} else {
							fmt.Printf("%s┤\n", strings.Repeat("─", valueColWidth+2))
						}
					}
				}
			}
		}
	}

	// Print bottom border
	fmt.Printf("└%s┴%s┴",
		strings.Repeat("─", hostColWidth+2),
		strings.Repeat("─", hcaColWidth+2))
	for i := 0; i < totalTargetCols; i++ {
		if i < totalTargetCols-1 {
			fmt.Printf("%s┴", strings.Repeat("─", valueColWidth+2))
		} else {
			fmt.Printf("%s┘\n", strings.Repeat("─", valueColWidth+2))
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

// displayLatencyMatrixIncast displays the client×server latency matrix for incast mode
func displayLatencyMatrixIncast(latencyData []LatencyData, cfg *config.Config) {
	if len(latencyData) == 0 {
		fmt.Println("⚠️  No latency data to display")
		return
	}

	// Build matrix structure for incast mode (clients → servers)
	clientHostHCAs := make(map[string][]string)   // client host -> []hca
	serverHostHCAs := make(map[string][]string)   // server host -> []hca
	matrix := make(map[string]map[string]float64) // "client:hca" -> "server:hca" -> latency

	// Separate clients and servers based on config
	clientHostSet := make(map[string]bool)
	for _, host := range cfg.Client.Hostname {
		clientHostSet[host] = true
	}

	for _, data := range latencyData {
		// Determine if this is client→server or vice versa based on config
		isClientToServer := clientHostSet[data.SourceHost]

		if isClientToServer {
			// Track client (source) hosts and HCAs
			if clientHostHCAs[data.SourceHost] == nil {
				clientHostHCAs[data.SourceHost] = []string{}
			}
			found := false
			for _, hca := range clientHostHCAs[data.SourceHost] {
				if hca == data.SourceHCA {
					found = true
					break
				}
			}
			if !found {
				clientHostHCAs[data.SourceHost] = append(clientHostHCAs[data.SourceHost], data.SourceHCA)
			}

			// Track server (target) hosts and HCAs
			if serverHostHCAs[data.TargetHost] == nil {
				serverHostHCAs[data.TargetHost] = []string{}
			}
			found = false
			for _, hca := range serverHostHCAs[data.TargetHost] {
				if hca == data.TargetHCA {
					found = true
					break
				}
			}
			if !found {
				serverHostHCAs[data.TargetHost] = append(serverHostHCAs[data.TargetHost], data.TargetHCA)
			}

			// Build matrix
			clientKey := fmt.Sprintf("%s:%s", data.SourceHost, data.SourceHCA)
			serverKey := fmt.Sprintf("%s:%s", data.TargetHost, data.TargetHCA)
			if matrix[clientKey] == nil {
				matrix[clientKey] = make(map[string]float64)
			}
			matrix[clientKey][serverKey] = data.AvgLatencyUs
		}
	}

	// Sort hosts and their HCAs
	var clientHosts []string
	for host := range clientHostHCAs {
		clientHosts = append(clientHosts, host)
		sort.Strings(clientHostHCAs[host])
	}
	sort.Strings(clientHosts)

	var serverHosts []string
	for host := range serverHostHCAs {
		serverHosts = append(serverHosts, host)
		sort.Strings(serverHostHCAs[host])
	}
	sort.Strings(serverHosts)

	// Calculate column widths
	hostColWidth := 10
	for _, host := range clientHosts {
		if len(host) > hostColWidth && len(host) <= 20 {
			hostColWidth = len(host)
		} else if len(host) > 20 {
			hostColWidth = 20
		}
	}
	for _, host := range serverHosts {
		if len(host) > hostColWidth && len(host) <= 20 {
			hostColWidth = len(host)
		} else if len(host) > 20 {
			hostColWidth = 20
		}
	}

	hcaColWidth := 10
	for _, hcas := range clientHostHCAs {
		for _, hca := range hcas {
			if len(hca) > hcaColWidth && len(hca) <= 15 {
				hcaColWidth = len(hca)
			} else if len(hca) > 15 {
				hcaColWidth = 15
			}
		}
	}

	valueColWidth := 12

	// Print title
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📊 Latency Matrix - INCAST Mode (Client → Server)")
	fmt.Println("   Average Latency in microseconds")
	fmt.Println(strings.Repeat("=", 80))

	// Print top border
	fmt.Printf("┌%s┬%s┬",
		strings.Repeat("─", hostColWidth+2),
		strings.Repeat("─", hcaColWidth+2))
	for i, serverHost := range serverHosts {
		numHCAs := len(serverHostHCAs[serverHost])
		width := numHCAs*valueColWidth + (numHCAs-1)*3 + 2
		if i < len(serverHosts)-1 {
			fmt.Printf("%s┬", strings.Repeat("─", width))
		} else {
			fmt.Printf("%s┐\n", strings.Repeat("─", width))
		}
	}

	// Print first header row (server hostnames)
	fmt.Printf("│%*s│%*s│",
		hostColWidth+2, " ",
		hcaColWidth+2, " ")
	for i, serverHost := range serverHosts {
		numHCAs := len(serverHostHCAs[serverHost])
		width := numHCAs*valueColWidth + (numHCAs-1)*3
		displayHost := serverHost
		if len(serverHost) > width {
			displayHost = serverHost[:width-2] + ".."
		}
		if i < len(serverHosts)-1 {
			fmt.Printf(" %-*s │", width, displayHost)
		} else {
			fmt.Printf(" %-*s │\n", width, displayHost)
		}
	}

	// Print separator between hostname row and HCA row
	fmt.Printf("│%*s│%*s├",
		hostColWidth+2, " ",
		hcaColWidth+2, " ")
	for i, serverHost := range serverHosts {
		hcas := serverHostHCAs[serverHost]
		for j := range hcas {
			if j < len(hcas)-1 {
				fmt.Printf("%s┬", strings.Repeat("─", valueColWidth+2))
			} else {
				if i < len(serverHosts)-1 {
					fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
				} else {
					fmt.Printf("%s┤\n", strings.Repeat("─", valueColWidth+2))
				}
			}
		}
	}

	// Print second header row (server HCAs)
	fmt.Printf("│%*s│%*s│",
		hostColWidth+2, " ",
		hcaColWidth+2, " ")
	for i, serverHost := range serverHosts {
		hcas := serverHostHCAs[serverHost]
		for j, hca := range hcas {
			displayHCA := hca
			if len(hca) > valueColWidth {
				displayHCA = hca[:valueColWidth-2] + ".."
			}
			if j < len(hcas)-1 || i < len(serverHosts)-1 {
				fmt.Printf(" %-*s │", valueColWidth, displayHCA)
			} else {
				fmt.Printf(" %-*s │\n", valueColWidth, displayHCA)
			}
		}
	}

	// Print header separator
	fmt.Printf("├%s┼%s┼",
		strings.Repeat("─", hostColWidth+2),
		strings.Repeat("─", hcaColWidth+2))
	for i, serverHost := range serverHosts {
		numHCAs := len(serverHostHCAs[serverHost])
		for j := 0; j < numHCAs; j++ {
			if j < numHCAs-1 {
				fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
			} else {
				if i < len(serverHosts)-1 {
					fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
				} else {
					fmt.Printf("%s┤\n", strings.Repeat("─", valueColWidth+2))
				}
			}
		}
	}

	// Print data rows (clients)
	for clientHostIdx, clientHostName := range clientHosts {
		hcas := clientHostHCAs[clientHostName]
		for hcaIdx, clientHCA := range hcas {
			// Print client hostname (only on first HCA row)
			if hcaIdx == 0 {
				displayHost := clientHostName
				if len(clientHostName) > hostColWidth {
					displayHost = clientHostName[:hostColWidth-2] + ".."
				}
				fmt.Printf("│ %-*s │", hostColWidth, displayHost)
			} else {
				fmt.Printf("│%*s│", hostColWidth+2, " ")
			}

			// Print client HCA
			displayHCA := clientHCA
			if len(clientHCA) > hcaColWidth {
				displayHCA = clientHCA[:hcaColWidth-2] + ".."
			}
			fmt.Printf(" %-*s │", hcaColWidth, displayHCA)

			// Print latency values for all servers
			clientKey := fmt.Sprintf("%s:%s", clientHostName, clientHCA)
			for _, serverHost := range serverHosts {
				serverHcas := serverHostHCAs[serverHost]
				for _, serverHCA := range serverHcas {
					serverKey := fmt.Sprintf("%s:%s", serverHost, serverHCA)
					latency := matrix[clientKey][serverKey]
					if latency > 0 {
						fmt.Printf(" %*.2f μs │", valueColWidth-3, latency)
					} else {
						fmt.Printf(" %*s │", valueColWidth, "-")
					}
				}
			}
			fmt.Println()

			// Print row separator
			isLastHCA := hcaIdx == len(hcas)-1
			isLastHost := clientHostIdx == len(clientHosts)-1

			if isLastHost && isLastHCA {
				// Last row - bottom border
				fmt.Printf("└%s┴%s┴",
					strings.Repeat("─", hostColWidth+2),
					strings.Repeat("─", hcaColWidth+2))
				for i, serverHost := range serverHosts {
					numHCAs := len(serverHostHCAs[serverHost])
					for j := 0; j < numHCAs; j++ {
						if j < numHCAs-1 {
							fmt.Printf("%s┴", strings.Repeat("─", valueColWidth+2))
						} else {
							if i < len(serverHosts)-1 {
								fmt.Printf("%s┴", strings.Repeat("─", valueColWidth+2))
							} else {
								fmt.Printf("%s┘\n", strings.Repeat("─", valueColWidth+2))
							}
						}
					}
				}
			} else if isLastHCA {
				// End of host group - use crossing separator
				fmt.Printf("├%s┼%s┼",
					strings.Repeat("─", hostColWidth+2),
					strings.Repeat("─", hcaColWidth+2))
				for i, serverHost := range serverHosts {
					numHCAs := len(serverHostHCAs[serverHost])
					for j := 0; j < numHCAs; j++ {
						if j < numHCAs-1 {
							fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
						} else {
							if i < len(serverHosts)-1 {
								fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
							} else {
								fmt.Printf("%s┤\n", strings.Repeat("─", valueColWidth+2))
							}
						}
					}
				}
			} else {
				// Within host group - use non-crossing separator
				fmt.Printf("│%*s├%s┼",
					hostColWidth+2, " ",
					strings.Repeat("─", hcaColWidth+2))
				for i, serverHost := range serverHosts {
					numHCAs := len(serverHostHCAs[serverHost])
					for j := 0; j < numHCAs; j++ {
						if j < numHCAs-1 {
							fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
						} else {
							if i < len(serverHosts)-1 {
								fmt.Printf("%s┼", strings.Repeat("─", valueColWidth+2))
							} else {
								fmt.Printf("%s┤\n", strings.Repeat("─", valueColWidth+2))
							}
						}
					}
				}
			}
		}
	}

	// Calculate and display statistics
	displayIncastStatistics(latencyData, clientHostHCAs, serverHostHCAs)
}

// displayIncastStatistics calculates and displays statistics for incast mode
func displayIncastStatistics(latencyData []LatencyData, clientHostHCAs map[string][]string, serverHostHCAs map[string][]string) {
	if len(latencyData) == 0 {
		return
	}

	// Calculate global statistics
	var allLatencies []float64
	for _, data := range latencyData {
		allLatencies = append(allLatencies, data.AvgLatencyUs)
	}

	sort.Float64s(allLatencies)
	minLatency := allLatencies[0]
	maxLatency := allLatencies[len(allLatencies)-1]
	var sum float64
	for _, lat := range allLatencies {
		sum += lat
	}
	avgLatency := sum / float64(len(allLatencies))

	// Calculate per-server statistics
	serverStats := make(map[string][]float64)
	for _, data := range latencyData {
		serverKey := fmt.Sprintf("%s:%s", data.TargetHost, data.TargetHCA)
		serverStats[serverKey] = append(serverStats[serverKey], data.AvgLatencyUs)
	}

	// Calculate per-client statistics
	clientStats := make(map[string][]float64)
	for _, data := range latencyData {
		clientKey := fmt.Sprintf("%s:%s", data.SourceHost, data.SourceHCA)
		clientStats[clientKey] = append(clientStats[clientKey], data.AvgLatencyUs)
	}

	// Print statistics
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📈 Statistics Summary")
	fmt.Println(strings.Repeat("=", 80))

	// Global statistics
	fmt.Println("\n🌐 Global Statistics:")
	fmt.Printf("   Total measurements: %d\n", len(allLatencies))
	fmt.Printf("   Minimum latency:    %.2f μs\n", minLatency)
	fmt.Printf("   Maximum latency:    %.2f μs\n", maxLatency)
	fmt.Printf("   Average latency:    %.2f μs\n", avgLatency)

	// Per-server statistics
	fmt.Println("\n🖥️  Per-Server Average Latency:")
	var serverHosts []string
	for host := range serverHostHCAs {
		serverHosts = append(serverHosts, host)
	}
	sort.Strings(serverHosts)

	for _, host := range serverHosts {
		for _, hca := range serverHostHCAs[host] {
			serverKey := fmt.Sprintf("%s:%s", host, hca)
			if latencies, ok := serverStats[serverKey]; ok && len(latencies) > 0 {
				var sum float64
				for _, lat := range latencies {
					sum += lat
				}
				avg := sum / float64(len(latencies))
				fmt.Printf("   %-30s  %.2f μs  (%d clients)\n", serverKey, avg, len(latencies))
			}
		}
	}

	// Per-client statistics
	fmt.Println("\n💻 Per-Client Average Latency:")
	var clientHosts []string
	for host := range clientHostHCAs {
		clientHosts = append(clientHosts, host)
	}
	sort.Strings(clientHosts)

	for _, host := range clientHosts {
		for _, hca := range clientHostHCAs[host] {
			clientKey := fmt.Sprintf("%s:%s", host, hca)
			if latencies, ok := clientStats[clientKey]; ok && len(latencies) > 0 {
				var sum float64
				for _, lat := range latencies {
					sum += lat
				}
				avg := sum / float64(len(latencies))
				fmt.Printf("   %-30s  %.2f μs  (%d servers)\n", clientKey, avg, len(latencies))
			}
		}
	}

	fmt.Println(strings.Repeat("=", 80))
}
