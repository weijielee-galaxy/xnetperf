package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"xnetperf/config"

	"github.com/spf13/cobra"
)

// Report represents the structure of a JSON report file
type Report struct {
	TestInfo struct {
		Test   string `json:"test"`
		Device string `json:"Device"`
	} `json:"test_info"`
	Results struct {
		BWAverage float64 `json:"BW_average"`
	} `json:"results"`
}

// DeviceData represents aggregated data for a device
type DeviceData struct {
	Hostname string
	Device   string
	BWSum    float64
	Count    int
	IsClient bool
}

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
	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		os.Exit(1)
	}

	// Use the reports directory from flag
	reportsDir := reportsPath

	// Check if reports directory exists
	if _, err := os.Stat(reportsDir); os.IsNotExist(err) {
		fmt.Printf("Reports directory not found: %s\n", reportsDir)
		return
	}

	// Handle different stream types with separate functions
	switch cfg.StreamType {
	case config.P2P:
		runP2PAnalyze(reportsDir, cfg)
	default:
		// Handle fullmesh and incast with existing logic
		runTraditionalAnalyze(reportsDir, cfg)
	}
}

// runTraditionalAnalyze handles fullmesh and incast analysis with existing logic
func runTraditionalAnalyze(reportsDir string, cfg *config.Config) {
	// Collect all report data using existing function
	clientData, serverData, err := collectReportData(reportsDir)
	if err != nil {
		fmt.Printf("Error collecting report data: %v\n", err)
		return
	}

	// Display results using existing function
	displayResults(clientData, serverData, cfg.Speed)

	// Generate markdown file if requested
	if generateMD {
		err := generateMarkdownTable(clientData, serverData, cfg.Speed)
		if err != nil {
			fmt.Printf("Error generating markdown file: %v\n", err)
		} else {
			fmt.Println("\nMarkdown table generated: network_performance_analysis.md")
		}
	}
}

// runP2PAnalyze handles P2P-specific analysis
func runP2PAnalyze(reportsDir string, cfg *config.Config) {
	// Collect P2P report data
	p2pData, err := collectP2PReportData(reportsDir)
	if err != nil {
		fmt.Printf("Error collecting P2P report data: %v\n", err)
		return
	}

	// Display P2P results
	displayP2PResults(p2pData)

	// Generate P2P markdown file if requested
	if generateMD {
		err := generateP2PMarkdownTable(p2pData)
		if err != nil {
			fmt.Printf("Error generating P2P markdown file: %v\n", err)
		} else {
			fmt.Println("\nP2P Markdown table generated: p2p_performance_analysis.md")
		}
	}
}

func collectReportData(reportsDir string) (map[string]map[string]*DeviceData, map[string]map[string]*DeviceData, error) {
	clientData := make(map[string]map[string]*DeviceData)
	serverData := make(map[string]map[string]*DeviceData)

	err := filepath.Walk(reportsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}

		// Parse filename to extract information
		filename := info.Name()
		parts := strings.Split(filename, "_")
		if len(parts) < 5 {
			return nil
		}

		isClient := strings.HasPrefix(filename, "report_c_")
		hostname := parts[2]
		device := parts[3] + "_" + parts[4] // Reconstruct device name like mlx5_0

		// Read and parse JSON file
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("Error reading file %s: %v\n", path, err)
			return nil
		}

		var report Report
		if err := json.Unmarshal(content, &report); err != nil {
			fmt.Printf("Error parsing JSON file %s: %v\n", path, err)
			return nil
		}

		// Choose the appropriate data map
		var dataMap map[string]map[string]*DeviceData
		if isClient {
			dataMap = clientData
		} else {
			dataMap = serverData
		}

		// Initialize hostname map if it doesn't exist
		if dataMap[hostname] == nil {
			dataMap[hostname] = make(map[string]*DeviceData)
		}

		// Initialize or update device data
		if dataMap[hostname][device] == nil {
			dataMap[hostname][device] = &DeviceData{
				Hostname: hostname,
				Device:   device,
				BWSum:    0,
				Count:    0,
				IsClient: isClient,
			}
		}

		dataMap[hostname][device].BWSum += report.Results.BWAverage
		dataMap[hostname][device].Count++

		return nil
	})

	return clientData, serverData, err
}

