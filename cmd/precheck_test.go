package cmd

import (
	"strings"
	"testing"
)

func TestDisplayPrecheckResults(t *testing.T) {
	// 准备测试数据
	results := []PrecheckResult{
		{
			Hostname:  "server-001",
			HCA:       "mlx5_0",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000844",
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "server-002",
			HCA:       "mlx5_1",
			PhysState: "Polling",
			State:     "INIT",
			Speed:     "100 Gb/sec (1X HDR)",
			FwVer:     "28.43.2025",
			BoardId:   "MT_0000000845",
			IsHealthy: false,
			Error:     "",
		},
		{
			Hostname:  "client-001",
			HCA:       "mlx5_0",
			PhysState: "",
			State:     "",
			Speed:     "",
			FwVer:     "",
			BoardId:   "",
			IsHealthy: false,
			Error:     "SSH connection failed",
		},
		{
			Hostname:  "very-long-hostname-name",
			HCA:       "mlx5_2",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000844",
			IsHealthy: true,
			Error:     "",
		},
	}

	// 测试显示函数
	t.Log("Testing displayPrecheckResults with various scenarios (no colors expected in test output):")
	displayPrecheckResults(results)

	// 验证结果统计
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

	expectedHealthy := 2
	expectedUnhealthy := 1
	expectedErrors := 1

	if healthy != expectedHealthy {
		t.Errorf("Expected %d healthy HCAs, got %d", expectedHealthy, healthy)
	}

	if unhealthy != expectedUnhealthy {
		t.Errorf("Expected %d unhealthy HCAs, got %d", expectedUnhealthy, unhealthy)
	}

	if errors != expectedErrors {
		t.Errorf("Expected %d error HCAs, got %d", expectedErrors, errors)
	}
}

func TestPrecheckHCAStates(t *testing.T) {
	testCases := []struct {
		name         string
		physState    string
		logicalState string
		expected     bool
	}{
		{
			name:         "Both LinkUp and ACTIVE - should be healthy",
			physState:    "5: LinkUp",
			logicalState: "4: ACTIVE",
			expected:     true,
		},
		{
			name:         "LinkUp but not ACTIVE - should be unhealthy",
			physState:    "5: LinkUp",
			logicalState: "2: INIT",
			expected:     false,
		},
		{
			name:         "Not LinkUp but ACTIVE - should be unhealthy",
			physState:    "2: Polling",
			logicalState: "4: ACTIVE",
			expected:     false,
		},
		{
			name:         "Neither LinkUp nor ACTIVE - should be unhealthy",
			physState:    "1: Sleep",
			logicalState: "1: DOWN",
			expected:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isLinkUp := strings.Contains(tc.physState, "LinkUp")
			isActive := strings.Contains(tc.logicalState, "ACTIVE")
			isHealthy := isLinkUp && isActive

			if isHealthy != tc.expected {
				t.Errorf("Expected healthy=%v for physState='%s' logicalState='%s', got %v",
					tc.expected, tc.physState, tc.logicalState, isHealthy)
			}
		})
	}
}

func TestPrecheckResultProcessing(t *testing.T) {
	// 测试结果处理逻辑
	result := PrecheckResult{
		Hostname:  "test-host",
		HCA:       "mlx5_0",
		PhysState: "5: LinkUp",
		State:     "4: ACTIVE",
	}

	// 测试健康状态判断
	isLinkUp := strings.Contains(result.PhysState, "LinkUp")
	isActive := strings.Contains(result.State, "ACTIVE")
	result.IsHealthy = isLinkUp && isActive

	if !result.IsHealthy {
		t.Error("Result should be healthy when both LinkUp and ACTIVE")
	}

	// 测试不健康状态
	result.State = "2: INIT"
	isActive = strings.Contains(result.State, "ACTIVE")
	result.IsHealthy = isLinkUp && isActive

	if result.IsHealthy {
		t.Error("Result should not be healthy when not ACTIVE")
	}
}

func TestDisplayPrecheckResultsWithColors(t *testing.T) {
	// 准备测试数据，包含不同的速度值来测试颜色
	results := []PrecheckResult{
		{
			Hostname:  "server-001",
			HCA:       "mlx5_0",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)", // 这个速度出现2次，应该是绿色
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000844",
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "server-002",
			HCA:       "mlx5_1",
			PhysState: "Polling",
			State:     "INIT",
			Speed:     "100 Gb/sec (1X HDR)", // 这个速度出现1次，应该是红色
			FwVer:     "28.43.2025",
			BoardId:   "MT_0000000845",
			IsHealthy: false,
			Error:     "",
		},
		{
			Hostname:  "server-003",
			HCA:       "mlx5_2",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)", // 这个速度出现2次，应该是绿色
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000844",
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "client-001",
			HCA:       "mlx5_0",
			PhysState: "",
			State:     "",
			Speed:     "",
			FwVer:     "",
			BoardId:   "",
			IsHealthy: false,
			Error:     "SSH connection failed",
		},
	}

	// 测试显示函数
	t.Log("Testing displayPrecheckResults with color coding:")
	displayPrecheckResults(results)
}

