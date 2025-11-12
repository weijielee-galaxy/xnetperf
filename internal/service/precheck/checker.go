package precheck

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
	"xnetperf/config"
	"xnetperf/internal/tools"
	"xnetperf/pkg/tools/logger"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/samber/lo"
)

type checker struct {
	cfg    *config.Config
	logger *slog.Logger
}

func New(cfg *config.Config) *checker {
	return &checker{
		cfg:    cfg,
		logger: logger.GetLogger().With("module", "PRECHECK"),
	}
}

func (c *checker) DoCheck() []PrecheckResult {
	// 1. 解析配置文件，获取所有主机和HCA信息
	// 收集所有需要检查的主机和HCA，支持去重
	hostHCAs := c.buildHostHCAs()
	if len(hostHCAs) == 0 {
		return []PrecheckResult{{Error: "No hosts configured in config file"}}
	}

	// 2. 生成每个host上要执行的命令
	hostCommands := c.buildHostCommands(hostHCAs)

	// 3. 并发执行命令，收集结果（DTO层）
	hostDataList := c.execPrecheckCommands(hostCommands)

	// 4. 转换为展示层数据
	results := c.convertHostDataToResults(hostDataList)

	return results
}

// 1. 收集所有需要检查的主机和HCA
func (c *checker) buildHostHCAs() map[string][]string {
	hostHCAs := make(map[string][]string)
	// 添加服务器端的主机和HCA
	for _, hostname := range c.cfg.Server.Hostname {
		if _, exists := hostHCAs[hostname]; !exists {
			hostHCAs[hostname] = make([]string, 0)
		}
		// 合并HCA列表并去重
		hostHCAs[hostname] = lo.Union(hostHCAs[hostname], c.cfg.Server.Hca)
	}

	// 添加客户端的主机和HCA
	for _, hostname := range c.cfg.Client.Hostname {
		if _, exists := hostHCAs[hostname]; !exists {
			hostHCAs[hostname] = make([]string, 0)
		}
		// 合并HCA列表并去重
		hostHCAs[hostname] = lo.Union(hostHCAs[hostname], c.cfg.Client.Hca)
	}

	return hostHCAs
}

// 2. 生成每个host上要执行的命令,使用JSON格式输出便于解析
/*
	2.1 每个host有多个HCA需要检查
	2.2 每个HCA有多个检查命令
	2.3 使用JSON格式输出,便于解析
	2.4 合并命令为一条SSH命令,减少SSH连接次数
*/
func (c *checker) buildHostCommands(hostHCAs map[string][]string) map[string]string {
	hostCommands := make(map[string]string)
	for hostname, hcas := range hostHCAs {
		// 构建JSON输出的shell脚本
		var jsonBuilder strings.Builder
		jsonBuilder.WriteString(`echo "{`)

		// 添加 serial number
		jsonBuilder.WriteString(`\"serial\":\"$(cat /sys/class/dmi/id/product_serial 2>/dev/null || echo ERROR)\",`)

		// 添加 HCAs 数组
		jsonBuilder.WriteString(`\"hcas\":[`)

		for i, hca := range hcas {
			if i > 0 {
				jsonBuilder.WriteString(`,`)
			}
			jsonBuilder.WriteString(fmt.Sprintf(`{\"name\":\"%s\",`, hca))
			jsonBuilder.WriteString(fmt.Sprintf(`\"phys_state\":\"$(cat /sys/class/infiniband/%s/ports/1/phys_state 2>/dev/null || echo ERROR)\",`, hca))
			jsonBuilder.WriteString(fmt.Sprintf(`\"state\":\"$(cat /sys/class/infiniband/%s/ports/1/state 2>/dev/null || echo ERROR)\",`, hca))
			jsonBuilder.WriteString(fmt.Sprintf(`\"speed\":\"$(cat /sys/class/infiniband/%s/ports/1/rate 2>/dev/null || echo ERROR)\",`, hca))
			jsonBuilder.WriteString(fmt.Sprintf(`\"fw_ver\":\"$(cat /sys/class/infiniband/%s/fw_ver 2>/dev/null || echo ERROR)\",`, hca))
			jsonBuilder.WriteString(fmt.Sprintf(`\"board_id\":\"$(cat /sys/class/infiniband/%s/board_id 2>/dev/null || echo ERROR)\"`, hca))
			jsonBuilder.WriteString(`}`)
		}

		jsonBuilder.WriteString(`]}"`)

		hostCommands[hostname] = jsonBuilder.String()
	}

	return hostCommands
}

