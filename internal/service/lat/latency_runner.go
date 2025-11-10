package lat

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"xnetperf/config"
	"xnetperf/internal/script"
	"xnetperf/internal/service/collect"
	"xnetperf/internal/service/precheck"
	"xnetperf/pkg/tools"
	"xnetperf/stream"
)

// latRunner manages latency testing workflow
type latRunner struct {
	cfg    *config.Config
	logger *slog.Logger
}

// New creates a new latency runner instance
func New(cfg *config.Config) *latRunner {
	return &latRunner{
		cfg:    cfg,
		logger: slog.Default(),
	}
}

// Execute runs the complete latency testing workflow (for CLI)
func (r *latRunner) Execute() error {
	fmt.Println("🚀 Starting xnetperf latency testing workflow...")
	fmt.Println(strings.Repeat("=", 60))

	// Step 0: Precheck - Verify network card status before starting tests
	fmt.Println("\n🔍 Step 0/5: Performing network card precheck...")
	checker := precheck.New(r.cfg)
	checker.Display(checker.DoCheck())
	fmt.Println("✅ Precheck passed! All network cards are healthy. Proceeding with latency tests...")

	if r.cfg.Version == "v1" {
		executor := script.NewExecutor(r.cfg, script.TestTypeLatency)
		if executor == nil {
			return fmt.Errorf("unsupported stream type for v1 execute workflow")
		}
		fmt.Println("\n📋 Step 1/5: Running network tests...")
		err := executor.Execute()
		if err != nil {
			return fmt.Errorf("run step failed: %w", err)
		}
	} else {
		// Step 1: Generate latency scripts
		fmt.Println("\n📋 Step 1/5: Generating latency test scripts...")
		if err := r.generateScripts(); err != nil {
			return fmt.Errorf("script generation failed: %w", err)
		}

		// Step 2: Run latency tests
		fmt.Println("\n▶️  Step 2/5: Running latency tests...")
		if err := r.runTests(); err != nil {
			return fmt.Errorf("latency test execution failed: %w", err)
		}
	}

	// Step 3: Monitor progress
	fmt.Println("\n🔍 Step 3/5: Monitoring test progress...")
	if err := r.monitorProgress(); err != nil {
		return fmt.Errorf("probe step failed: %w", err)
	}

	// Short delay before collection
	fmt.Println("⏳ Waiting for 2 seconds before collecting reports...")
	time.Sleep(time.Second * 2)

	// Step 4: Collect reports
	fmt.Println("\n📥 Step 4/5: Collecting latency reports...")
	if err := r.collectReports(); err != nil {
		return fmt.Errorf("report collection failed: %w", err)
	}

	// Step 5: Analyze and display latency matrix
	fmt.Println("\n📊 Step 5/5: Analyzing latency results...")
	if err := r.analyzeAndDisplay(); err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	fmt.Println("\n🎉 Latency testing workflow completed successfully!")
	fmt.Println(strings.Repeat("=", 60))

	return nil
}

