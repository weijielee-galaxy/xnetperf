package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"xnetperf/config"
	"xnetperf/internal/script"
	"xnetperf/stream"
)

// RunResult 运行测试的结果
type RunResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// ExecuteRun 执行流量测试（不包含 precheck）
func ExecuteRun(cfg *config.Config) (*RunResult, error) {
	result := &RunResult{}

	// 在运行测试前清理远程主机上的旧JSON报告文件
	if cfg.Report.Enable {
		if err := cleanupRemoteReportFiles(cfg); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Failed to cleanup remote files: %v", err)
			return result, err
		}
	}

	err := stream.GenerateScripts(cfg)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Failed to generate scripts: %v", err)
		return result, err
	}

	// 分发并运行脚本
	stream.DistributeAndRunScripts(cfg)

	result.Success = true
	result.Message = "Test scripts distributed and started successfully"
	return result, nil
}

// ExecuteRun 执行流量测试（不包含 precheck）
func ExecuteRunV1(cfg *config.Config) (*RunResult, error) {
	result := &RunResult{}

	// 在运行测试前清理远程主机上的旧JSON报告文件
	if cfg.Report.Enable {
		if err := cleanupRemoteReportFiles(cfg); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Failed to cleanup remote files: %v", err)
			return result, err
		}
	}

	executor := script.NewExecutor(cfg, script.TestTypeBandwidth)
	if executor == nil {
		result.Success = false
		result.Error = "Unsupported stream type for v1 execute workflow"
		return result, fmt.Errorf("unsupported stream type for v1 execute workflow")
	}

	err := executor.Execute()
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Failed to execute scripts: %v", err)
		return result, err
	}

	result.Success = true
	result.Message = "Test scripts distributed and started successfully"
	return result, nil
}

