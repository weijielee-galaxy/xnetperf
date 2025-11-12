package v0

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"xnetperf/config"
	"xnetperf/internal/tools"
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
	Hostname     string
	Device       string
	SerialNumber string
	BWSum        float64
	Count        int
	IsClient     bool
}

func ExecAnalyzeCommand(cfg *config.Config, reportsPath string, generateMD bool) {
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
		runP2PAnalyze(reportsDir, cfg, generateMD)
	default:
		// Handle fullmesh and incast with existing logic
		runTraditionalAnalyze(reportsDir, cfg, generateMD)
	}
}

// runTraditionalAnalyze handles fullmesh and incast analysis with existing logic
func runTraditionalAnalyze(reportsDir string, cfg *config.Config, generateMD bool) {
	// Collect all report data using existing function
	clientData, serverData, err := collectReportData(reportsDir, cfg)
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
func runP2PAnalyze(reportsDir string, cfg *config.Config, generateMD bool) {
	// Collect P2P report data
	p2pData, err := collectP2PReportData(reportsDir, cfg.SSH.PrivateKey, cfg.SSH.User)
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

// getSerialNumberForHost 获取指定主机的序列号
func getSerialNumberForHost(hostname string, sshKeyPath string, user string) string {
	// 尝试通过SSH获取系统序列号
	var cmd *exec.Cmd
	if user != "" && !strings.Contains(hostname, "@") {
		hostname = fmt.Sprintf("%s@%s", user, hostname)
	}
	command := "cat /sys/class/dmi/id/product_serial"
	sshWrapper := tools.NewSSHWrapper(hostname).Command(command).PrivateKey(sshKeyPath)
	cmd = exec.Command("bash", "-c", sshWrapper.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "N/A"
	}

	serialNumber := strings.TrimSpace(string(output))
	if serialNumber == "" {
		return "N/A"
	}

	// 处理Serial Number：如果包含-则按-分割获取最后一个值
	if strings.Contains(serialNumber, "-") {
		parts := strings.Split(serialNumber, "-")
		if len(parts) > 0 {
			serialNumber = parts[len(parts)-1]
		}
	}

	return serialNumber
}

func collectReportData(reportsDir string, cfg *config.Config) (map[string]map[string]*DeviceData, map[string]map[string]*DeviceData, error) {
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
		// HCA device name is from parts[3] to the second-to-last part (before port number)
		// This supports any HCA naming format: mlx5_0, mlx5_bond_0, mlx5_1_bond, etc.
		device := strings.Join(parts[3:len(parts)-1], "_")

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
			// Get serial number for this hostname
			serialNumber := getSerialNumberForHost(hostname, cfg.SSH.PrivateKey, cfg.SSH.User)

			dataMap[hostname][device] = &DeviceData{
				Hostname:     hostname,
				Device:       device,
				SerialNumber: serialNumber,
				BWSum:        0,
				Count:        0,
				IsClient:     isClient,
			}
		}

		dataMap[hostname][device].BWSum += report.Results.BWAverage
		dataMap[hostname][device].Count++

		return nil
	})

	return clientData, serverData, err
}

// calculateMaxSerialNumberLength 计算数据中最长的序列号长度
func calculateMaxSerialNumberLength(dataMap map[string]map[string]*DeviceData) int {
	maxLen := 13 // 最小长度为"Serial Number"的长度
	for _, hostMap := range dataMap {
		for _, data := range hostMap {
			if len(data.SerialNumber) > maxLen {
				maxLen = len(data.SerialNumber)
			}
		}
	}
	return maxLen
}

// calculateMaxP2PSerialNumberLength 计算P2P数据中最长的序列号长度
func calculateMaxP2PSerialNumberLength(dataMap map[string]map[string]*P2PDeviceData) int {
	maxLen := 13 // 最小长度为"Serial Number"的长度
	for _, hostMap := range dataMap {
		for _, data := range hostMap {
			if len(data.SerialNumber) > maxLen {
				maxLen = len(data.SerialNumber)
			}
		}
	}
	return maxLen
}

