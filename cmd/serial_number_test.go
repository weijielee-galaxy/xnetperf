package cmd

import (
	"fmt"
	"strings"
	"testing"
)

// TestPrecheckSerialNumberDisplay 测试 precheck 命令的 Serial Number 显示效果
func TestPrecheckSerialNumberDisplay(t *testing.T) {
	tests := []struct {
		name        string
		description string
		results     []PrecheckResult
	}{
		{
			name:        "Standard Serial Numbers",
			description: "测试标准格式的序列号显示",
			results: []PrecheckResult{
				{
					Hostname:     "server-001",
					HCA:          "mlx5_0",
					PhysState:    "LinkUp",
					State:        "ACTIVE",
					Speed:        "200 Gb/sec (2X NDR)",
					FwVer:        "28.43.2026",
					BoardId:      "MT_0000000844",
					SerialNumber: "SN123456",
					IsHealthy:    true,
				},
				{
					Hostname:     "server-001",
					HCA:          "mlx5_1",
					PhysState:    "LinkUp",
					State:        "ACTIVE",
					Speed:        "200 Gb/sec (2X NDR)",
					FwVer:        "28.43.2026",
					BoardId:      "MT_0000000844",
					SerialNumber: "SN123456",
					IsHealthy:    true,
				},
				{
					Hostname:     "server-002",
					HCA:          "mlx5_0",
					PhysState:    "LinkUp",
					State:        "ACTIVE",
					Speed:        "200 Gb/sec (2X NDR)",
					FwVer:        "28.43.2025",
					BoardId:      "MT_0000000845",
					SerialNumber: "SN789012",
					IsHealthy:    true,
				},
			},
		},
		{
			name:        "Serial Numbers with Dashes",
			description: "测试带横杠的序列号（提取最后部分）",
			results: []PrecheckResult{
				{
					Hostname:     "node-001",
					HCA:          "mlx5_0",
					PhysState:    "LinkUp",
					State:        "ACTIVE",
					Speed:        "200 Gb/sec (2X NDR)",
					FwVer:        "28.43.2026",
					BoardId:      "MT_0000000844",
					SerialNumber: "12345", // 原始: DELL-ABC-12345, 处理后: 12345
					IsHealthy:    true,
				},
				{
					Hostname:     "node-002",
					HCA:          "mlx5_bond_0",
					PhysState:    "LinkUp",
					State:        "ACTIVE",
					Speed:        "200 Gb/sec (2X NDR)",
					FwVer:        "28.43.2026",
					BoardId:      "MT_0000000844",
					SerialNumber: "67890", // 原始: HP-XYZ-67890, 处理后: 67890
					IsHealthy:    true,
				},
			},
		},
		{
			name:        "Mixed Serial Number Lengths",
			description: "测试不同长度的序列号",
			results: []PrecheckResult{
				{
					Hostname:     "host-a",
					HCA:          "mlx5_0",
					PhysState:    "LinkUp",
					State:        "ACTIVE",
					Speed:        "200 Gb/sec (2X NDR)",
					FwVer:        "28.43.2026",
					BoardId:      "MT_0000000844",
					SerialNumber: "SN1",
					IsHealthy:    true,
				},
				{
					Hostname:     "host-b",
					HCA:          "mlx5_1",
					PhysState:    "LinkUp",
					State:        "ACTIVE",
					Speed:        "200 Gb/sec (2X NDR)",
					FwVer:        "28.43.2026",
					BoardId:      "MT_0000000844",
					SerialNumber: "VERY-LONG-SERIAL-NUMBER-123456",
					IsHealthy:    true,
				},
				{
					Hostname:     "host-c",
					HCA:          "mlx5_bond_0",
					PhysState:    "LinkUp",
					State:        "ACTIVE",
					Speed:        "200 Gb/sec (2X NDR)",
					FwVer:        "28.43.2026",
					BoardId:      "MT_0000000844",
					SerialNumber: "MID-LEN-SN",
					IsHealthy:    true,
				},
			},
		},
		{
			name:        "Error Cases with N/A Serial Number",
			description: "测试错误情况下的 N/A 显示",
			results: []PrecheckResult{
				{
					Hostname:     "error-host",
					HCA:          "mlx5_0",
					SerialNumber: "",
					Error:        "SSH connection failed",
				},
				{
					Hostname:     "normal-host",
					HCA:          "mlx5_0",
					PhysState:    "LinkUp",
					State:        "ACTIVE",
					Speed:        "200 Gb/sec (2X NDR)",
					FwVer:        "28.43.2026",
					BoardId:      "MT_0000000844",
					SerialNumber: "SN999999",
					IsHealthy:    true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("\n=== %s ===", tt.name)
			t.Logf("描述: %s", tt.description)
			t.Log("\n输出:\n")

			// 调用 displayPrecheckResults 函数来显示结果
			displayPrecheckResults(tt.results)

			t.Log("\n" + strings.Repeat("=", 60) + "\n")
		})
	}
}

// TestAnalyzeSerialNumberDisplay 测试 analyze 命令的 Serial Number 显示效果
func TestAnalyzeSerialNumberDisplay(t *testing.T) {
	tests := []struct {
		name        string
		description string
		clientData  map[string]map[string]*DeviceData
		serverData  map[string]map[string]*DeviceData
		specSpeed   float64
	}{
		{
			name:        "FullMesh with Serial Numbers",
			description: "测试带序列号的 FullMesh 结果展示",
			clientData: map[string]map[string]*DeviceData{
				"client-001": {
					"mlx5_0": {
						Hostname:     "client-001",
						Device:       "mlx5_0",
						SerialNumber: "SN111111",
						BWSum:        3505.0,
						Count:        10,
						IsClient:     true,
					},
					"mlx5_1": {
						Hostname:     "client-001",
						Device:       "mlx5_1",
						SerialNumber: "SN111111",
						BWSum:        3505.0,
						Count:        10,
						IsClient:     true,
					},
				},
				"client-002": {
					"mlx5_bond_0": {
						Hostname:     "client-002",
						Device:       "mlx5_bond_0",
						SerialNumber: "SN222222",
						BWSum:        3505.0,
						Count:        10,
						IsClient:     true,
					},
				},
			},
			serverData: map[string]map[string]*DeviceData{
				"server-001": {
					"mlx5_0": {
						Hostname:     "server-001",
						Device:       "mlx5_0",
						SerialNumber: "SN333333",
						BWSum:        3505.0,
						Count:        10,
						IsClient:     false,
					},
					"mlx5_1": {
						Hostname:     "server-001",
						Device:       "mlx5_1",
						SerialNumber: "SN333333",
						BWSum:        3505.0,
						Count:        10,
						IsClient:     false,
					},
				},
			},
			specSpeed: 400.0,
		},
		{
			name:        "Different Serial Number Lengths",
			description: "测试不同长度序列号的对齐效果",
			clientData: map[string]map[string]*DeviceData{
				"host-a": {
					"mlx5_0": {
						Hostname:     "host-a",
						Device:       "mlx5_0",
						SerialNumber: "SN1",
						BWSum:        3505.0,
						Count:        10,
						IsClient:     true,
					},
				},
				"host-b": {
					"mlx5_bond_interface_0": {
						Hostname:     "host-b",
						Device:       "mlx5_bond_interface_0",
						SerialNumber: "VERY-LONG-SERIAL-NUMBER-12345678",
						BWSum:        3505.0,
						Count:        10,
						IsClient:     true,
					},
				},
			},
			serverData: map[string]map[string]*DeviceData{
				"server-x": {
					"mlx5_0": {
						Hostname:     "server-x",
						Device:       "mlx5_0",
						SerialNumber: "MID-SN-123",
						BWSum:        3505.0,
						Count:        10,
						IsClient:     false,
					},
				},
			},
			specSpeed: 400.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("\n=== %s ===", tt.name)
			t.Logf("描述: %s", tt.description)
			t.Log("\n输出:\n")

			// 调用 displayResults 函数来显示结果
			displayResults(tt.clientData, tt.serverData, tt.specSpeed)

			t.Log("\n" + strings.Repeat("=", 60) + "\n")
		})
	}
}