func cleanupRemoteReportFiles(cfg *config.Config) error {
	fmt.Println("Cleaning up old report files on remote hosts before starting tests...")

	// 获取所有主机列表
	allHosts := make(map[string]bool)
	for _, host := range cfg.Server.Hostname {
		allHosts[host] = true
	}
	for _, host := range cfg.Client.Hostname {
		allHosts[host] = true
	}

	var wg sync.WaitGroup
	errors := make([]string, 0)
	var mu sync.Mutex

	for hostname := range allHosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()

			// 删除远程主机上属于当前主机的JSON报告文件
			rmCmd := fmt.Sprintf("rm -f %s/*%s*.json", cfg.Report.Dir, host)
			cmd := buildSSHCommand(host, rmCmd, cfg.SSH.PrivateKey)

			output, err := cmd.CombinedOutput()
			if err != nil {
				mu.Lock()
				errors = append(errors, fmt.Sprintf("%s: %v", host, err))
				mu.Unlock()
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

	if len(errors) > 0 {
		return fmt.Errorf("cleanup failed on some hosts: %s", strings.Join(errors, "; "))
	}
	return nil
}

// ProbeResult 探测结果
type ProbeResult struct {
	Hostname     string   `json:"hostname"`
	ProcessCount int      `json:"process_count"`
	Processes    []string `json:"processes,omitempty"`
	Error        string   `json:"error,omitempty"`
	Status       string   `json:"status"` // RUNNING, COMPLETED, ERROR
}

// ProbeSum mary 探测汇总
type ProbeSummary struct {
	Timestamp      string        `json:"timestamp"`
	Results        []ProbeResult `json:"results"`
	RunningHosts   int           `json:"running_hosts"`
	CompletedHosts int           `json:"completed_hosts"`
	ErrorHosts     int           `json:"error_hosts"`
	TotalProcesses int           `json:"total_processes"`
	AllCompleted   bool          `json:"all_completed"`
}

// ExecuteProbe 执行探测并返回结构化数据
func ExecuteProbe(cfg *config.Config) (*ProbeSummary, error) {
	// 获取所有主机列表
	allHosts := make(map[string]bool)
	for _, host := range cfg.Server.Hostname {
		allHosts[host] = true
	}
	for _, host := range cfg.Client.Hostname {
		allHosts[host] = true
	}

	if len(allHosts) == 0 {
		return nil, fmt.Errorf("no hosts configured in config file")
	}

	// 探测所有主机
	results := probeAllHosts(allHosts, cfg.SSH.PrivateKey)

	// 统计信息
	summary := &ProbeSummary{
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

	summary.AllCompleted = (summary.CompletedHosts == len(allHosts))

	return summary, nil
}

func probeAllHosts(hosts map[string]bool, sshKeyPath string) []ProbeResult {
	var results []ProbeResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for hostname := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			result := probeHost(host, sshKeyPath)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(hostname)
	}

	wg.Wait()
	return results
}

func probeHost(hostname string, sshKeyPath string) ProbeResult {
	result := ProbeResult{
		Hostname: hostname,
	}

	// 使用SSH执行ps命令查找ib_write_bw进程
	cmd := buildSSHCommand(hostname, "ps aux | grep ib_write_bw | grep -v grep", sshKeyPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// 如果没有找到进程或SSH连接失败
		if strings.Contains(string(output), "") && cmd.ProcessState != nil && cmd.ProcessState.ExitCode() == 1 {
			// ps命令返回1通常表示没有找到匹配的进程
			result.ProcessCount = 0
			result.Status = "COMPLETED"
		} else {
			result.Error = fmt.Sprintf("SSH error: %v", err)
			result.Status = "ERROR"
			fmt.Printf("Error: %s - %s\n", hostname, result.Error)
		}
		return result
	}

	// 解析输出
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		result.ProcessCount = 0
		result.Status = "COMPLETED"
		return result
	}

	// 过滤和计数有效的进程行
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

// CollectResult 收集结果
type CollectResult struct {
	Success        bool           `json:"success"`
	Message        string         `json:"message"`
	CollectedFiles map[string]int `json:"collected_files"` // hostname -> file count
	Error          string         `json:"error,omitempty"`
}

// ExecuteCollect 执行收集并返回结果
func ExecuteCollect(cfg *config.Config) (*CollectResult, error) {
	result := &CollectResult{
		CollectedFiles: make(map[string]int),
	}

	if !cfg.Report.Enable {
		result.Success = false
		result.Error = "Report is not enabled in config"
		return result, fmt.Errorf("report is not enabled in config")
	}

	// 创建本地reports目录
	reportsDir := "reports"

	// 删除已存在的reports目录
	if _, err := os.Stat(reportsDir); err == nil {
		err = os.RemoveAll(reportsDir)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("Failed to remove existing reports directory: %v", err)
			return result, err
		}
		fmt.Printf("Removed existing reports directory\n")
	}

	// 创建新的reports目录
	err := os.MkdirAll(reportsDir, 0755)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Failed to create reports directory: %v", err)
		return result, err
	}

	// 获取所有主机列表
	allHosts := make(map[string]bool)
	for _, host := range cfg.Server.Hostname {
		allHosts[host] = true
	}
	for _, host := range cfg.Client.Hostname {
		allHosts[host] = true
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	fmt.Printf("Collecting reports from %d hosts...\n", len(allHosts))

	for hostname := range allHosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			count := collectFromHost(host, cfg.Report.Dir, reportsDir)
			mu.Lock()
			result.CollectedFiles[host] = count
			mu.Unlock()
		}(hostname)
	}

	wg.Wait()

	result.Success = true
	result.Message = fmt.Sprintf("Report collection completed from %d hosts", len(allHosts))
	fmt.Printf("Report collection completed. Files saved to '%s' directory.\n", reportsDir)

	return result, nil
}

func collectFromHost(hostname, remoteDir, localBaseDir string) int {
	// 为每个主机创建本地子目录
	hostDir := filepath.Join(localBaseDir, hostname)
	err := os.MkdirAll(hostDir, 0755)
	if err != nil {
		fmt.Printf("Error creating directory for host %s: %v\n", hostname, err)
		return 0
	}

	fmt.Printf("-> Collecting reports from %s...\n", hostname)

	// 使用scp收集报告文件
	scpCmd := fmt.Sprintf("%s/*%s*.json", remoteDir, hostname)
	cmd := exec.Command("scp", fmt.Sprintf("%s:%s", hostname, scpCmd), hostDir+"/")

	output, err := cmd.CombinedOutput()
	if err != nil {
		if string(output) != "" {
			fmt.Printf("   [WARNING] ⚠️  %s: %s\n", hostname, string(output))
		} else {
			fmt.Printf("   [WARNING] ⚠️  %s: No report files found or scp failed: %v\n", hostname, err)
		}
		return 0
	}

	// 计算收集到的文件数量
	files, err := filepath.Glob(filepath.Join(hostDir, "*.json"))
	if err != nil {
		fmt.Printf("   [ERROR] ❌ %s: Error counting files: %v\n", hostname, err)
		return 0
	}

	if len(files) > 0 {
		fmt.Printf("   [SUCCESS] ✅ %s: Collected %d report files\n", hostname, len(files))
		return len(files)
	} else {
		fmt.Printf("   [INFO] ℹ️  %s: No report files found\n", hostname)
		return 0
	}
}

