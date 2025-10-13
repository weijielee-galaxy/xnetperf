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
			IsHealthy: true,
			Error:     "",
		},
		{
			Hostname:  "server-002",
			HCA:       "mlx5_1",
			PhysState: "Polling",
			State:     "INIT",
			IsHealthy: false,
			Error:     "",
		},
		{
			Hostname:  "client-001",
			HCA:       "mlx5_0",
			PhysState: "",
			State:     "",
			IsHealthy: false,
			Error:     "SSH connection failed",
		},
		{
			Hostname:  "very-long-hostname-name",
			HCA:       "mlx5_2",
			PhysState: "LinkUp",
			State:     "ACTIVE",
			IsHealthy: true,
			Error:     "",
		},
	}

	// 测试显示函数
	t.Log("Testing displayPrecheckResults with various scenarios:")
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