// HCAData 单个 HCA 的检查数据（从 JSON 解析）
type HCAData struct {
	Name      string `json:"name"`
	PhysState string `json:"phys_state"`
	State     string `json:"state"`
	Speed     string `json:"speed"`
	FwVer     string `json:"fw_ver"`
	BoardId   string `json:"board_id"`
}

// HasError 检查 HCA 的任何字段是否包含 ERROR
func (h *HCAData) HasError() bool {
	return h.PhysState == "ERROR" || h.State == "ERROR" || h.Speed == "ERROR" ||
		h.FwVer == "ERROR" || h.BoardId == "ERROR"
}

// IsHealthy 检查 HCA 是否健康（LinkUp 且 ACTIVE）
func (h *HCAData) IsHealthy() bool {
	return strings.Contains(h.PhysState, "LinkUp") && strings.Contains(h.State, "ACTIVE")
}

// CleanPhysState 去掉 PhysState 字段前面的数字和冒号
// 例如: "5: LinkUp" -> "LinkUp"
func (h *HCAData) CleanPhysState() string {
	return cleanStateString(h.PhysState)
}

// CleanState 去掉 State 字段前面的数字和冒号
// 例如: "4: ACTIVE" -> "ACTIVE"
func (h *HCAData) CleanState() string {
	return cleanStateString(h.State)
}

// HostPrecheckData 单个主机的 precheck 数据（解析后）
type HostPrecheckData struct {
	Hostname string     `json:"hostname"`
	Serial   string     `json:"serial"`
	HCAs     []*HCAData `json:"hcas"`
	Error    string     `json:"error,omitempty"`
}

// ProcessSerialNumber 处理 Serial Number：如果包含 - 则按 - 分割获取最后一个值
func (h *HostPrecheckData) ProcessSerialNumber() {
	if strings.Contains(h.Serial, "-") {
		parts := strings.Split(h.Serial, "-")
		if len(parts) > 0 {
			h.Serial = parts[len(parts)-1]
		}
	}
}

// ValidateSerialNumber 检查 Serial Number 是否有效
func (h *HostPrecheckData) ValidateSerialNumber() {
	if h.Serial == "ERROR" {
		h.Error = "Failed to read serial number"
	}
}

// ValidateHCAs 检查 HCA 数据中是否有 ERROR（使用 HCAData 的面向对象方法）
func (h *HostPrecheckData) ValidateHCAs() {
	for _, hca := range h.HCAs {
		if hca.HasError() {
			if h.Error == "" {
				h.Error = fmt.Sprintf("Failed to read some HCA attributes for %s", hca.Name)
			}
			break
		}
	}
}

// 3. 并发执行命令，收集结果并解析 JSON
func (c *checker) execPrecheckCommands(hostCommands map[string]string) []*HostPrecheckData {
	var results []*HostPrecheckData
	var mu sync.Mutex
	var wg sync.WaitGroup

	for hostname, command := range hostCommands {
		wg.Add(1)
		go func(host, cmd string) {
			defer wg.Done()

			// 执行单个主机的命令并解析结果
			hostData := c.execAndParseHostCommand(host, cmd)

			mu.Lock()
			results = append(results, hostData)
			mu.Unlock()
		}(hostname, command)
	}

	wg.Wait()
	return results
}

// convertHostDataToResults 将 HostPrecheckData（DTO）转换为 PrecheckResult（展示层）
// 遵循 DTO 设计思想：数据的生产和展示分离
func (c *checker) convertHostDataToResults(hostDataList []*HostPrecheckData) []PrecheckResult {
	var results []PrecheckResult

	for _, hostData := range hostDataList {
		// 如果主机级别有错误，为该主机创建一条错误记录
		if hostData.Error != "" && len(hostData.HCAs) == 0 {
			results = append(results, PrecheckResult{
				Hostname: hostData.Hostname,
				Error:    hostData.Error,
			})
			continue
		}

		// 为每个 HCA 创建一条 PrecheckResult 记录
		for _, hca := range hostData.HCAs {
			result := PrecheckResult{
				Hostname:     hostData.Hostname,
				HCA:          hca.Name,
				PhysState:    hca.CleanPhysState(), // 使用 HCAData 的面向对象方法
				State:        hca.CleanState(),     // 使用 HCAData 的面向对象方法
				Speed:        hca.Speed,
				FwVer:        hca.FwVer,
				BoardId:      hca.BoardId,
				SerialNumber: hostData.Serial,
				IsHealthy:    hca.IsHealthy(), // 使用 HCAData 的面向对象方法
			}

			// 如果有错误信息，记录到 Error 字段
			if hostData.Error != "" {
				result.Error = hostData.Error
			}

			results = append(results, result)
		}
	}

	return results
}

