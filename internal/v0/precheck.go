package v0

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"xnetperf/config"
	"xnetperf/pkg/tools"
)

// ANSI 颜色代码
const (
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorReset  = "\033[0m"
)

// PrecheckResult 表示预检查结果
type PrecheckResult struct {
	Hostname     string
	HCA          string
	PhysState    string
	State        string
	Speed        string
	FwVer        string
	BoardId      string
	SerialNumber string
	IsHealthy    bool
	Error        string
}

// ExecPrecheckCommand
// Deprecated: 旧版命令行接口，保留以兼容历史数据和文档
func ExecPrecheckCommand(cfg *config.Config) bool {
	fmt.Println("Starting InfiniBand HCA precheck...")
	fmt.Println()

	// 收集所有需要检查的主机和HCA
	var checkItems []struct {
		hostname string
		hca      string
	}

	// 添加服务器端的HCA
	for _, hostname := range cfg.Server.Hostname {
		for _, hca := range cfg.Server.Hca {
			checkItems = append(checkItems, struct {
				hostname string
				hca      string
			}{hostname, hca})
		}
	}

	// 添加客户端的HCA
	for _, hostname := range cfg.Client.Hostname {
		for _, hca := range cfg.Client.Hca {
			checkItems = append(checkItems, struct {
				hostname string
				hca      string
			}{hostname, hca})
		}
	}

	if len(checkItems) == 0 {
		fmt.Println("No HCAs configured in config file")
		return false
	}

	fmt.Printf("Checking %d HCAs across all hosts...\n\n", len(checkItems))

	// 并发执行检查
	results := precheckAllHCAs(checkItems, cfg.SSH.PrivateKey, cfg.SSH.User)

	// 显示结果
	displayPrecheckResults(results)

	// 检查是否所有HCA都健康
	allHealthy := true
	for _, result := range results {
		if !result.IsHealthy {
			allHealthy = false
			break
		}
	}

	// v0.0.3: 检查所有 speed 是否相同
	allSpeedsSame := true
	if len(results) > 1 {
		firstSpeed := ""
		for _, result := range results {
			if result.Error == "" && result.Speed != "" {
				if firstSpeed == "" {
					firstSpeed = result.Speed
				} else if result.Speed != firstSpeed {
					allSpeedsSame = false
					break
				}
			}
		}
	}

	return allHealthy && allSpeedsSame
}

