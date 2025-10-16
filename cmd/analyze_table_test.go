package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// TestTableAlignment 测试不同长度 HCA 名称的表格对齐
func TestTableAlignment(t *testing.T) {
	tests := []struct {
		name        string
		devices     []string
		description string
	}{
		{
			name:        "Standard HCA names",
			devices:     []string{"mlx5_0", "mlx5_1"},
			description: "短 HCA 名称（8 字符以内）",
		},
		{
			name:        "Bond HCA names",
			devices:     []string{"mlx5_bond_0", "mlx5_bond_1"},
			description: "中等长度 HCA 名称（11 字符）",
		},
		{
			name:        "Mixed length HCA names",
			devices:     []string{"mlx5_0", "mlx5_bond_0", "ib0", "custom_hca_name"},
			description: "混合长度 HCA 名称",
		},
		{
			name:        "Very long HCA names",
			devices:     []string{"mlx5_bond_interface_0", "mlx5_bond_interface_1"},
			description: "超长 HCA 名称（21 字符）",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试数据
			clientData := make(map[string]map[string]*DeviceData)
			serverData := make(map[string]map[string]*DeviceData)

			hostname := "test-host"
			clientData[hostname] = make(map[string]*DeviceData)
			serverData[hostname] = make(map[string]*DeviceData)

			for _, device := range tt.devices {
				clientData[hostname][device] = &DeviceData{
					Hostname: hostname,
					Device:   device,
					BWSum:    350.5,
					Count:    1,
					IsClient: true,
				}
				serverData[hostname][device] = &DeviceData{
					Hostname: hostname,
					Device:   device,
					BWSum:    350.5,
					Count:    1,
					IsClient: false,
				}
			}

			// 捕获输出
			output := captureOutput(func() {
				displayResults(clientData, serverData, 400.0)
			})

			t.Logf("\n%s\n输出:\n%s", tt.description, output)

			// 验证表格线对齐 - 分别检查 Client 和 Server 表格
			lines := strings.Split(output, "\n")
			var clientTableLines []string
			var serverTableLines []string
			inClientTable := false
			inServerTable := false

			for _, line := range lines {
				if strings.Contains(line, "CLIENT DATA") {
					inClientTable = true
					inServerTable = false
					continue
				}
				if strings.Contains(line, "SERVER DATA") {
					inClientTable = false
					inServerTable = true
					continue
				}
				if strings.Contains(line, "Theoretical BW") {
					inClientTable = false
					continue
				}

				if strings.HasPrefix(line, "│") || strings.HasPrefix(line, "┌") ||
					strings.HasPrefix(line, "├") || strings.HasPrefix(line, "└") {
					if inClientTable {
						clientTableLines = append(clientTableLines, line)
					} else if inServerTable {
						serverTableLines = append(serverTableLines, line)
					}
				}
			}

			// 检查 Client 表格对齐
			if len(clientTableLines) > 0 {
				clientLineLen := len([]rune(clientTableLines[0]))
				for i, line := range clientTableLines {
					lineLen := len([]rune(line))
					if lineLen != clientLineLen {
						t.Errorf("CLIENT 表格第 %d 行长度不一致: 期望 %d, 实际 %d\n行内容: %s",
							i, clientLineLen, lineLen, line)
					}
				}
			}

			// 检查 Server 表格对齐
			if len(serverTableLines) > 0 {
				serverLineLen := len([]rune(serverTableLines[0]))
				for i, line := range serverTableLines {
					lineLen := len([]rune(line))
					if lineLen != serverLineLen {
						t.Errorf("SERVER 表格第 %d 行长度不一致: 期望 %d, 实际 %d\n行内容: %s",
							i, serverLineLen, lineLen, line)
					}
				}
			}

			if len(clientTableLines) == 0 && len(serverTableLines) == 0 {
				t.Fatal("没有找到表格输出")
			}

			// 验证 Client 表格每行都正确闭合
			for i, line := range clientTableLines {
				if strings.HasPrefix(line, "│") && !strings.HasSuffix(line, "│") {
					t.Errorf("CLIENT 表格第 %d 行没有正确闭合\n行内容: %s", i, line)
				}
			}

			// 验证 Server 表格每行都正确闭合
			for i, line := range serverTableLines {
				if strings.HasPrefix(line, "│") && !strings.HasSuffix(line, "│") {
					t.Errorf("SERVER 表格第 %d 行没有正确闭合\n行内容: %s", i, line)
				}
			}
		})
	}
}