// calculateMaxDeviceNameLength 计算数据中最长的设备名称长度
func calculateMaxDeviceNameLength(dataMap map[string]map[string]*DeviceData) int {
	maxLen := 8 // 最小宽度为 "Device" 列标题长度
	for _, devices := range dataMap {
		for device := range devices {
			if len(device) > maxLen {
				maxLen = len(device)
			}
		}
	}
	return maxLen
}

// calculateMaxP2PDeviceNameLength 计算 P2P 数据中最长的设备名称长度
func calculateMaxP2PDeviceNameLength(dataMap map[string]map[string]*P2PDeviceData) int {
	maxLen := 8 // 最小宽度
	for _, devices := range dataMap {
		for device := range devices {
			if len(device) > maxLen {
				maxLen = len(device)
			}
		}
	}
	return maxLen
}

// displayClientTableHeader 显示客户端表格头部（动态列宽）
func displayClientTableHeader(serialNumberWidth, deviceWidth int) {
	serialNumberDashes := strings.Repeat("─", serialNumberWidth)
	deviceDashes := strings.Repeat("─", deviceWidth)
	fmt.Printf("┌─%s─┬─────────────────────┬─%s─┬─────────────┬──────────────┬─────────────────┬──────────┐\n", serialNumberDashes, deviceDashes)
	fmt.Printf("│ %-*s │ Hostname            │ %-*s │ TX (Gbps)   │ SPEC (Gbps)  │ DELTA           │ Status   │\n", serialNumberWidth, "Serial Number", deviceWidth, "Device")
	fmt.Printf("├─%s─┼─────────────────────┼─%s─┼─────────────┼──────────────┼─────────────────┼──────────┤\n", serialNumberDashes, deviceDashes)
}

// displayClientTableFooter 显示客户端表格尾部（动态列宽）
func displayClientTableFooter(serialNumberWidth, deviceWidth int) {
	serialNumberDashes := strings.Repeat("─", serialNumberWidth)
	deviceDashes := strings.Repeat("─", deviceWidth)
	fmt.Printf("└─%s─┴─────────────────────┴─%s─┴─────────────┴──────────────┴─────────────────┴──────────┘\n", serialNumberDashes, deviceDashes)
}

// displayServerTableHeader 显示服务端表格头部（动态列宽）
func displayServerTableHeader(serialNumberWidth, deviceWidth int) {
	serialNumberDashes := strings.Repeat("─", serialNumberWidth)
	deviceDashes := strings.Repeat("─", deviceWidth)
	fmt.Printf("┌─%s─┬─────────────────────┬─%s─┬─────────────┬──────────────┬─────────────────┬──────────┐\n", serialNumberDashes, deviceDashes)
	fmt.Printf("│ %-*s │ Hostname            │ %-*s │ RX (Gbps)   │ SPEC (Gbps)  │ DELTA           │ Status   │\n", serialNumberWidth, "Serial Number", deviceWidth, "Device")
	fmt.Printf("├─%s─┼─────────────────────┼─%s─┼─────────────┼──────────────┼─────────────────┼──────────┤\n", serialNumberDashes, deviceDashes)
}