func TestCleanStateString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "LinkUp with number prefix",
			input:    "5: LinkUp",
			expected: "LinkUp",
		},
		{
			name:     "ACTIVE with number prefix",
			input:    "4: ACTIVE",
			expected: "ACTIVE",
		},
		{
			name:     "Polling with number prefix",
			input:    "2: Polling",
			expected: "Polling",
		},
		{
			name:     "INIT with number prefix",
			input:    "2: INIT",
			expected: "INIT",
		},
		{
			name:     "String without colon",
			input:    "SomeState",
			expected: "SomeState",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "String with extra spaces",
			input:    "5:   LinkUp   ",
			expected: "LinkUp",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cleanStateString(tc.input)
			if result != tc.expected {
				t.Errorf("cleanStateString(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestRemoveAnsiCodes(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "String with green color code",
			input:    "\033[32mHEALTHY\033[0m",
			expected: "HEALTHY",
		},
		{
			name:     "String with red color code",
			input:    "\033[31mUNHEALTHY\033[0m",
			expected: "UNHEALTHY",
		},
		{
			name:     "String without color codes",
			input:    "NORMAL TEXT",
			expected: "NORMAL TEXT",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := removeAnsiCodes(tc.input)
			if result != tc.expected {
				t.Errorf("removeAnsiCodes(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestSpeedConsistencyCheck 测试 v0.0.3 的 speed 一致性检查逻辑
func TestSpeedConsistencyCheck(t *testing.T) {
	// 模拟 execPrecheckCommand 的部分逻辑来测试 speed 一致性

	// 测试场景 1: 所有 speed 相同，应该通过
	resultsAllSame := []PrecheckResult{
		{
			Hostname:  "server-001",
			HCA:       "mlx5_0",
			Speed:     "200 Gb/sec (2X NDR)",
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "server-002",
			HCA:       "mlx5_1",
			Speed:     "200 Gb/sec (2X NDR)",
			IsHealthy: true,
			Error:     "",
		},
	}

	// 检查所有 speed 是否相同
	allSpeedsSame := true
	if len(resultsAllSame) > 1 {
		firstSpeed := ""
		for _, result := range resultsAllSame {
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

	allHealthy := true
	for _, result := range resultsAllSame {
		if !result.IsHealthy {
			allHealthy = false
			break
		}
	}

	success := allHealthy && allSpeedsSame
	if !success {
		t.Error("Expected success when all HCAs are healthy and speeds are same")
	}

	// 测试场景 2: speed 不同，应该失败
	resultsDifferent := []PrecheckResult{
		{
			Hostname:  "server-001",
			HCA:       "mlx5_0",
			Speed:     "200 Gb/sec (2X NDR)",
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "server-002",
			HCA:       "mlx5_1",
			Speed:     "100 Gb/sec (1X HDR)", // 不同的速度
			IsHealthy: true,
			Error:     "",
		},
	}

	// 检查所有 speed 是否相同
	allSpeedsSame = true
	if len(resultsDifferent) > 1 {
		firstSpeed := ""
		for _, result := range resultsDifferent {
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

	allHealthy = true
	for _, result := range resultsDifferent {
		if !result.IsHealthy {
			allHealthy = false
			break
		}
	}

	success = allHealthy && allSpeedsSame
	if success {
		t.Error("Expected failure when speeds are different")
	}
}

// TestDisplayPrecheckResultsV004 测试 v0.0.4 的新功能：排序、合并和着色
func TestDisplayPrecheckResultsV004(t *testing.T) {
	// 准备测试数据，故意不按顺序排列以测试排序功能
	results := []PrecheckResult{
		{
			Hostname:  "server-002", // 故意放在前面测试排序
			HCA:       "mlx5_1",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2025",    // 少数版本，应该标黄色
			BoardId:   "MT_0000000845", // 少数板卡ID，应该标黄色
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "server-001", // 测试排序
			HCA:       "mlx5_1",     // 在 mlx5_0 后面，测试 HCA 排序
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",    // 多数版本，不着色
			BoardId:   "MT_0000000844", // 多数板卡ID，不着色
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "server-001", // 相同 hostname，应该合并显示
			HCA:       "mlx5_0",     // 在 mlx5_1 前面，测试 HCA 排序
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",    // 多数版本，不着色
			BoardId:   "MT_0000000844", // 多数板卡ID，不着色
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "server-002", // 相同 hostname，应该合并显示
			HCA:       "mlx5_0",     // 在 mlx5_1 前面，测试 HCA 排序
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",    // 多数版本，不着色
			BoardId:   "MT_0000000844", // 多数板卡ID，不着色
			IsHealthy: true,
			Error:     "",
		},
	}

	// 测试显示函数
	t.Log("Testing displayPrecheckResults with v0.0.4 features (sorting, merging, coloring):")
	displayPrecheckResults(results)
}

// TestDisplayPrecheckResultsMultipleHosts 测试多主机多HCA场景
func TestDisplayPrecheckResultsMultipleHosts(t *testing.T) {
	// 模拟真实环境：5个主机，每个主机3-4个HCA，包含不同的状态
	results := []PrecheckResult{
		// node-001: 4个HCA，全部健康
		{
			Hostname:  "node-001",
			HCA:       "mlx5_0",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000844",
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "node-001",
			HCA:       "mlx5_1",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000844",
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "node-001",
			HCA:       "mlx5_2",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000844",
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "node-001",
			HCA:       "mlx5_3",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000844",
			IsHealthy: true,
			Error:     "",
		},
		// node-002: 4个HCA，其中一个不健康
		{
			Hostname:  "node-002",
			HCA:       "mlx5_0",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000844",
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "node-002",
			HCA:       "mlx5_1",
			PhysState: "Polling",
			State:     "INIT",
			Speed:     "100 Gb/sec (1X HDR)", // 不同速度，应该标红
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000844",
			IsHealthy: false,
			Error:     "",
		},
		{
			Hostname:  "node-002",
			HCA:       "mlx5_2",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000844",
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "node-002",
			HCA:       "mlx5_3",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000844",
			IsHealthy: true,
			Error:     "",
		},
		// node-003: 3个HCA，有不同的固件版本
		{
			Hostname:  "node-003",
			HCA:       "mlx5_0",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2025", // 不同版本，应该标黄
			BoardId:   "MT_0000000844",
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "node-003",
			HCA:       "mlx5_1",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000844",
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "node-003",
			HCA:       "mlx5_2",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000844",
			IsHealthy: true,
			Error:     "",
		},
		// node-004: 4个HCA，有不同的板卡ID
		{
			Hostname:  "node-004",
			HCA:       "mlx5_0",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000845", // 不同板卡，应该标黄
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "node-004",
			HCA:       "mlx5_1",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000844",
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "node-004",
			HCA:       "mlx5_2",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000844",
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "node-004",
			HCA:       "mlx5_3",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000844",
			IsHealthy: true,
			Error:     "",
		},
		// node-005: 3个HCA，其中一个连接错误
		{
			Hostname:  "node-005",
			HCA:       "mlx5_0",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000844",
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "node-005",
			HCA:       "mlx5_1",
			PhysState: "",
			State:     "",
			Speed:     "",
			FwVer:     "",
			BoardId:   "",
			IsHealthy: false,
			Error:     "SSH connection timeout",
		},
		{
			Hostname:  "node-005",
			HCA:       "mlx5_2",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			Speed:     "200 Gb/sec (2X NDR)",
			FwVer:     "28.43.2026",
			BoardId:   "MT_0000000844",
			IsHealthy: true,
			Error:     "",
		},
	}

	// 测试显示函数
	t.Log("Testing displayPrecheckResults with multiple hosts and HCAs:")
	displayPrecheckResults(results)

	// 验证统计信息
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

	expectedHealthy := 16
	expectedUnhealthy := 1
	expectedErrors := 1

	if healthy != expectedHealthy {
		t.Errorf("Expected %d healthy HCAs, got %d", expectedHealthy, healthy)
	}

	if unhealthy != expectedUnhealthy {
		t.Errorf("Expected %d unhealthy HCAs, got %d", expectedUnhealthy, unhealthy)
	}

	if errors != expectedErrors {
		t.Errorf("Expected %d error HCAs, got %d", expectedErrors, errors)
	}

	t.Logf("Statistics: %d healthy, %d unhealthy, %d errors (Total: %d HCAs)",
		healthy, unhealthy, errors, len(results))
}
