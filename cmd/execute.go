package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"xnetperf/config"
	"xnetperf/stream"

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

func init() {
	// No additional flags needed - uses global config flag
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

	// Clean up old reports on remote hosts using the same logic as run.go
	if cfg.Report.Enable {
		cleanupRemoteReportFilesForExecute(cfg)
	}

	// Generate and run scripts
	switch cfg.StreamType {
	case config.FullMesh:
		stream.GenerateFullMeshScript(cfg)
	case config.InCast:
		stream.GenerateIncastScripts(cfg)
	default:
		fmt.Printf("❌ Invalid stream_type '%s' in config.\n", cfg.StreamType)
		return false
	}

	stream.DistributeAndRunScripts(cfg)
	fmt.Println("✅ Network tests started successfully")
	return true
}

// executeProbeStep monitors the test progress using probe logic
func executeProbeStep(cfg *config.Config) bool {
	fmt.Println("Monitoring ib_write_bw processes (5-second intervals)...")

	// Get all unique hostnames
	allHosts := make(map[string]bool)
	for _, host := range cfg.Server.Hostname {
		allHosts[host] = true
	}
	for _, host := range cfg.Client.Hostname {
		allHosts[host] = true
	}

	probeInterval := 5 // 5 seconds
	fmt.Printf("Monitoring %d hosts with %d-second intervals...\n", len(allHosts), probeInterval)
	fmt.Println()

	for {
		results := probeAllHostsForExecute(allHosts)
		displayProbeResultsForExecute(results)

		// Check if all processes have completed
		allCompleted := true
		for _, result := range results {
			if result.ProcessCount > 0 {
				allCompleted = false
				break
			}
		}

		if allCompleted {
			fmt.Println("✅ All ib_write_bw processes have completed!")
			break
		}

		// Wait for next probe
		fmt.Printf("Waiting %d seconds for next probe...\n\n", probeInterval)
		time.Sleep(time.Duration(probeInterval) * time.Second)
	}

	return true
}

// executeCollectStep collects report files with cleanup
func executeCollectStep(cfg *config.Config) bool {
	if !cfg.Report.Enable {
		fmt.Println("⚠️  Report generation is disabled in config. Skipping collect step.")
		return true
	}

	fmt.Println("Collecting report files from remote hosts...")

	// Use the same logic as collectReports but inline
	reportsDir := "reports"
	err := os.MkdirAll(reportsDir, 0755)
	if err != nil {
		fmt.Printf("❌ Error creating reports directory: %v\n", err)
		return false
	}

	// Get all hosts
	allHosts := make(map[string]bool)
	for _, host := range cfg.Server.Hostname {
		allHosts[host] = true
	}
	for _, host := range cfg.Client.Hostname {
		allHosts[host] = true
	}

	// Clean up existing local files first
	cleanupLocalFilesForExecute(reportsDir, allHosts)

	var wg sync.WaitGroup
	fmt.Printf("Collecting reports from %d hosts...\n", len(allHosts))

	for hostname := range allHosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			collectFromHostForExecute(host, cfg.Report.Dir, reportsDir, true) // Enable cleanup
		}(hostname)
	}

	wg.Wait()

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

	// Call the analyze function with simulated command and args
	// We'll create a temporary cobra command to pass to runAnalyze
	tempCmd := &cobra.Command{}
	tempCmd.Flag("markdown").Changed = false
	tempCmd.Flag("reports-dir").Changed = false

	// Save current global values
	originalGenerateMD := generateMD
	originalReportsPath := reportsPath

	// Set values for execute
	generateMD = false
	reportsPath = "reports"

	// Call the analyze function
	runAnalyze(tempCmd, []string{})

	// Restore original values
	generateMD = originalGenerateMD
	reportsPath = originalReportsPath

	fmt.Println("✅ Analysis completed successfully")
	return true
}

// Helper functions for execute workflow

func cleanupRemoteReportFilesForExecute(cfg *config.Config) {
	fmt.Println("Cleaning up old report files on remote hosts before starting tests...")

	// Get all hosts
	allHosts := make(map[string]bool)
	for _, host := range cfg.Server.Hostname {
		allHosts[host] = true
	}
	for _, host := range cfg.Client.Hostname {
		allHosts[host] = true
	}

	var wg sync.WaitGroup

	for hostname := range allHosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()

			rmCmd := fmt.Sprintf("rm -f %s/*%s*.json", cfg.Report.Dir, host)
			cmd := exec.Command("ssh", host, rmCmd)

			output, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Printf("   [WARNING] ⚠️  %s: Failed to cleanup old reports: %v\n", host, err)
				if len(output) > 0 {
					fmt.Printf("   [WARNING] ⚠️  %s: SSH output: %s\n", host, string(output))
				}
			} else {
				fmt.Printf("   [CLEANUP] 🧹 %s: Old report files cleaned\n", host)
			}
		}(hostname)
	}

	wg.Wait()
	fmt.Println()
}

func cleanupLocalFilesForExecute(reportsDir string, hosts map[string]bool) {
	fmt.Printf("Cleaning up existing local report files...\n")

	for hostname := range hosts {
		hostDir := filepath.Join(reportsDir, hostname)

		if _, err := os.Stat(hostDir); os.IsNotExist(err) {
			continue
		}

		pattern := filepath.Join(hostDir, "*.json")
		files, err := filepath.Glob(pattern)
		if err != nil {
			fmt.Printf("   [WARNING] ⚠️  %s: Error finding local files: %v\n", hostname, err)
			continue
		}

		if len(files) > 0 {
			for _, file := range files {
				if err := os.Remove(file); err != nil {
					fmt.Printf("   [WARNING] ⚠️  %s: Failed to remove %s: %v\n", hostname, file, err)
				}
			}
			fmt.Printf("   [CLEANUP] 🧹 %s: Removed %d existing local files\n", hostname, len(files))
		}
	}
	fmt.Println()
}