// ReportData 报告数据结构
type ReportData struct {
	StreamType             string                                  `json:"stream_type"`
	TheoreticalBWPerClient float64                                 `json:"theoretical_bw_per_client,omitempty"`
	TotalServerBW          float64                                 `json:"total_server_bw,omitempty"`
	ClientCount            int                                     `json:"client_count,omitempty"`
	ClientData             map[string]map[string]*ClientDeviceData `json:"client_data,omitempty"`
	ServerData             map[string]map[string]*ServerDeviceData `json:"server_data,omitempty"`
	P2PData                map[string]map[string]*P2PDeviceData    `json:"p2p_data,omitempty"`
	P2PSummary             *P2PSummary                             `json:"p2p_summary,omitempty"`
}

// ClientDeviceData 客户端设备数据
type ClientDeviceData struct {
	Hostname      string  `json:"hostname"`
	Device        string  `json:"device"`
	ActualBW      float64 `json:"actual_bw"`
	TheoreticalBW float64 `json:"theoretical_bw"`
	Delta         float64 `json:"delta"`
	DeltaPercent  float64 `json:"delta_percent"`
	Status        string  `json:"status"` // OK, NOT OK
}

// ServerDeviceData 服务端设备数据
type ServerDeviceData struct {
	Hostname string  `json:"hostname"`
	Device   string  `json:"device"`
	RxBW     float64 `json:"rx_bw"`
}

// P2PDeviceData P2P设备数据
type P2PDeviceData struct {
	Hostname string  `json:"hostname"`
	Device   string  `json:"device"`
	AvgSpeed float64 `json:"avg_speed"`
	Count    int     `json:"count"`
}

// P2PSummary P2P汇总数据
type P2PSummary struct {
	TotalPairs int     `json:"total_pairs"`
	AvgSpeed   float64 `json:"avg_speed"`
}

// Report JSON报告文件结构
type Report struct {
	TestInfo struct {
		Test   string `json:"test"`
		Device string `json:"Device"`
	} `json:"test_info"`
	Results struct {
		BWAverage float64 `json:"BW_average"`
	} `json:"results"`
}

// deviceData 内部使用的设备数据累加结构
type deviceData struct {
	Hostname string
	Device   string
	BWSum    float64
	Count    int
	IsClient bool
}

// GenerateReport 生成报告数据
func GenerateReport(cfg *config.Config) (*ReportData, error) {
	reportsDir := "reports"

	// 检查 reports 目录是否存在
	if _, err := os.Stat(reportsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("reports directory not found: %s", reportsDir)
	}

	report := &ReportData{
		StreamType: string(cfg.StreamType),
	}

	switch cfg.StreamType {
	case config.P2P:
		// P2P 分析
		p2pData, err := collectP2PReportData(reportsDir)
		if err != nil {
			return nil, fmt.Errorf("failed to collect P2P report data: %v", err)
		}

		// 转换为 API 响应格式
		report.P2PData = convertP2PData(p2pData)
		report.P2PSummary = calculateP2PSummary(p2pData)

	default:
		// FullMesh 和 InCast 分析
		clientData, serverData, err := collectTraditionalReportData(reportsDir)
		if err != nil {
			return nil, fmt.Errorf("failed to collect report data: %v", err)
		}

		// 计算理论带宽
		report.TotalServerBW = calculateTotalServerBandwidth(serverData, cfg.Speed)
		report.ClientCount = calculateClientCount(clientData)
		if report.ClientCount > 0 {
			report.TheoreticalBWPerClient = report.TotalServerBW / float64(report.ClientCount)
		}

		// 转换为 API 响应格式
		report.ClientData = convertClientData(clientData, report.TheoreticalBWPerClient)
		report.ServerData = convertServerData(serverData)
	}

	return report, nil
}