// displayServerTableFooter 显示服务端表格尾部（动态列宽）
func displayServerTableFooter(serialNumberWidth, deviceWidth int) {
	serialNumberDashes := strings.Repeat("─", serialNumberWidth)
	deviceDashes := strings.Repeat("─", deviceWidth)
	fmt.Printf("└─%s─┴─────────────────────┴─%s─┴─────────────┴──────────────┴─────────────────┴──────────┘\n", serialNumberDashes, deviceDashes)
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

	// 计算最大设备名称长度（客户端和服务端数据合并）
	maxDeviceLen := calculateMaxDeviceNameLength(clientData)
	serverMaxDeviceLen := calculateMaxDeviceNameLength(serverData)
	if serverMaxDeviceLen > maxDeviceLen {
		maxDeviceLen = serverMaxDeviceLen
	}
	if maxDeviceLen < 8 {
		maxDeviceLen = 8 // 最小宽度
	}

	// 计算最大序列号长度
	maxSerialNumberLen := calculateMaxSerialNumberLength(clientData)
	serverMaxSerialNumberLen := calculateMaxSerialNumberLength(serverData)
	if serverMaxSerialNumberLen > maxSerialNumberLen {
		maxSerialNumberLen = serverMaxSerialNumberLen
	}

	// Display client data with enhanced table
	fmt.Println("CLIENT DATA (TX)")
	displayClientTableHeader(maxSerialNumberLen, maxDeviceLen)

	displayEnhancedClientTable(clientData, theoreticalBWPerClient, maxSerialNumberLen, maxDeviceLen)
	displayClientTableFooter(maxSerialNumberLen, maxDeviceLen)

	fmt.Printf("\nTheoretical BW per client: %.2f Gbps (Total server BW: %.2f Gbps ÷ %d clients)\n",
		theoreticalBWPerClient, totalServerBW, clientCount)

	fmt.Println()

	// Display server data with enhanced table
	fmt.Println("SERVER DATA (RX)")
	displayServerTableHeader(maxSerialNumberLen, maxDeviceLen)

	displayEnhancedServerTable(serverData, specSpeed, maxSerialNumberLen, maxDeviceLen)
	displayServerTableFooter(maxSerialNumberLen, maxDeviceLen)
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

	// Server data table with enhanced columns
	content += "## Server Data (RX)\n\n"
	content += "| Hostname | Device | RX (Gbps) | SPEC (Gbps) | DELTA | Status |\n"
	content += "|----------|--------|-----------|-------------|-------|--------|\n"

	content += generateEnhancedMarkdownServerContent(serverData, specSpeed)

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
func displayEnhancedClientTable(clientData map[string]map[string]*DeviceData, theoreticalBW float64, serialNumberWidth, deviceWidth int) {
	// Get sorted hostnames
	var hostnames []string
	for hostname := range clientData {
		hostnames = append(hostnames, hostname)
	}
	sort.Strings(hostnames)

	deviceDashes := strings.Repeat("─", deviceWidth)

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

			// Format serial number and hostname (only show for first device of each host)
			serialNumberStr := ""
			hostnameStr := ""
			if j == 0 {
				serialNumberStr = data.SerialNumber
				hostnameStr = hostname
			}

			fmt.Printf("│ %-*s │ %-19s │ %-*s │ %11.2f │ %12.2f │ %15s │ %-8s │\n",
				serialNumberWidth, serialNumberStr, hostnameStr, deviceWidth, device, actualBW, theoreticalBW, deltaStr, status)
		}

		// Add separator between different hostnames (except for the last one)
		if i < len(hostnames)-1 && len(clientData[hostname]) > 0 {
			serialNumberDashes := strings.Repeat("─", serialNumberWidth)
			fmt.Printf("├─%s─┼─────────────────────┼─%s─┼─────────────┼──────────────┼─────────────────┼──────────┤\n", serialNumberDashes, deviceDashes)
		}
	}
}