func collectFromHostForExecute(hostname, remoteDir, localBaseDir string, cleanup bool) {
	hostDir := filepath.Join(localBaseDir, hostname)
	err := os.MkdirAll(hostDir, 0755)
	if err != nil {
		fmt.Printf("❌ Error creating directory for host %s: %v\n", hostname, err)
		return
	}

	fmt.Printf("-> Collecting reports from %s...\n", hostname)

	// Use scp to collect files matching hostname pattern
	scpCmd := fmt.Sprintf("%s/*%s*.json", remoteDir, hostname)
	cmd := exec.Command("scp", fmt.Sprintf("%s:%s", hostname, scpCmd), hostDir+"/")

	output, err := cmd.CombinedOutput()
	if err != nil {
		if string(output) != "" {
			fmt.Printf("   [WARNING] ⚠️  %s: %s\n", hostname, string(output))
		} else {
			fmt.Printf("   [WARNING] ⚠️  %s: No report files found or scp failed: %v\n", hostname, err)
		}
		return
	}

	files, err := filepath.Glob(filepath.Join(hostDir, "*.json"))
	if err != nil {
		fmt.Printf("   [ERROR] ❌ %s: Error counting files: %v\n", hostname, err)
		return
	}

	if len(files) > 0 {
		fmt.Printf("   [SUCCESS] ✅ %s: Collected %d report files\n", hostname, len(files))

		if cleanup {
			cleanupRemoteFilesForExecute(hostname, remoteDir)
		}
	} else {
		fmt.Printf("   [INFO] ℹ️  %s: No report files found\n", hostname)
	}
}

func cleanupRemoteFilesForExecute(hostname, remoteDir string) {
	fmt.Printf("   [CLEANUP] 🧹 %s: Cleaning up remote report files...\n", hostname)

	checkCmd := fmt.Sprintf("ls %s/*%s*.json 2>/dev/null | wc -l", remoteDir, hostname)
	checkExec := exec.Command("ssh", hostname, checkCmd)

	checkOutput, err := checkExec.CombinedOutput()
	if err != nil {
		fmt.Printf("   [WARNING] ⚠️  %s: Failed to check remote files: %v\n", hostname, err)
		return
	}

	if string(checkOutput) == "0\n" {
		fmt.Printf("   [CLEANUP] ℹ️  %s: No remote files to cleanup\n", hostname)
		return
	}

	rmCmd := fmt.Sprintf("rm -f %s/*%s*.json", remoteDir, hostname)
	cmd := exec.Command("ssh", hostname, rmCmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("   [WARNING] ⚠️  %s: Failed to cleanup remote files: %v\n", hostname, err)
		if len(output) > 0 {
			fmt.Printf("   [WARNING] ⚠️  %s: SSH output: %s\n", hostname, string(output))
		}
		return
	}

	verifyCmd := fmt.Sprintf("ls %s/*%s*.json 2>/dev/null | wc -l", remoteDir, hostname)
	verifyExec := exec.Command("ssh", hostname, verifyCmd)

	verifyOutput, err := verifyExec.CombinedOutput()
	if err == nil && string(verifyOutput) == "0\n" {
		fmt.Printf("   [CLEANUP] ✅ %s: Remote files cleaned up successfully\n", hostname)
	} else {
		fmt.Printf("   [WARNING] ⚠️  %s: Cleanup verification failed\n", hostname)
	}
}

// Probe-related helper functions for execute workflow
type ExecuteProbeResult struct {
	Hostname     string
	ProcessCount int
	Processes    []string
	Error        string
	Status       string
}

func probeAllHostsForExecute(hosts map[string]bool) []ExecuteProbeResult {
	var results []ExecuteProbeResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for hostname := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			result := probeHostForExecute(host)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(hostname)
	}

	wg.Wait()
	return results
}

func probeHostForExecute(hostname string) ExecuteProbeResult {
	result := ExecuteProbeResult{
		Hostname: hostname,
	}

	// Use SSH to check for ib_write_bw processes
	cmd := exec.Command("ssh", hostname, "ps aux | grep ib_write_bw | grep -v grep")
	output, err := cmd.CombinedOutput()

	if err != nil {
		if strings.Contains(string(output), "") && cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
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
		if line != "" && strings.Contains(line, "ib_write_bw") {
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

func displayProbeResultsForExecute(results []ExecuteProbeResult) {
	fmt.Printf("=== Probe Results (%s) ===\n", time.Now().Format("15:04:05"))
	fmt.Println("┌─────────────────────┬─────────────┬──────────────┬─────────────┐")
	fmt.Println("│ Hostname            │ Status      │ Process Count│ Details     │")
	fmt.Println("├─────────────────────┼─────────────┼──────────────┼─────────────┤")

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

		fmt.Printf("│ %-19s │ %-11s │ %12d │ %-11s │\n",
			result.Hostname, statusIcon, result.ProcessCount, details)

		if result.Error != "" {
			fmt.Printf("│ %-19s │ %-11s │ %12s │ %-11s │\n",
				"", "Error:", "", result.Error)
		}
	}

	fmt.Println("└─────────────────────┴─────────────┴──────────────┴─────────────┘")

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