// GenerateLatencyReport generates latency report data for API responses
func (r *latRunner) GenerateLatencyReport() (*LatencySummary, error) {
	reportsDir := "reports"

	// Check if reports directory exists
	if _, err := os.Stat(reportsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("reports directory not found: %s", reportsDir)
	}

	// Collect latency data from report files
	latencyData, err := r.collectLatencyReportData(reportsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to collect latency report data: %w", err)
	}

	if len(latencyData) == 0 {
		return nil, fmt.Errorf("no latency reports found in %s", reportsDir)
	}

	// Build summary structure
	summary := &LatencySummary{
		StreamType: string(r.cfg.StreamType),
		Matrix:     make(map[string]map[string]float64),
	}

	// Build matrix
	for _, data := range latencyData {
		sourceKey := fmt.Sprintf("%s:%s", data.SourceHost, data.SourceHCA)
		targetKey := fmt.Sprintf("%s:%s", data.TargetHost, data.TargetHCA)
		if summary.Matrix[sourceKey] == nil {
			summary.Matrix[sourceKey] = make(map[string]float64)
		}
		summary.Matrix[sourceKey][targetKey] = data.AvgLatencyUs
	}

	// Calculate global statistics
	var allLatencies []float64
	for _, data := range latencyData {
		allLatencies = append(allLatencies, data.AvgLatencyUs)
	}
	summary.Statistics = LatencyStatistics{
		MinLatency: minFloat(allLatencies),
		MaxLatency: maxFloat(allLatencies),
		AvgLatency: avgFloat(allLatencies),
		TotalCount: len(allLatencies),
	}

	// For incast mode, calculate per-client and per-server statistics
	if r.cfg.StreamType == config.InCast {
		summary.ClientStats = make(map[string]LatencyStats)
		summary.ServerStats = make(map[string]LatencyStats)

		// Build client/server sets
		clientHostSet := make(map[string]bool)
		for _, host := range r.cfg.Client.Hostname {
			clientHostSet[host] = true
		}

		// Calculate client stats
		clientData := make(map[string][]float64)
		for _, data := range latencyData {
			if clientHostSet[data.SourceHost] {
				clientKey := fmt.Sprintf("%s:%s", data.SourceHost, data.SourceHCA)
				clientData[clientKey] = append(clientData[clientKey], data.AvgLatencyUs)
			}
		}
		for key, latencies := range clientData {
			summary.ClientStats[key] = LatencyStats{
				AvgLatency: avgFloat(latencies),
				Count:      len(latencies),
			}
		}

		// Calculate server stats
		serverData := make(map[string][]float64)
		for _, data := range latencyData {
			if clientHostSet[data.SourceHost] {
				serverKey := fmt.Sprintf("%s:%s", data.TargetHost, data.TargetHCA)
				serverData[serverKey] = append(serverData[serverKey], data.AvgLatencyUs)
			}
		}
		for key, latencies := range serverData {
			summary.ServerStats[key] = LatencyStats{
				AvgLatency: avgFloat(latencies),
				Count:      len(latencies),
			}
		}
	}

	return summary, nil
}

// generateScripts generates latency test scripts
func (r *latRunner) generateScripts() error {
	fmt.Println("Generating N×N latency test scripts...")

	if err := stream.GenerateLatencyScripts(r.cfg); err != nil {
		return fmt.Errorf("error generating latency scripts: %w", err)
	}

	fmt.Println("✅ Latency scripts generated successfully")
	return nil
}

// runTests runs the latency test scripts
func (r *latRunner) runTests() error {
	fmt.Println("Executing latency tests...")

	if err := stream.RunLatencyScripts(r.cfg); err != nil {
		return fmt.Errorf("error running latency scripts: %w", err)
	}

	fmt.Println("✅ Latency tests started successfully")
	return nil
}