func collectTraditionalReportData(reportsDir string) (map[string]map[string]*deviceData, map[string]map[string]*deviceData, error) {
	clientData := make(map[string]map[string]*deviceData)
	serverData := make(map[string]map[string]*deviceData)

	err := filepath.Walk(reportsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}

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

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var report Report
		if err := json.Unmarshal(content, &report); err != nil {
			return nil
		}

		var dataMap map[string]map[string]*deviceData
		if isClient {
			dataMap = clientData
		} else {
			dataMap = serverData
		}

		if dataMap[hostname] == nil {
			dataMap[hostname] = make(map[string]*deviceData)
		}

		if dataMap[hostname][device] == nil {
			dataMap[hostname][device] = &deviceData{
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

func collectP2PReportData(reportsDir string) (map[string]map[string]*deviceData, error) {
	p2pData := make(map[string]map[string]*deviceData)

	err := filepath.Walk(reportsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(info.Name(), ".json") {
			return nil
		}

		filename := info.Name()

		// 跳过传统 client/server 报告文件
		if strings.HasPrefix(filename, "report_c_") || strings.HasPrefix(filename, "report_s_") {
			return nil
		}

		// 必须以 "report_" 开头
		if !strings.HasPrefix(filename, "report_") {
			return nil
		}

		parts := strings.Split(filename, "_")
		if len(parts) < 4 {
			return nil
		}

		hostname := parts[1]
		// HCA device name is from parts[2] to the second-to-last part (before port number)
		// This supports any HCA naming format: mlx5_0, mlx5_bond_0, mlx5_1_bond, etc.
		device := strings.Join(parts[2:len(parts)-1], "_")

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var report Report
		if err := json.Unmarshal(content, &report); err != nil {
			return nil
		}

		if p2pData[hostname] == nil {
			p2pData[hostname] = make(map[string]*deviceData)
		}

		if p2pData[hostname][device] == nil {
			p2pData[hostname][device] = &deviceData{
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

func calculateTotalServerBandwidth(serverData map[string]map[string]*deviceData, specSpeed float64) float64 {
	total := float64(0)
	for _, devices := range serverData {
		for range devices {
			total += specSpeed
		}
	}
	return total
}

func calculateClientCount(clientData map[string]map[string]*deviceData) int {
	count := 0
	for _, devices := range clientData {
		count += len(devices)
	}
	return count
}

func convertClientData(clientData map[string]map[string]*deviceData, theoreticalBW float64) map[string]map[string]*ClientDeviceData {
	result := make(map[string]map[string]*ClientDeviceData)

	for hostname, devices := range clientData {
		result[hostname] = make(map[string]*ClientDeviceData)
		for device, data := range devices {
			actualBW := data.BWSum
			delta := actualBW - theoreticalBW
			deltaPercent := float64(0)
			if theoreticalBW > 0 {
				deltaPercent = (delta / theoreticalBW) * 100
			}

			status := "OK"
			if abs(deltaPercent) > 20 {
				status = "NOT OK"
			}

			result[hostname][device] = &ClientDeviceData{
				Hostname:      hostname,
				Device:        device,
				ActualBW:      actualBW,
				TheoreticalBW: theoreticalBW,
				Delta:         delta,
				DeltaPercent:  deltaPercent,
				Status:        status,
			}
		}
	}

	return result
}

func convertServerData(serverData map[string]map[string]*deviceData) map[string]map[string]*ServerDeviceData {
	result := make(map[string]map[string]*ServerDeviceData)

	for hostname, devices := range serverData {
		result[hostname] = make(map[string]*ServerDeviceData)
		for device, data := range devices {
			result[hostname][device] = &ServerDeviceData{
				Hostname: hostname,
				Device:   device,
				RxBW:     data.BWSum,
			}
		}
	}

	return result
}

func convertP2PData(p2pData map[string]map[string]*deviceData) map[string]map[string]*P2PDeviceData {
	result := make(map[string]map[string]*P2PDeviceData)

	for hostname, devices := range p2pData {
		result[hostname] = make(map[string]*P2PDeviceData)
		for device, data := range devices {
			result[hostname][device] = &P2PDeviceData{
				Hostname: hostname,
				Device:   device,
				AvgSpeed: data.BWSum / float64(data.Count),
				Count:    data.Count,
			}
		}
	}

	return result
}

func calculateP2PSummary(p2pData map[string]map[string]*deviceData) *P2PSummary {
	totalPairs := 0
	totalSpeed := 0.0

	for _, devices := range p2pData {
		for _, data := range devices {
			totalPairs++
			totalSpeed += data.BWSum / float64(data.Count)
		}
	}

	summary := &P2PSummary{
		TotalPairs: totalPairs,
	}

	if totalPairs > 0 {
		summary.AvgSpeed = totalSpeed / float64(totalPairs)
	}

	return summary
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// buildSSHCommand builds an ssh command with optional private key
func buildSSHCommand(hostname, remoteCmd, sshKeyPath string) *exec.Cmd {
	if sshKeyPath != "" {
		return exec.Command("ssh", "-i", sshKeyPath, hostname, remoteCmd)
	}
	return exec.Command("ssh", hostname, remoteCmd)
}
