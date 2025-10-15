package precheck

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"xnetperf/config"
)

// PrecheckResult 表示预检查结果
type PrecheckResult struct {
	Hostname  string `json:"hostname"`
	HCA       string `json:"hca"`
	PhysState string `json:"phys_state"`
	State     string `json:"state"`
	Speed     string `json:"speed"`
	FwVer     string `json:"fw_ver"`
	BoardId   string `json:"board_id"`
	IsHealthy bool   `json:"is_healthy"`
	Error     string `json:"error"`
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

// Execute 执行 precheck 并返回结构化数据
func Execute(cfg *config.Config) (*PrecheckSummary, error) {
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
	results := checkAllHCAs(checkItems)

	// 按 hostname 然后按 HCA 排序
	sort.Slice(results, func(i, j int) bool {
		if results[i].Hostname != results[j].Hostname {
			return results[i].Hostname < results[j].Hostname
		}
		return results[i].HCA < results[j].HCA
	})

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

func checkAllHCAs(checkItems []struct {
	hostname string
	hca      string
}) []PrecheckResult {
	var results []PrecheckResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, item := range checkItems {
		wg.Add(1)
		go func(hostname, hca string) {
			defer wg.Done()

			result := PrecheckResult{
				Hostname: hostname,
				HCA:      hca,
			}

			// 检查物理状态
			physState, err := getHCAInfo(hostname, hca, "phys_state")
			if err != nil {
				result.Error = fmt.Sprintf("Failed to get phys_state: %v", err)
				fmt.Printf("Error: %s/%s - %s\n", hostname, hca, result.Error)
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}
			result.PhysState = strings.TrimSpace(cleanStateString(physState))

			// 检查逻辑状态
			state, err := getHCAInfo(hostname, hca, "state")
			if err != nil {
				result.Error = fmt.Sprintf("Failed to get state: %v", err)
				fmt.Printf("Error: %s/%s - %s\n", hostname, hca, result.Error)
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}
			result.State = strings.TrimSpace(cleanStateString(state))

			// v0.0.2: 检查 speed
			speed, err := getHCAInfo(hostname, hca, "rate")
			if err != nil {
				result.Error = fmt.Sprintf("Failed to get rate: %v", err)
				fmt.Printf("Error: %s/%s - %s\n", hostname, hca, result.Error)
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
				return
			}
			result.Speed = strings.TrimSpace(speed)

			// v0.0.3: 检查 FW Version（直接在 HCA 根目录）
			fwVer, err := getHCAInfoRoot(hostname, hca, "fw_ver")
			if err != nil {
				// 不中断，只记录警告
				result.FwVer = "N/A"
				fmt.Printf("Warning: Failed to get fw_ver for %s/%s: %v\n", hostname, hca, err)
			} else {
				result.FwVer = strings.TrimSpace(fwVer)
			}

			// v0.0.3: 检查 Board ID（直接在 HCA 根目录）
			boardId, err := getHCAInfoRoot(hostname, hca, "board_id")
			if err != nil {
				// 不中断，只记录警告
				result.BoardId = "N/A"
				fmt.Printf("Warning: Failed to get board_id for %s/%s: %v\n", hostname, hca, err)
			} else {
				result.BoardId = strings.TrimSpace(boardId)
			}

			// 判断是否健康
			result.IsHealthy = strings.Contains(result.PhysState, "LinkUp") &&
				strings.Contains(result.State, "ACTIVE")

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(item.hostname, item.hca)
	}

	wg.Wait()
	return results
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

func getHCAInfo(hostname, hca, infoType string) (string, error) {
	path := fmt.Sprintf("/sys/class/infiniband/%s/ports/1/%s", hca, infoType)
	cmd := exec.Command("ssh", hostname, "cat", path)

	// 设置超时
	timer := time.AfterFunc(10*time.Second, func() {
		cmd.Process.Kill()
	})
	defer timer.Stop()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ssh command failed: %v, output: %s", err, string(output))
	}

	return string(output), nil
}

// getHCAInfoRoot 获取 HCA 根目录的信息（如 fw_ver, board_id）
func getHCAInfoRoot(hostname, hca, infoType string) (string, error) {
	path := fmt.Sprintf("/sys/class/infiniband/%s/%s", hca, infoType)
	cmd := exec.Command("ssh", hostname, "cat", path)

	// 设置超时
	timer := time.AfterFunc(10*time.Second, func() {
		cmd.Process.Kill()
	})
	defer timer.Stop()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ssh command failed: %v, output: %s", err, string(output))
	}

	return string(output), nil
}