// TestP2PTableAlignment 测试 P2P 模式的表格对齐
func TestP2PTableAlignment(t *testing.T) {
	devices := []string{"mlx5_0", "mlx5_bond_0", "mlx5_bond_interface_0"}

	p2pData := make(map[string]map[string]*P2PDeviceData)
	hostname := "test-host"
	p2pData[hostname] = make(map[string]*P2PDeviceData)

	for _, device := range devices {
		p2pData[hostname][device] = &P2PDeviceData{
			Hostname: hostname,
			Device:   device,
			BWSum:    350.5,
			Count:    1,
		}
	}

	// 捕获输出
	output := captureOutput(func() {
		displayP2PResults(p2pData)
	})

	t.Logf("\nP2P 表格输出:\n%s", output)

	// 验证表格对齐
	lines := strings.Split(output, "\n")
	var tableLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "│") || strings.HasPrefix(line, "┌") ||
			strings.HasPrefix(line, "├") || strings.HasPrefix(line, "└") {
			tableLines = append(tableLines, line)
		}
	}

	if len(tableLines) == 0 {
		t.Fatal("没有找到 P2P 表格输出")
	}

	// 检查所有行长度一致
	firstLineLen := len([]rune(tableLines[0]))
	for i, line := range tableLines {
		lineLen := len([]rune(line))
		if lineLen != firstLineLen {
			t.Errorf("P2P 表格第 %d 行长度不一致: 期望 %d, 实际 %d\n行内容: %s",
				i, firstLineLen, lineLen, line)
		}
	}
}

// TestCalculateMaxDeviceLength 测试计算最大设备名称长度
func TestCalculateMaxDeviceLength(t *testing.T) {
	tests := []struct {
		name        string
		devices     []string
		expectedMax int
	}{
		{
			name:        "Standard devices",
			devices:     []string{"mlx5_0", "mlx5_1"},
			expectedMax: 8, // 最小宽度为 8
		},
		{
			name:        "Bond devices",
			devices:     []string{"mlx5_bond_0", "mlx5_bond_1"},
			expectedMax: 11,
		},
		{
			name:        "Mixed devices",
			devices:     []string{"ib0", "mlx5_0", "mlx5_bond_0"},
			expectedMax: 11,
		},
		{
			name:        "Very long device",
			devices:     []string{"mlx5_bond_interface_0"},
			expectedMax: 21,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataMap := make(map[string]map[string]*DeviceData)
			dataMap["host"] = make(map[string]*DeviceData)

			for _, device := range tt.devices {
				dataMap["host"][device] = &DeviceData{
					Hostname: "host",
					Device:   device,
					BWSum:    100.0,
					Count:    1,
				}
			}

			maxLen := calculateMaxDeviceNameLength(dataMap)
			if maxLen != tt.expectedMax {
				t.Errorf("Expected max length %d, got %d", tt.expectedMax, maxLen)
			}
		})
	}
}

// TestDynamicColumnWidth 测试动态列宽格式化
func TestDynamicColumnWidth(t *testing.T) {
	tests := []struct {
		deviceName string
		width      int
		expected   string
	}{
		{"mlx5_0", 8, "mlx5_0  "},
		{"mlx5_bond_0", 12, "mlx5_bond_0 "},
		{"ib0", 8, "ib0     "},
		{"mlx5_bond_interface_0", 22, "mlx5_bond_interface_0 "},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_width_%d", tt.deviceName, tt.width), func(t *testing.T) {
			result := fmt.Sprintf("%-*s", tt.width, tt.deviceName)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
			if len(result) != tt.width {
				t.Errorf("Expected length %d, got %d", tt.width, len(result))
			}
		})
	}
}

// captureOutput 捕获函数的标准输出
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}