// displayEnhancedServerTable 显示增强的服务端表格（包含SPEC、DELTA和Status列）
func displayEnhancedServerTable(serverData map[string]map[string]*DeviceData, specSpeed float64, serialNumberWidth, deviceWidth int) {
	// Get sorted hostnames
	var hostnames []string
	for hostname := range serverData {
		hostnames = append(hostnames, hostname)
	}
	sort.Strings(hostnames)

	deviceDashes := strings.Repeat("─", deviceWidth)

	for i, hostname := range hostnames {
		devices := serverData[hostname]

		// Get sorted devices
		var deviceNames []string
		for device := range devices {
			deviceNames = append(deviceNames, device)
		}
		sort.Strings(deviceNames)

		for j, device := range deviceNames {
			data := devices[device]
			actualBW := data.BWSum        // RX 带宽
			delta := specSpeed - actualBW // DELTA = SPEC - RX
			deltaPercent := float64(0)
			if specSpeed > 0 {
				deltaPercent = (delta / specSpeed) * 100
			}

			// 格式化DELTA列
			deltaStr := fmt.Sprintf("%.1f(%.0f%%)", delta, deltaPercent)

			// 计算状态 - 使用和客户端相同的逻辑
			status := "OK"
			if abs(deltaPercent) > 20 {
				status = "NOT OK"
			}

			// Format serial number and hostname (only show for first device of each host)
			serialNumberStr := ""
			hostnameStr := ""
			if j == 0 {
				serialNumberStr = data.SerialNumber
				hostnameStr = hostname
			}

			fmt.Printf("│ %-*s │ %-19s │ %-*s │ %11.2f │ %12.2f │ %15s │ %-8s │\n",
				serialNumberWidth, serialNumberStr, hostnameStr, deviceWidth, device, actualBW, specSpeed, deltaStr, status)
		}

		// Add separator between different hostnames (except for the last one)
		if i < len(hostnames)-1 && len(serverData[hostname]) > 0 {
			serialNumberDashes := strings.Repeat("─", serialNumberWidth)
			fmt.Printf("├─%s─┼─────────────────────┼─%s─┼─────────────┼──────────────┼─────────────────┼──────────┤\n", serialNumberDashes, deviceDashes)
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

// generateEnhancedMarkdownServerContent 生成增强的服务端Markdown表格内容
func generateEnhancedMarkdownServerContent(serverData map[string]map[string]*DeviceData, specSpeed float64) string {
	var content strings.Builder

	// Get sorted hostnames
	var hostnames []string
	for hostname := range serverData {
		hostnames = append(hostnames, hostname)
	}
	sort.Strings(hostnames)

	for _, hostname := range hostnames {
		devices := serverData[hostname]

		// Get sorted devices
		var deviceNames []string
		for device := range devices {
			deviceNames = append(deviceNames, device)
		}
		sort.Strings(deviceNames)

		for j, device := range deviceNames {
			data := devices[device]
			actualBW := data.BWSum        // RX 带宽
			delta := specSpeed - actualBW // DELTA = SPEC - RX
			deltaPercent := float64(0)
			if specSpeed > 0 {
				deltaPercent = (delta / specSpeed) * 100
			}

			// 格式化DELTA列
			deltaStr := fmt.Sprintf("%.1f(%.0f%%)", delta, deltaPercent)

			// 计算状态 - 使用和客户端相同的逻辑
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
				hostnameStr, device, actualBW, specSpeed, deltaStr, status))
		}
	}

	return content.String()
}

// P2PDeviceData represents aggregated data for a P2P device
type P2PDeviceData struct {
	Hostname     string
	Device       string
	SerialNumber string
	BWSum        float64
	Count        int
}

// parseCustomFormat parses the custom format report files
func parseCustomFormat(content []byte) (*Report, error) {
	text := string(content)

	// Create regex patterns to extract values
	deviceRegex := regexp.MustCompile(`Device:\s*"([^"]+)"`)
	bwAverageRegex := regexp.MustCompile(`BW_average:\s*([0-9]+\.?[0-9]*)`)
	testRegex := regexp.MustCompile(`test:\s*([^,\n]+)`)

	report := &Report{}

	// Extract Device
	if matches := deviceRegex.FindStringSubmatch(text); len(matches) > 1 {
		report.TestInfo.Device = matches[1]
	}

	// Extract Test
	if matches := testRegex.FindStringSubmatch(text); len(matches) > 1 {
		report.TestInfo.Test = strings.TrimSpace(matches[1])
	}

	// Extract BW_average
	if matches := bwAverageRegex.FindStringSubmatch(text); len(matches) > 1 {
		bw, err := strconv.ParseFloat(matches[1], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse BW_average: %v", err)
		}
		report.Results.BWAverage = bw
	}

	return report, nil
}

// parseReportFile attempts to parse both JSON and custom formats
func parseReportFile(path string) (*Report, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	// First try JSON parsing
	var report Report
	if err := json.Unmarshal(content, &report); err == nil {
		return &report, nil
	}

	// If JSON parsing fails, try custom format
	return parseCustomFormat(content)
}

// collectP2PReportData collects report data specifically for P2P mode
func collectP2PReportData(reportsDir string, sshKeyPath string, user string) (map[string]map[string]*P2PDeviceData, error) {
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
		// HCA device name is from parts[2] to the second-to-last part (before port number)
		// This supports any HCA naming format: mlx5_0, mlx5_bond_0, mlx5_1_bond, etc.
		device := strings.Join(parts[2:len(parts)-1], "_")

		// Read and parse JSON file
		// content, err := os.ReadFile(path)
		// if err != nil {
		// 	fmt.Printf("Error reading P2P file %s: %v\n", path, err)
		// 	return nil
		// }

		// var report Report
		// if err := json.Unmarshal(content, &report); err != nil {
		// 	fmt.Printf("Error parsing P2P JSON file %s: %v\n", path, err)
		// 	return nil
		// }
		report, err := parseReportFile(path)
		if err != nil {
			fmt.Printf("Error parsing P2P file %s: %v\n", path, err)
			return nil
		}

		// Initialize hostname map if it doesn't exist
		if p2pData[hostname] == nil {
			p2pData[hostname] = make(map[string]*P2PDeviceData)
		}

		// Initialize or update device data
		if p2pData[hostname][device] == nil {
			// Get serial number for this hostname
			serialNumber := getSerialNumberForHost(hostname, sshKeyPath, user)

			p2pData[hostname][device] = &P2PDeviceData{
				Hostname:     hostname,
				Device:       device,
				SerialNumber: serialNumber,
				BWSum:        0,
				Count:        0,
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

	// 计算最大设备名称长度和序列号长度
	maxDeviceLen := calculateMaxP2PDeviceNameLength(p2pData)
	if maxDeviceLen < 8 {
		maxDeviceLen = 8
	}

	maxSerialNumberLen := calculateMaxP2PSerialNumberLength(p2pData)

	// 显示表格头部
	serialNumberDashes := strings.Repeat("─", maxSerialNumberLen)
	deviceDashes := strings.Repeat("─", maxDeviceLen)
	fmt.Printf("┌─%s─┬─────────────────────┬─%s─┬─────────────┐\n", serialNumberDashes, deviceDashes)
	fmt.Printf("│ %-*s │ Hostname            │ %-*s │ Speed (Gbps)│\n", maxSerialNumberLen, "Serial Number", maxDeviceLen, "Device")
	fmt.Printf("├─%s─┼─────────────────────┼─%s─┼─────────────┤\n", serialNumberDashes, deviceDashes)

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

			// Format serial number and hostname (only show for first device of each host)
			serialNumberStr := ""
			hostnameStr := ""
			if j == 0 {
				serialNumberStr = data.SerialNumber
				hostnameStr = hostname
			}

			fmt.Printf("│ %-*s │ %-19s │ %-*s │ %11.2f │\n",
				maxSerialNumberLen, serialNumberStr, hostnameStr, maxDeviceLen, device, avgSpeed)

			// Add separator between different hosts (except for the last host)
			if j == len(deviceNames)-1 && i < len(hostnames)-1 {
				fmt.Printf("├─%s─┼─────────────────────┼─%s─┼─────────────┤\n", serialNumberDashes, deviceDashes)
			}
		}
	}

	// 显示表格尾部
	fmt.Printf("└─%s─┴─────────────────────┴─%s─┴─────────────┘\n", serialNumberDashes, deviceDashes)

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
		content.WriteString("\n## Summary\n\n")
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