func displayResults(clientData, serverData map[string]map[string]*DeviceData, specSpeed float64) {
	fmt.Println("=== Network Performance Analysis ===")

	// 计算总服务端带宽和客户端数量
	totalServerBW := calculateTotalServerBandwidth(serverData, specSpeed)
	clientCount := calculateClientCount(clientData)
	theoreticalBWPerClient := float64(0)
	if clientCount > 0 {
		theoreticalBWPerClient = totalServerBW / float64(clientCount)
	}

	// Display client data with enhanced table
	fmt.Println("CLIENT DATA (TX)")
	fmt.Println("┌─────────────────────┬──────────┬─────────────┬──────────────┬─────────────────┬──────────┐")
	fmt.Println("│ Hostname            │ Device   │ TX (Gbps)   │ SPEC (Gbps)  │ DELTA           │ Status   │")
	fmt.Println("├─────────────────────┼──────────┼─────────────┼──────────────┼─────────────────┼──────────┤")

	displayEnhancedClientTable(clientData, theoreticalBWPerClient)
	fmt.Println("└─────────────────────┴──────────┴─────────────┴──────────────┴─────────────────┴──────────┘")

	fmt.Printf("\nTheoretical BW per client: %.2f Gbps (Total server BW: %.2f Gbps ÷ %d clients)\n",
		theoreticalBWPerClient, totalServerBW, clientCount)

	fmt.Println()

	// Display server data
	fmt.Println("SERVER DATA (RX)")
	fmt.Println("┌─────────────────────┬──────────┬─────────────┐")
	fmt.Println("│ Hostname            │ Device   │ RX (Gbps)   │")
	fmt.Println("├─────────────────────┼──────────┼─────────────┤")

	displayDataTable(serverData, true)
	fmt.Println("└─────────────────────┴──────────┴─────────────┘")
}

func displayDataTable(dataMap map[string]map[string]*DeviceData, isServer bool) {
	// Get sorted hostnames
	var hostnames []string
	for hostname := range dataMap {
		hostnames = append(hostnames, hostname)
	}
	sort.Strings(hostnames)

	for i, hostname := range hostnames {
		devices := dataMap[hostname]

		// Get sorted devices
		var deviceNames []string
		for device := range devices {
			deviceNames = append(deviceNames, device)
		}
		sort.Strings(deviceNames)

		for j, device := range deviceNames {
			data := devices[device]
			total := data.BWSum // 使用累加值而不是平均值

			// Format hostname (only show for first device of each host)
			hostnameStr := ""
			if j == 0 {
				hostnameStr = hostname
			}

			fmt.Printf("│ %-19s │ %-8s │ %11.2f │\n",
				hostnameStr, device, total)
		}

		// Add separator between different hostnames (except for the last one)
		if i < len(hostnames)-1 && len(dataMap[hostname]) > 0 {
			fmt.Println("├─────────────────────┼──────────┼─────────────┤")
		}
	}
}

func generateMarkdownTable(clientData, serverData map[string]map[string]*DeviceData, specSpeed float64) error {
	content := "# Network Performance Analysis\n\n"

	// 计算理论带宽
	totalServerBW := calculateTotalServerBandwidth(serverData, specSpeed)
	clientCount := calculateClientCount(clientData)
	theoreticalBWPerClient := float64(0)
	if clientCount > 0 {
		theoreticalBWPerClient = totalServerBW / float64(clientCount)
	}

	// Client data table with enhanced columns
	content += "## Client Data (TX)\n\n"
	content += fmt.Sprintf("Theoretical BW per client: %.2f Gbps (Total server BW: %.2f Gbps ÷ %d clients)\n\n",
		theoreticalBWPerClient, totalServerBW, clientCount)
	content += "| Hostname | Device | TX (Gbps) | SPEC (Gbps) | DELTA | Status |\n"
	content += "|----------|--------|-----------|-------------|-------|--------|\n"

	content += generateEnhancedMarkdownClientContent(clientData, theoreticalBWPerClient)
	content += "\n"

	// Server data table
	content += "## Server Data (RX)\n\n"
	content += "| Hostname | Device | RX (Gbps) |\n"
	content += "|----------|--------|----------|\n"

	content += generateMarkdownTableContent(serverData)

	return os.WriteFile("network_performance_analysis.md", []byte(content), 0644)
}

func generateMarkdownTableContent(dataMap map[string]map[string]*DeviceData) string {
	var content strings.Builder

	// Get sorted hostnames
	var hostnames []string
	for hostname := range dataMap {
		hostnames = append(hostnames, hostname)
	}
	sort.Strings(hostnames)

	for _, hostname := range hostnames {
		devices := dataMap[hostname]

		// Get sorted devices
		var deviceNames []string
		for device := range devices {
			deviceNames = append(deviceNames, device)
		}
		sort.Strings(deviceNames)

		for j, device := range deviceNames {
			data := devices[device]
			total := data.BWSum // 使用累加值而不是平均值

			// Format hostname (only show for first device of each host)
			hostnameStr := ""
			if j == 0 {
				hostnameStr = hostname
			}

			content.WriteString(fmt.Sprintf("| %s | %s | %.2f |\n",
				hostnameStr, device, total))
		}
	}

	return content.String()
}