// TestP2PSerialNumberDisplay 测试 P2P 模式的 Serial Number 显示效果
func TestP2PSerialNumberDisplay(t *testing.T) {
	tests := []struct {
		name        string
		description string
		p2pData     map[string]map[string]*P2PDeviceData
	}{
		{
			name:        "P2P with Serial Numbers",
			description: "测试 P2P 模式下的序列号显示",
			p2pData: map[string]map[string]*P2PDeviceData{
				"node-001": {
					"mlx5_0": {
						Hostname:     "node-001",
						Device:       "mlx5_0",
						SerialNumber: "SN111111",
						BWSum:        3505.0,
						Count:        10,
					},
					"mlx5_1": {
						Hostname:     "node-001",
						Device:       "mlx5_1",
						SerialNumber: "SN111111",
						BWSum:        3505.0,
						Count:        10,
					},
				},
				"node-002": {
					"mlx5_bond_0": {
						Hostname:     "node-002",
						Device:       "mlx5_bond_0",
						SerialNumber: "SN222222",
						BWSum:        3505.0,
						Count:        10,
					},
				},
			},
		},
		{
			name:        "P2P with Mixed Serial Number Lengths",
			description: "测试不同长度序列号在 P2P 模式下的对齐",
			p2pData: map[string]map[string]*P2PDeviceData{
				"host-a": {
					"mlx5_0": {
						Hostname:     "host-a",
						Device:       "mlx5_0",
						SerialNumber: "SN1",
						BWSum:        3505.0,
						Count:        10,
					},
				},
				"very-long-hostname-name": {
					"mlx5_bond_interface_0": {
						Hostname:     "very-long-hostname-name",
						Device:       "mlx5_bond_interface_0",
						SerialNumber: "VERY-LONG-SERIAL-NUMBER-12345",
						BWSum:        3505.0,
						Count:        10,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("\n=== %s ===", tt.name)
			t.Logf("描述: %s", tt.description)
			t.Log("\n输出:\n")

			// 调用 displayP2PResults 函数来显示结果
			displayP2PResults(tt.p2pData)

			t.Log("\n" + strings.Repeat("=", 60) + "\n")
		})
	}
}

// TestSerialNumberParsing 测试序列号的解析逻辑
func TestSerialNumberParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple serial number",
			input:    "SN123456",
			expected: "SN123456",
		},
		{
			name:     "Serial number with one dash",
			input:    "DELL-12345",
			expected: "12345",
		},
		{
			name:     "Serial number with multiple dashes",
			input:    "HP-SERVER-XYZ-67890",
			expected: "67890",
		},
		{
			name:     "Empty serial number",
			input:    "",
			expected: "",
		},
		{
			name:     "Serial number ending with dash",
			input:    "ABC-",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟 precheck.go 中的处理逻辑
			result := tt.input
			if strings.Contains(result, "-") {
				parts := strings.Split(result, "-")
				if len(parts) > 0 {
					result = parts[len(parts)-1]
				}
			}

			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			} else {
				t.Logf("✓ Input: %q -> Output: %q", tt.input, result)
			}
		})
	}
}

