package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	// Get config
	configFile := viper.GetString("config")
	if configFile == "" {
		configFile = "config/config.yaml"
	}

	// Use the reports directory from flag
	reportsDir := reportsPath

	// Check if reports directory exists
	if _, err := os.Stat(reportsDir); os.IsNotExist(err) {
		fmt.Printf("Reports directory not found: %s\n", reportsDir)
		return
	}

	// Collect all report data
	clientData, serverData, err := collectReportData(reportsDir)
	if err != nil {
		fmt.Printf("Error collecting report data: %v\n", err)
		return
	}

	// Display results
	displayResults(clientData, serverData)

	// Generate markdown file if requested
	if generateMD {
		err := generateMarkdownTable(clientData, serverData)
		if err != nil {
			fmt.Printf("Error generating markdown file: %v\n", err)
		} else {
			fmt.Println("\nMarkdown table generated: network_performance_analysis.md")
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
		content, err := ioutil.ReadFile(path)
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

func displayResults(clientData, serverData map[string]map[string]*DeviceData) {
	fmt.Println("=== Network Performance Analysis ===\n")

	// 计算总服务端带宽和客户端数量
	totalServerBW := calculateTotalServerBandwidth(serverData)
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

func generateMarkdownTable(clientData, serverData map[string]map[string]*DeviceData) error {
	content := "# Network Performance Analysis\n\n"

	// 计算理论带宽
	totalServerBW := calculateTotalServerBandwidth(serverData)
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

	return ioutil.WriteFile("network_performance_analysis.md", []byte(content), 0644)
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
func calculateTotalServerBandwidth(serverData map[string]map[string]*DeviceData) float64 {
	total := float64(0)
	for _, devices := range serverData {
		for _, data := range devices {
			total += data.BWSum
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

			fmt.Printf("│ %-19s │ %-8s │ %11.2f │ %12.2f │ %-15s │ %-8s │\n",
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