// calculateTotalServerBandwidth 计算总服务端带宽
func calculateTotalServerBandwidth(serverData map[string]map[string]*DeviceData, specSpeed float64) float64 {
	total := float64(0)
	for _, devices := range serverData {
		for range devices {
			total += specSpeed
		}
	}
	return total
}

// calculateClientCount 计算客户端数量（host+device组合数）
func calculateClientCount(clientData map[string]map[string]*DeviceData) int {
	count := 0
	for _, devices := range clientData {
		count += len(devices)
	}
	return count
}

// displayEnhancedClientTable 显示增强的客户端表格
func displayEnhancedClientTable(clientData map[string]map[string]*DeviceData, theoreticalBW float64) {
	// Get sorted hostnames
	var hostnames []string
	for hostname := range clientData {
		hostnames = append(hostnames, hostname)
	}
	sort.Strings(hostnames)

	for i, hostname := range hostnames {
		devices := clientData[hostname]

		// Get sorted devices
		var deviceNames []string
		for device := range devices {
			deviceNames = append(deviceNames, device)
		}
		sort.Strings(deviceNames)

		for j, device := range deviceNames {
			data := devices[device]
			actualBW := data.BWSum
			delta := actualBW - theoreticalBW
			deltaPercent := float64(0)
			if theoreticalBW > 0 {
				deltaPercent = (delta / theoreticalBW) * 100
			}

			// 格式化DELTA列
			deltaStr := fmt.Sprintf("%.1f(%.0f%%)", delta, deltaPercent)

			// 计算状态
			status := "OK"
			if abs(deltaPercent) > 20 {
				status = "NOT OK"
			}

			// Format hostname (only show for first device of each host)
			hostnameStr := ""
			if j == 0 {
				hostnameStr = hostname
			}

			fmt.Printf("│ %-19s │ %-8s │ %11.2f │ %12.2f │ %15s │ %-8s │\n",
				hostnameStr, device, actualBW, theoreticalBW, deltaStr, status)
		}

		// Add separator between different hostnames (except for the last one)
		if i < len(hostnames)-1 && len(clientData[hostname]) > 0 {
			fmt.Println("├─────────────────────┼──────────┼─────────────┼──────────────┼─────────────────┼──────────┤")
		}
	}
}

// abs 返回浮点数的绝对值
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// generateEnhancedMarkdownClientContent 生成增强的客户端Markdown表格内容
func generateEnhancedMarkdownClientContent(clientData map[string]map[string]*DeviceData, theoreticalBW float64) string {
	var content strings.Builder

	// Get sorted hostnames
	var hostnames []string
	for hostname := range clientData {
		hostnames = append(hostnames, hostname)
	}
	sort.Strings(hostnames)

	for _, hostname := range hostnames {
		devices := clientData[hostname]

		// Get sorted devices
		var deviceNames []string
		for device := range devices {
			deviceNames = append(deviceNames, device)
		}
		sort.Strings(deviceNames)

		for j, device := range deviceNames {
			data := devices[device]
			actualBW := data.BWSum
			delta := actualBW - theoreticalBW
			deltaPercent := float64(0)
			if theoreticalBW > 0 {
				deltaPercent = (delta / theoreticalBW) * 100
			}

			// 格式化DELTA列
			deltaStr := fmt.Sprintf("%.1f(%.0f%%)", delta, deltaPercent)

			// 计算状态
			status := "OK"
			if abs(deltaPercent) > 20 {
				status = "NOT OK"
			}

			// Format hostname (only show for first device of each host)
			hostnameStr := ""
			if j == 0 {
				hostnameStr = hostname
			}

			content.WriteString(fmt.Sprintf("| %s | %s | %.2f | %.2f | %s | %s |\n",
				hostnameStr, device, actualBW, theoreticalBW, deltaStr, status))
		}
	}

	return content.String()
}

// P2PDeviceData represents aggregated data for a P2P device
type P2PDeviceData struct {
	Hostname string
	Device   string
	BWSum    float64
	Count    int
}

// collectP2PReportData collects report data specifically for P2P mode
func collectP2PReportData(reportsDir string) (map[string]map[string]*P2PDeviceData, error) {
	p2pData := make(map[string]map[string]*P2PDeviceData)

	err := filepath.Walk(reportsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}

		// Parse P2P filename format: report_hostname_device_port.json
		filename := info.Name()

		// Skip traditional client/server report files
		if strings.HasPrefix(filename, "report_c_") || strings.HasPrefix(filename, "report_s_") {
			return nil
		}

		// Must start with "report_" for P2P format
		if !strings.HasPrefix(filename, "report_") {
			return nil
		}

		parts := strings.Split(filename, "_")
		if len(parts) < 4 {
			return nil // Not enough parts for P2P format
		}

		hostname := parts[1]
		device := parts[2] + "_" + parts[3] // Reconstruct device name like mlx5_0

		// Read and parse JSON file
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("Error reading P2P file %s: %v\n", path, err)
			return nil
		}

		var report Report
		if err := json.Unmarshal(content, &report); err != nil {
			fmt.Printf("Error parsing P2P JSON file %s: %v\n", path, err)
			return nil
		}

		// Initialize hostname map if it doesn't exist
		if p2pData[hostname] == nil {
			p2pData[hostname] = make(map[string]*P2PDeviceData)
		}

		// Initialize or update device data
		if p2pData[hostname][device] == nil {
			p2pData[hostname][device] = &P2PDeviceData{
				Hostname: hostname,
				Device:   device,
				BWSum:    0,
				Count:    0,
			}
		}

		p2pData[hostname][device].BWSum += report.Results.BWAverage
		p2pData[hostname][device].Count++

		return nil
	})

	return p2pData, err
}