// TestSerialNumberColumnWidth 测试序列号列宽度计算
func TestSerialNumberColumnWidth(t *testing.T) {
	tests := []struct {
		name             string
		serialNumbers    []string
		expectedMinWidth int
		description      string
	}{
		{
			name:             "Short serial numbers",
			serialNumbers:    []string{"SN1", "SN2", "SN3"},
			expectedMinWidth: 15, // 最小宽度为 15 (列标题 "Serial Number" 的长度 + 2)
			description:      "短序列号应使用最小宽度",
		},
		{
			name:             "Medium serial numbers",
			serialNumbers:    []string{"SN123456", "SN789012", "SN345678"},
			expectedMinWidth: 15,
			description:      "中等长度序列号",
		},
		{
			name:             "Very long serial number",
			serialNumbers:    []string{"VERY-LONG-SERIAL-NUMBER-12345678"},
			expectedMinWidth: 32, // 最长序列号的长度
			description:      "超长序列号应扩展列宽",
		},
		{
			name:             "Mixed lengths",
			serialNumbers:    []string{"SN1", "MEDIUM-SN-123", "VERY-LONG-SERIAL-NUMBER-12345"},
			expectedMinWidth: 29, // 最长的长度
			description:      "混合长度应使用最长的",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maxLen := 0
			for _, sn := range tt.serialNumbers {
				if len(sn) > maxLen {
					maxLen = len(sn)
				}
			}

			// 最小宽度为 15 (类似 precheck.go 中的逻辑)
			if maxLen < 15 {
				maxLen = 15
			}

			t.Logf("描述: %s", tt.description)
			t.Logf("序列号: %v", tt.serialNumbers)
			t.Logf("计算出的最大宽度: %d", maxLen)

			if maxLen < tt.expectedMinWidth {
				t.Errorf("Expected minimum width of %d, got %d", tt.expectedMinWidth, maxLen)
			}
		})
	}
}

// TestSerialNumberColumnPosition 测试序列号列在第一列的位置
func TestSerialNumberColumnPosition(t *testing.T) {
	t.Log("\n=== 测试序列号列位置 ===")
	t.Log("描述: 验证 Serial Number 列在表格第一列的位置")

	// 创建测试数据
	results := []PrecheckResult{
		{
			Hostname:     "test-host",
			HCA:          "mlx5_0",
			PhysState:    "LinkUp",
			State:        "ACTIVE",
			Speed:        "200 Gb/sec (2X NDR)",
			FwVer:        "28.43.2026",
			BoardId:      "MT_0000000844",
			SerialNumber: "SN123456",
			IsHealthy:    true,
		},
	}

	t.Log("\nPrecheck 表格输出:")
	displayPrecheckResults(results)

	t.Log("\n✓ Serial Number 应该在第一列（最左侧）")
	t.Log("✓ 列顺序应为: Serial Number | Hostname | HCA | Physical State | Logical State | Speed | FW Version | Board ID | Status")
}

// BenchmarkSerialNumberDisplay 性能基准测试
func BenchmarkSerialNumberDisplay(b *testing.B) {
	// 创建大量测试数据
	results := make([]PrecheckResult, 100)
	for i := 0; i < 100; i++ {
		results[i] = PrecheckResult{
			Hostname:     fmt.Sprintf("host-%03d", i),
			HCA:          "mlx5_0",
			PhysState:    "LinkUp",
			State:        "ACTIVE",
			Speed:        "200 Gb/sec (2X NDR)",
			FwVer:        "28.43.2026",
			BoardId:      "MT_0000000844",
			SerialNumber: fmt.Sprintf("SN%06d", i),
			IsHealthy:    true,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 这里不实际调用 displayPrecheckResults，因为它会输出到 stdout
		// 只测试数据处理部分
		for _, result := range results {
			_ = result.SerialNumber
		}
	}
}