func precheckAllHCAs(checkItems []struct {
	hostname string
	hca      string
}, sshKeyPath, user string) []PrecheckResult {
	var results []PrecheckResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, item := range checkItems {
		wg.Add(1)
		go func(hostname, hca string) {
			defer wg.Done()
			result := precheckHCA(hostname, hca, sshKeyPath, user)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(item.hostname, item.hca)
	}

	wg.Wait()
	return results
}

func precheckHCA(hostname, hca, sshKeyPath, user string) PrecheckResult {
	result := PrecheckResult{
		Hostname: hostname,
		HCA:      hca,
	}

	// 检查物理状态
	physStateCmd := fmt.Sprintf("cat /sys/class/infiniband/%s/ports/1/phys_state", hca)
	cmd := tools.BuildSSHCommand(hostname, physStateCmd, sshKeyPath, user)
	physOutput, err := cmd.CombinedOutput()

	if err != nil {
		result.Error = fmt.Sprintf("Failed to check phys_state: %v", err)
		return result
	}

	// 检查逻辑状态
	stateCmd := fmt.Sprintf("cat /sys/class/infiniband/%s/ports/1/state", hca)
	cmd = tools.BuildSSHCommand(hostname, stateCmd, sshKeyPath, user)
	stateOutput, err := cmd.CombinedOutput()

	if err != nil {
		result.Error = fmt.Sprintf("Failed to check state: %v", err)
		return result
	}

	// 检查网卡速度
	speedCmd := fmt.Sprintf("cat /sys/class/infiniband/%s/ports/1/rate", hca)
	cmd = tools.BuildSSHCommand(hostname, speedCmd, sshKeyPath, user)
	speedOutput, err := cmd.CombinedOutput()

	if err != nil {
		result.Error = fmt.Sprintf("Failed to check speed: %v", err)
		return result
	}

	// 检查固件版本
	fwVerCmd := fmt.Sprintf("cat /sys/class/infiniband/%s/fw_ver", hca)
	cmd = tools.BuildSSHCommand(hostname, fwVerCmd, sshKeyPath, user)
	fwVerOutput, err := cmd.CombinedOutput()

	if err != nil {
		result.Error = fmt.Sprintf("Failed to check fw_ver: %v", err)
		return result
	}

	// 检查板卡ID
	boardIdCmd := fmt.Sprintf("cat /sys/class/infiniband/%s/board_id", hca)
	cmd = tools.BuildSSHCommand(hostname, boardIdCmd, sshKeyPath, user)
	boardIdOutput, err := cmd.CombinedOutput()

	if err != nil {
		result.Error = fmt.Sprintf("Failed to check board_id: %v", err)
		return result
	}

	// 检查系统序列号
	serialNumberCmd := "cat /sys/class/dmi/id/product_serial"
	cmd = tools.BuildSSHCommand(hostname, serialNumberCmd, sshKeyPath, user)
	serialNumberOutput, err := cmd.CombinedOutput()

	if err != nil {
		result.Error = fmt.Sprintf("Failed to check serial number: %v", err)
		return result
	}

	// 解析输出
	physStateStr := strings.TrimSpace(string(physOutput))
	stateStr := strings.TrimSpace(string(stateOutput))
	speedStr := strings.TrimSpace(string(speedOutput))
	fwVerStr := strings.TrimSpace(string(fwVerOutput))
	boardIdStr := strings.TrimSpace(string(boardIdOutput))
	serialNumberStr := strings.TrimSpace(string(serialNumberOutput))

	// 处理Serial Number：如果包含-则按-分割获取最后一个值
	if strings.Contains(serialNumberStr, "-") {
		parts := strings.Split(serialNumberStr, "-")
		if len(parts) > 0 {
			serialNumberStr = parts[len(parts)-1]
		}
	}

	// 去掉状态前面的数字和冒号，只保留有意义的文本
	result.PhysState = cleanStateString(physStateStr)
	result.State = cleanStateString(stateStr)
	result.Speed = speedStr // 保持速度信息的原始格式
	result.FwVer = fwVerStr
	result.BoardId = boardIdStr
	result.SerialNumber = serialNumberStr

	// 判断是否健康：需要同时满足 LinkUp 和 ACTIVE
	isLinkUp := strings.Contains(physStateStr, "LinkUp")
	isActive := strings.Contains(stateStr, "ACTIVE")

	result.IsHealthy = isLinkUp && isActive

	return result
}

// cleanStateString 去掉状态字符串前面的数字和冒号，只保留有意义的文本
// 例如: "5: LinkUp" -> "LinkUp", "4: ACTIVE" -> "ACTIVE"
func cleanStateString(stateStr string) string {
	// 查找冒号的位置
	colonIndex := strings.Index(stateStr, ":")
	if colonIndex == -1 {
		// 如果没有冒号，返回原字符串
		return stateStr
	}

	// 返回冒号后面的内容，去掉前后空格
	return strings.TrimSpace(stateStr[colonIndex+1:])
}

// padStringWithColor 处理带颜色代码的字符串，确保正确的填充宽度
func padStringWithColor(str string, width int) string {
	// 计算不包含ANSI颜色代码的实际显示宽度
	visibleLen := len(removeAnsiCodes(str))
	if visibleLen >= width {
		return str
	}
	// 添加适当的空格填充
	padding := strings.Repeat(" ", width-visibleLen)
	return str + padding
}

// removeAnsiCodes 移除字符串中的ANSI颜色代码，用于计算实际显示宽度
func removeAnsiCodes(str string) string {
	// 简单的ANSI代码移除 - 移除 \033[...m 格式的代码
	result := str
	for {
		start := strings.Index(result, "\033[")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "m")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	return result
}

func displayPrecheckResults(results []PrecheckResult) {
	fmt.Printf("=== Precheck Results (%s) ===\n", time.Now().Format("15:04:05"))

	// v0.0.4: 按照 hostname 然后按 HCA 排序
	sort.Slice(results, func(i, j int) bool {
		if results[i].Hostname != results[j].Hostname {
			return results[i].Hostname < results[j].Hostname
		}
		return results[i].HCA < results[j].HCA
	})

	// 统计 Speed 值的出现次数，用于着色
	speedCounts := make(map[string]int)
	for _, result := range results {
		if result.Error == "" && result.Speed != "" {
			speedCounts[result.Speed]++
		}
	}

	// 找出最常见的速度值（用于绿色显示）
	maxSpeedCount := 0
	for _, count := range speedCounts {
		if count > maxSpeedCount {
			maxSpeedCount = count
		}
	}

	// v0.0.4: 统计 FW Version 值的出现次数，用于着色
	fwVerCounts := make(map[string]int)
	for _, result := range results {
		if result.Error == "" && result.FwVer != "" {
			fwVerCounts[result.FwVer]++
		}
	}

	// 找出最常见的 FW Version 值
	maxFwVerCount := 0
	for _, count := range fwVerCounts {
		if count > maxFwVerCount {
			maxFwVerCount = count
		}
	}

	// v0.0.4: 统计 Board ID 值的出现次数，用于着色
	boardIdCounts := make(map[string]int)
	for _, result := range results {
		if result.Error == "" && result.BoardId != "" {
			boardIdCounts[result.BoardId]++
		}
	}

	// 找出最常见的 Board ID 值
	maxBoardIdCount := 0
	for _, count := range boardIdCounts {
		if count > maxBoardIdCount {
			maxBoardIdCount = count
		}
	}

	// 计算动态列宽
	maxHostnameWidth := len("Hostname")
	maxHCAWidth := len("HCA")
	maxPhysStateWidth := len("Physical State")
	maxStateWidth := len("Logical State")
	maxSpeedWidth := len("Speed")
	maxFwVerWidth := len("FW Version")
	maxBoardIdWidth := len("Board ID")
	maxSerialNumberWidth := len("Serial Number")
	maxStatusWidth := len("Status")

	for _, result := range results {
		if len(result.Hostname) > maxHostnameWidth {
			maxHostnameWidth = len(result.Hostname)
		}
		if len(result.HCA) > maxHCAWidth {
			maxHCAWidth = len(result.HCA)
		}
		if len(result.PhysState) > maxPhysStateWidth {
			maxPhysStateWidth = len(result.PhysState)
		}
		if len(result.State) > maxStateWidth {
			maxStateWidth = len(result.State)
		}
		if len(result.Speed) > maxSpeedWidth {
			maxSpeedWidth = len(result.Speed)
		}
		if len(result.FwVer) > maxFwVerWidth {
			maxFwVerWidth = len(result.FwVer)
		}
		if len(result.BoardId) > maxBoardIdWidth {
			maxBoardIdWidth = len(result.BoardId)
		}
		if len(result.SerialNumber) > maxSerialNumberWidth {
			maxSerialNumberWidth = len(result.SerialNumber)
		}

		// 计算状态文本长度，包含符号前缀（不含颜色代码）
		var statusText string
		if result.Error != "" {
			statusText = "[!] ERROR    " // 9+4=13字符
		} else if result.IsHealthy {
			statusText = "[+] HEALTHY  " // 11+2=13字符
		} else {
			statusText = "[X] UNHEALTHY" // 13字符
		}
		if len(statusText) > maxStatusWidth {
			maxStatusWidth = len(statusText)
		}
	}

	// 确保最小宽度
	if maxHostnameWidth < 12 {
		maxHostnameWidth = 12
	}
	if maxHCAWidth < 8 {
		maxHCAWidth = 8
	}
	if maxPhysStateWidth < 12 {
		maxPhysStateWidth = 12
	}
	if maxStateWidth < 12 {
		maxStateWidth = 12
	}
	if maxSpeedWidth < 15 {
		maxSpeedWidth = 15
	}
	if maxFwVerWidth < 12 {
		maxFwVerWidth = 12
	}
	if maxBoardIdWidth < 15 {
		maxBoardIdWidth = 15
	}
	if maxSerialNumberWidth < 15 {
		maxSerialNumberWidth = 15
	}
	if maxStatusWidth < 10 {
		maxStatusWidth = 10
	}

	// 生成表格格式（9列：Serial Number, Hostname, HCA, Physical State, Logical State, Speed, FW Version, Board ID, Status）
	headerFormat := fmt.Sprintf("│ %%-%ds │ %%-%ds │ %%-%ds │ %%-%ds │ %%-%ds │ %%-%ds │ %%-%ds │ %%-%ds │ %%-%ds │\n",
		maxSerialNumberWidth, maxHostnameWidth, maxHCAWidth, maxPhysStateWidth, maxStateWidth, maxSpeedWidth, maxFwVerWidth, maxBoardIdWidth, maxStatusWidth)
	separatorFormat := fmt.Sprintf("├─%%-%ds─┼─%%-%ds─┼─%%-%ds─┼─%%-%ds─┼─%%-%ds─┼─%%-%ds─┼─%%-%ds─┼─%%-%ds─┼─%%-%ds─┤\n",
		maxSerialNumberWidth, maxHostnameWidth, maxHCAWidth, maxPhysStateWidth, maxStateWidth, maxSpeedWidth, maxFwVerWidth, maxBoardIdWidth, maxStatusWidth)

	// 生成边框
	topBorder := fmt.Sprintf("┌─%s─┬─%s─┬─%s─┬─%s─┬─%s─┬─%s─┬─%s─┬─%s─┬─%s─┐\n",
		strings.Repeat("─", maxSerialNumberWidth),
		strings.Repeat("─", maxHostnameWidth),
		strings.Repeat("─", maxHCAWidth),
		strings.Repeat("─", maxPhysStateWidth),
		strings.Repeat("─", maxStateWidth),
		strings.Repeat("─", maxSpeedWidth),
		strings.Repeat("─", maxFwVerWidth),
		strings.Repeat("─", maxBoardIdWidth),
		strings.Repeat("─", maxStatusWidth))
	bottomBorder := fmt.Sprintf("└─%s─┴─%s─┴─%s─┴─%s─┴─%s─┴─%s─┴─%s─┴─%s─┴─%s─┘\n",
		strings.Repeat("─", maxSerialNumberWidth),
		strings.Repeat("─", maxHostnameWidth),
		strings.Repeat("─", maxHCAWidth),
		strings.Repeat("─", maxPhysStateWidth),
		strings.Repeat("─", maxStateWidth),
		strings.Repeat("─", maxSpeedWidth),
		strings.Repeat("─", maxFwVerWidth),
		strings.Repeat("─", maxBoardIdWidth),
		strings.Repeat("─", maxStatusWidth))

	// 打印表格
	fmt.Print(topBorder)
	fmt.Printf(headerFormat, "Serial Number", "Hostname", "HCA", "Physical State", "Logical State", "Speed", "FW Version", "Board ID", "Status")
	fmt.Printf(separatorFormat,
		strings.Repeat("─", maxSerialNumberWidth),
		strings.Repeat("─", maxHostnameWidth),
		strings.Repeat("─", maxHCAWidth),
		strings.Repeat("─", maxPhysStateWidth),
		strings.Repeat("─", maxStateWidth),
		strings.Repeat("─", maxSpeedWidth),
		strings.Repeat("─", maxFwVerWidth),
		strings.Repeat("─", maxBoardIdWidth),
		strings.Repeat("─", maxStatusWidth))

	// 打印数据行 - v0.0.4: 实现 hostname 合并
	var lastHostname string
	for i, result := range results {
		physState := result.PhysState
		logicalState := result.State
		speed := result.Speed
		fwVer := result.FwVer
		boardId := result.BoardId

		// 处理 Serial Number
		serialNumber := result.SerialNumber
		if serialNumber == "" {
			serialNumber = "N/A"
		}

		// 根据状态着色
		var coloredStatus string
		if result.Error != "" {
			// 简化错误显示，直接在状态列中显示
			physState = "N/A"
			logicalState = "N/A"
			speed = "N/A"
			fwVer = "N/A"
			boardId = "N/A"
			serialNumber = "N/A"
			coloredStatus = ColorYellow + "[!] ERROR    " + ColorReset
		} else if result.IsHealthy {
			coloredStatus = ColorGreen + "[+] HEALTHY  " + ColorReset
		} else {
			coloredStatus = ColorRed + "[X] UNHEALTHY" + ColorReset
		}

		// 根据速度着色
		var coloredSpeed string
		if result.Error == "" && speed != "" {
			speedCount := speedCounts[speed]
			if speedCount == maxSpeedCount && maxSpeedCount > 1 { // 数量最多的标绿色
				coloredSpeed = ColorGreen + speed + ColorReset
			} else if speedCount < maxSpeedCount { // 数量少的标红色
				coloredSpeed = ColorRed + speed + ColorReset
			} else {
				coloredSpeed = speed // 相同数量或只有一个不着色
			}
		} else {
			coloredSpeed = speed
		}

		// v0.0.4: 根据 FW Version 着色
		var coloredFwVer string
		if result.Error == "" && fwVer != "" {
			fwVerCount := fwVerCounts[fwVer]
			if fwVerCount < maxFwVerCount { // 数量少的标黄色
				coloredFwVer = ColorYellow + fwVer + ColorReset
			} else {
				coloredFwVer = fwVer // 数量多的不着色
			}
		} else {
			coloredFwVer = fwVer
		}

		// v0.0.4: 根据 Board ID 着色
		var coloredBoardId string
		if result.Error == "" && boardId != "" {
			boardIdCount := boardIdCounts[boardId]
			if boardIdCount < maxBoardIdCount { // 数量少的标黄色
				coloredBoardId = ColorYellow + boardId + ColorReset
			} else {
				coloredBoardId = boardId // 数量多的不着色
			}
		} else {
			coloredBoardId = boardId
		}

		// v0.0.4: 实现 hostname 合并逻辑和分隔线
		displayHostname := result.Hostname
		if result.Hostname == lastHostname {
			displayHostname = "" // 相同的 hostname 显示为空，实现合并效果
		} else {
			// 在不同hostname之间添加分隔线（除了第一行）
			if i > 0 {
				fmt.Printf(separatorFormat,
					strings.Repeat("─", maxSerialNumberWidth),
					strings.Repeat("─", maxHostnameWidth),
					strings.Repeat("─", maxHCAWidth),
					strings.Repeat("─", maxPhysStateWidth),
					strings.Repeat("─", maxStateWidth),
					strings.Repeat("─", maxSpeedWidth),
					strings.Repeat("─", maxFwVerWidth),
					strings.Repeat("─", maxBoardIdWidth),
					strings.Repeat("─", maxStatusWidth))
			}
			lastHostname = result.Hostname
		}

		// 使用固定格式，不依赖颜色代码的长度
		fmt.Printf("│ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │ %s │ %s │ %s │ %s │\n",
			maxSerialNumberWidth, serialNumber,
			maxHostnameWidth, displayHostname,
			maxHCAWidth, result.HCA,
			maxPhysStateWidth, physState,
			maxStateWidth, logicalState,
			padStringWithColor(coloredSpeed, maxSpeedWidth),
			padStringWithColor(coloredFwVer, maxFwVerWidth),
			padStringWithColor(coloredBoardId, maxBoardIdWidth),
			padStringWithColor(coloredStatus, maxStatusWidth))
	}

	fmt.Print(bottomBorder)

	// 显示统计信息
	healthy := 0
	unhealthy := 0
	errors := 0

	for _, result := range results {
		if result.Error != "" {
			errors++
		} else if result.IsHealthy {
			healthy++
		} else {
			unhealthy++
		}
	}

	fmt.Printf("Summary: %d healthy, %d unhealthy, %d errors (Total: %d HCAs)\n",
		healthy, unhealthy, errors, len(results))
}

// PrecheckSummary API 返回的 precheck 汇总信息
type PrecheckSummary struct {
	TotalHCAs      int              `json:"total_hcas"`
	HealthyCount   int              `json:"healthy_count"`
	UnhealthyCount int              `json:"unhealthy_count"`
	ErrorCount     int              `json:"error_count"`
	AllHealthy     bool             `json:"all_healthy"`
	AllSpeedsSame  bool             `json:"all_speeds_same"`
	CheckPassed    bool             `json:"check_passed"`
	Results        []PrecheckResult `json:"results"`
	SpeedStats     map[string]int   `json:"speed_stats"`    // 速度统计
	FwVerStats     map[string]int   `json:"fw_ver_stats"`   // 固件版本统计
	BoardIdStats   map[string]int   `json:"board_id_stats"` // 板卡ID统计
}

// ExecPrecheck 执行 precheck 并返回结构化数据（用于 API）
func ExecPrecheck(cfg *config.Config) (*PrecheckSummary, error) {
	// 收集所有需要检查的主机和HCA
	var checkItems []struct {
		hostname string
		hca      string
	}

	// 添加服务器端的HCA
	for _, hostname := range cfg.Server.Hostname {
		for _, hca := range cfg.Server.Hca {
			checkItems = append(checkItems, struct {
				hostname string
				hca      string
			}{hostname, hca})
		}
	}

	// 添加客户端的HCA
	for _, hostname := range cfg.Client.Hostname {
		for _, hca := range cfg.Client.Hca {
			checkItems = append(checkItems, struct {
				hostname string
				hca      string
			}{hostname, hca})
		}
	}

	if len(checkItems) == 0 {
		return nil, fmt.Errorf("no HCAs configured in config file")
	}

	// 并发执行检查
	results := precheckAllHCAs(checkItems, cfg.SSH.PrivateKey, cfg.SSH.User)

	// 统计信息
	summary := &PrecheckSummary{
		TotalHCAs:    len(results),
		Results:      results,
		SpeedStats:   make(map[string]int),
		FwVerStats:   make(map[string]int),
		BoardIdStats: make(map[string]int),
	}

	// 统计各项数据
	for _, result := range results {
		if result.Error != "" {
			summary.ErrorCount++
		} else if result.IsHealthy {
			summary.HealthyCount++
		} else {
			summary.UnhealthyCount++
		}

		// 统计 speed
		if result.Speed != "" {
			summary.SpeedStats[result.Speed]++
		}

		// 统计 fw_ver
		if result.FwVer != "" {
			summary.FwVerStats[result.FwVer]++
		}

		// 统计 board_id
		if result.BoardId != "" {
			summary.BoardIdStats[result.BoardId]++
		}
	}

	// 检查是否所有HCA都健康
	summary.AllHealthy = (summary.HealthyCount == summary.TotalHCAs)

	// 检查所有 speed 是否相同
	summary.AllSpeedsSame = (len(summary.SpeedStats) <= 1)

	// 总体检查结果
	summary.CheckPassed = summary.AllHealthy && summary.AllSpeedsSame

	return summary, nil
}