// displayP2PResults displays results for P2P mode
func displayP2PResults(p2pData map[string]map[string]*P2PDeviceData) {
	fmt.Println("=== P2P Performance Analysis ===")
	fmt.Println("┌─────────────────────┬──────────┬─────────────┐")
	fmt.Println("│ Hostname            │ Device   │ Speed (Gbps)│")
	fmt.Println("├─────────────────────┼──────────┼─────────────┤")

	// Get sorted hostnames
	var hostnames []string
	for hostname := range p2pData {
		hostnames = append(hostnames, hostname)
	}
	sort.Strings(hostnames)

	for i, hostname := range hostnames {
		devices := p2pData[hostname]

		// Get sorted devices
		var deviceNames []string
		for device := range devices {
			deviceNames = append(deviceNames, device)
		}
		sort.Strings(deviceNames)

		for j, device := range deviceNames {
			data := devices[device]
			avgSpeed := data.BWSum / float64(data.Count)

			// Format hostname (only show for first device of each host)
			hostnameStr := ""
			if j == 0 {
				hostnameStr = hostname
			}

			fmt.Printf("│ %-19s │ %-8s │ %11.2f │\n",
				hostnameStr, device, avgSpeed)

			// Add separator between different hosts (except for the last host)
			if j == len(deviceNames)-1 && i < len(hostnames)-1 {
				fmt.Println("├─────────────────────┼──────────┼─────────────┤")
			}
		}
	}

	fmt.Println("└─────────────────────┴──────────┴─────────────┘")

	// Calculate and display summary
	totalPairs := 0
	totalSpeed := 0.0
	for _, devices := range p2pData {
		for _, data := range devices {
			totalPairs++
			totalSpeed += data.BWSum / float64(data.Count)
		}
	}

	if totalPairs > 0 {
		fmt.Printf("\nP2P Summary: %d connection pairs, Average speed: %.2f Gbps\n",
			totalPairs, totalSpeed/float64(totalPairs))
	}
}

// generateP2PMarkdownTable generates markdown table for P2P results
func generateP2PMarkdownTable(p2pData map[string]map[string]*P2PDeviceData) error {
	var content strings.Builder

	content.WriteString("# P2P Performance Analysis Report\n\n")
	content.WriteString("| Hostname | Device | Speed (Gbps) |\n")
	content.WriteString("|----------|--------|-------------|\n")

	// Get sorted hostnames
	var hostnames []string
	for hostname := range p2pData {
		hostnames = append(hostnames, hostname)
	}
	sort.Strings(hostnames)

	for _, hostname := range hostnames {
		devices := p2pData[hostname]

		// Get sorted devices
		var deviceNames []string
		for device := range devices {
			deviceNames = append(deviceNames, device)
		}
		sort.Strings(deviceNames)

		for j, device := range deviceNames {
			data := devices[device]
			avgSpeed := data.BWSum / float64(data.Count)

			// Format hostname (only show for first device of each host)
			hostnameStr := ""
			if j == 0 {
				hostnameStr = hostname
			}

			content.WriteString(fmt.Sprintf("| %s | %s | %.2f |\n",
				hostnameStr, device, avgSpeed))
		}
	}

	// Add summary
	totalPairs := 0
	totalSpeed := 0.0
	for _, devices := range p2pData {
		for _, data := range devices {
			totalPairs++
			totalSpeed += data.BWSum / float64(data.Count)
		}
	}

	if totalPairs > 0 {
		content.WriteString(fmt.Sprintf("\n## Summary\n\n"))
		content.WriteString(fmt.Sprintf("- Total P2P pairs: %d\n", totalPairs))
		content.WriteString(fmt.Sprintf("- Average speed: %.2f Gbps\n", totalSpeed/float64(totalPairs)))
	}

	// Write to file
	filename := "p2p_performance_analysis.md"
	err := os.WriteFile(filename, []byte(content.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write P2P markdown file: %w", err)
	}

	return nil
}