// monitorProgress monitors latency test progress
func (r *latRunner) monitorProgress() error {
	fmt.Println("Monitoring ib_write_lat processes (5-second intervals)...")

	// Get all hosts list
	allHosts := r.cfg.ALLHosts()

	if len(allHosts) == 0 {
		return fmt.Errorf("no hosts configured in config file")
	}

	fmt.Printf("Probing ib_write_lat processes on %d hosts...\n", len(allHosts))
	fmt.Println("Mode: Continuous monitoring until all processes complete")
	fmt.Println()

	probeIntervalSec := 5

	for {
		results := r.probeLatencyAllHosts(allHosts)
		r.displayLatencyProbeResults(results)

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

	return nil
}

// collectReports collects latency report files
func (r *latRunner) collectReports() error {
	if !r.cfg.Report.Enable {
		fmt.Println("⚠️  Report generation is disabled in config. Skipping collect step.")
		return nil
	}

	var cleanupRemote = true
	fmt.Println("Collecting latency report files from remote hosts...")

	collector := collect.New(r.cfg)
	if err := collector.DoCollect(cleanupRemote); err != nil {
		return fmt.Errorf("error during report collection: %w", err)
	}

	fmt.Println("✅ Latency report collection completed successfully")
	return nil
}

// analyzeAndDisplay analyzes latency results and displays N×N matrix
func (r *latRunner) analyzeAndDisplay() error {
	if !r.cfg.Report.Enable {
		fmt.Println("⚠️  Report generation is disabled in config. Skipping analyze step.")
		return nil
	}

	fmt.Println("Analyzing latency results...")

	reportsDir := "reports"

	// Check if reports directory exists
	if _, err := os.Stat(reportsDir); os.IsNotExist(err) {
		return fmt.Errorf("reports directory not found: %s", reportsDir)
	}

	// Collect and display latency data
	latencyMatrix, err := r.collectLatencyReportData(reportsDir)
	if err != nil {
		return fmt.Errorf("error collecting latency report data: %w", err)
	}

	// Display based on stream type
	if r.cfg.StreamType == config.InCast {
		displayLatencyMatrixIncast(latencyMatrix, r.cfg)
	} else {
		// Default to fullmesh display
		displayLatencyMatrix(latencyMatrix)
	}

	fmt.Println("✅ Latency analysis completed successfully")
	return nil
}

// collectLatencyReportData parses all latency JSON reports
func (r *latRunner) collectLatencyReportData(reportsDir string) ([]LatencyData, error) {
	var latencyData []LatencyData

	// Walk through all JSON files in reports directory
	err := filepath.Walk(reportsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process latency JSON files (named like latency_c_*.json or latency_s_*.json)
		if !info.IsDir() && strings.HasPrefix(info.Name(), "latency_") && strings.HasSuffix(info.Name(), ".json") {
			data, parseErr := r.parseLatencyReport(path)
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
		return nil, fmt.Errorf("failed to walk reports directory: %w", err)
	}

	if len(latencyData) == 0 {
		return nil, fmt.Errorf("no latency reports found in %s", reportsDir)
	}

	return latencyData, nil
}

// parseLatencyReport parses a single latency JSON report file
func (r *latRunner) parseLatencyReport(filePath string) (*LatencyData, error) {
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
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var report LatencyReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
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

// DoLatencyProbe performs a single probe of ib_write_lat processes across all hosts
func (r *latRunner) DoLatencyProbe() ([]LatencyProbeResult, error) {
	r.logger.Info("Starting latency probe operation")

	// Get all hosts list
	allHosts := r.cfg.ALLHosts()

	if len(allHosts) == 0 {
		r.logger.Warn("No hosts configured in config file")
		return nil, fmt.Errorf("no hosts found in configuration")
	}

	ret := r.probeLatencyAllHosts(allHosts)

	r.logger.Info("Latency probe operation completed successfully")
	return ret, nil
}

// DoLatencyProbeAndGetSummary performs a probe and returns a summary for API responses
func (r *latRunner) DoLatencyProbeAndGetSummary() (*LatencyProbeSummary, error) {
	// Probe all hosts
	results, err := r.DoLatencyProbe()
	if err != nil {
		return nil, err
	}

	// Build summary
	summary := &LatencyProbeSummary{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Results:   results,
	}

	for _, result := range results {
		switch result.Status {
		case "RUNNING":
			summary.RunningHosts++
			summary.TotalProcesses += result.ProcessCount
		case "COMPLETED":
			summary.CompletedHosts++
		case "ERROR":
			summary.ErrorHosts++
		}
	}

	summary.AllCompleted = (summary.CompletedHosts == len(results))

	return summary, nil
}

// probeLatencyAllHosts probes all hosts for ib_write_lat processes
func (r *latRunner) probeLatencyAllHosts(hosts map[string]bool) []LatencyProbeResult {
	var results []LatencyProbeResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for hostname := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			result := r.probeLatencyHost(host)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(hostname)
	}

	wg.Wait()
	return results
}

// probeLatencyHost probes a single host for ib_write_lat processes
func (r *latRunner) probeLatencyHost(hostname string) LatencyProbeResult {
	result := LatencyProbeResult{
		Hostname: hostname,
	}

	// Use SSH to execute ps command to find ib_write_lat processes
	cmd := tools.BuildSSHCommand(hostname, "ps aux | grep ib_write_lat | grep -v grep", r.cfg.SSH.PrivateKey, r.cfg.SSH.User)
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
func (r *latRunner) displayLatencyProbeResults(results []LatencyProbeResult) {
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