// execAndParseHostCommand 执行单个主机的命令并解析 JSON 输出
func (c *checker) execAndParseHostCommand(hostname, command string) *HostPrecheckData {
	result := &HostPrecheckData{
		Hostname: hostname,
	}

	// 构建 SSH 命令
	sshWrapper := tools.NewSSHWrapper(hostname, "'").PrivateKey(c.cfg.SSH.PrivateKey).User(c.cfg.SSH.User).Command(command)

	c.logger.Debug("Executing precheck command", slog.String("ssh_command", sshWrapper.String()))
	cmd := exec.Command("bash", "-c", sshWrapper.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		c.logger.Error("Precheck command execution failed", slog.String("ssh_command", sshWrapper.String()), slog.Any("error", err))
		result.Error = fmt.Sprintf("SSH execution failed: %v", err)
		return result
	}

	var SerialData struct {
		Serial string     `json:"serial"`
		HCAs   []*HCAData `json:"hcas"`
	}

	err = json.Unmarshal([]byte(strings.TrimSpace(string(output))), &SerialData)
	if err != nil {
		c.logger.Error("JSON parse error", slog.String("ssh_command", sshWrapper.String()), slog.Any("error", err))
		result.Error = fmt.Sprintf("Failed to parse JSON output: %v. Output: %s", err, output)
		return result
	}

	// 提取数据
	result.Serial = SerialData.Serial
	result.HCAs = SerialData.HCAs

	// 使用面向对象方法处理和验证数据
	result.ValidateSerialNumber()
	result.ProcessSerialNumber()
	result.ValidateHCAs()

	c.logger.Debug("Parsed precheck data", slog.String("hostname", hostname), slog.Any("data", result))
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
func (c *checker) DoCheckForAPI(cfg *config.Config) (*PrecheckSummary, error) {
	// 复用 Precheck 函数获取结果
	results := c.DoCheck()

	if len(results) == 0 {
		return nil, fmt.Errorf("no HCAs configured in config file")
	}

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

// DisplayPrecheckResultsV2 使用新的展示层DTO（清晰分离着色和展示逻辑）
func (c *checker) Display(results []PrecheckResult) {
	fmt.Printf("=== Precheck Results - %s ===\n\n", time.Now().Format("15:04:05"))

	// 1. 创建展示数据（应用着色规则）
	displayData := NewPrecheckDisplayData(results)
	displayData.SortByHostAndHCA()

	// 2. 创建表格
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)

	// 3. 设置表头
	t.AppendHeader(table.Row{
		"Serial Number",
		"Hostname",
		"HCA",
		"Physical State",
		"Logical State",
		"Speed",
		"FW Version",
		"Board ID",
		"Status",
	})

	// 4. 填充数据（只负责展示，着色已在DTO中完成）
	lastHostname := ""
	for _, item := range displayData.Items {
		// 实现 hostname 合并效果
		displayHostname := item.Hostname
		if item.Hostname == lastHostname {
			displayHostname = ""
		} else {
			if lastHostname != "" {
				t.AppendSeparator()
			}
			lastHostname = item.Hostname
		}

		// 添加行数据（应用颜色）
		t.AppendRow(table.Row{
			item.SerialNumber,
			displayHostname,
			item.HCA,
			item.PhysState,
			item.State,
			item.Speed.ApplyColor(),   // 应用颜色
			item.FwVer.ApplyColor(),   // 应用颜色
			item.BoardId.ApplyColor(), // 应用颜色
			item.Status.ApplyColor(),  // 应用颜色
		})
	}

	// 5. 渲染表格
	t.Render()

	// 6. 显示统计信息（使用DTO中的统计数据）
	fmt.Printf("\nSummary: %s%d healthy%s, %s%d unhealthy%s, %s%d errors%s (Total: %d HCAs)\n",
		ColorGreen, displayData.HealthyCount, ColorReset,
		ColorRed, displayData.UnhealthyCount, ColorReset,
		ColorYellow, displayData.ErrorCount, ColorReset,
		displayData.TotalCount)
}
