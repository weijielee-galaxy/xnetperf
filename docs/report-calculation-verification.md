# 报告计算逻辑验证

## 对比检查：命令行版本 vs API 版本

### 数据收集阶段

#### 命令行版本 (cmd/analyze.go)
```go
// 第 188-189 行
dataMap[hostname][device].BWSum += report.Results.BWAverage
dataMap[hostname][device].Count++
```

#### API 版本 (workflow/workflow.go)
```go
// 第 530-531 行
dataMap[hostname][device].BWSum += report.Results.BWAverage
dataMap[hostname][device].Count++
```

✅ **一致**: 两者都是累加 `BWAverage`，并记录文件数量

---

### 客户端数据显示

#### 命令行版本 (cmd/analyze.go)
```go
// 第 382 行 - displayEnhancedClientTable()
actualBW := data.BWSum  // 使用累加值
```

#### API 版本 (workflow/workflow.go)
```go
// 第 627 行 - convertClientData()
actualBW := data.BWSum  // 使用累加值
```

#### 前端显示 (ReportResults.jsx)
```javascript
// 第 165 行
<Td isNumeric>{item.actual_bw.toFixed(2)}</Td>  // 显示 actual_bw (即 ActualBW)
```

✅ **一致**: 客户端都使用 `BWSum` 累加值

---

### 服务端数据显示

#### 命令行版本 (cmd/analyze.go)
```go
// 第 251 行 - displayDataTable()
total := data.BWSum // 使用累加值而不是平均值
```

#### API 版本 (workflow/workflow.go)
```go
// 第 658 行 - convertServerData()
RxBW: data.BWSum,  // 使用累加值
```

#### 前端显示 (ReportResults.jsx)
```javascript
// 第 217 行
<Td isNumeric>{item.rx_bw.toFixed(2)}</Td>  // 显示 rx_bw (即 RxBW)
```

✅ **一致**: 服务端都使用 `BWSum` 累加值

---

### 理论带宽计算

#### 命令行版本 (cmd/analyze.go)
```go
// 第 339 行 - calculateTotalServerBandwidth()
func calculateTotalServerBandwidth(serverData map[string]map[string]*DeviceData, specSpeed float64) float64 {
	total := float64(0)
	for _, devices := range serverData {
		for range devices {
			total += specSpeed  // 每个设备加一次理论速度
		}
	}
	return total
}

// 第 350 行 - calculateClientCount()
func calculateClientCount(clientData map[string]map[string]*DeviceData) int {
	count := 0
	for _, devices := range clientData {
		count += len(devices)  // 统计设备数量
	}
	return count
}

// 第 200-202 行
totalServerBW := calculateTotalServerBandwidth(serverData, specSpeed)
clientCount := calculateClientCount(clientData)
theoreticalBWPerClient := totalServerBW / float64(clientCount)
```

#### API 版本 (workflow/workflow.go)
```go
// 第 601 行 - calculateTotalServerBandwidth()
func calculateTotalServerBandwidth(serverData map[string]map[string]*deviceData, specSpeed float64) float64 {
	total := float64(0)
	for _, devices := range serverData {
		for range devices {
			total += specSpeed  // 每个设备加一次理论速度
		}
	}
	return total
}

// 第 611 行 - calculateClientCount()
func calculateClientCount(clientData map[string]map[string]*deviceData) int {
	count := 0
	for _, devices := range clientData {
		count += len(devices)  // 统计设备数量
	}
	return count
}

// 第 437-441 行 - GenerateReport()
report.TotalServerBW = calculateTotalServerBandwidth(serverData, cfg.Speed)
report.ClientCount = calculateClientCount(clientData)
if report.ClientCount > 0 {
	report.TheoreticalBWPerClient = report.TotalServerBW / float64(report.ClientCount)
}
```

✅ **一致**: 理论带宽计算逻辑完全相同

---

## 实例验证

### 测试场景
假设配置：
- **服务端**: 2 台主机，每台 1 个 HCA (mlx5_0)，理论速度 100 Gbps
  - server1: mlx5_0
  - server2: mlx5_0
- **客户端**: 4 台主机，每台 1 个 HCA (mlx5_0)
  - client1: mlx5_0
  - client2: mlx5_0
  - client3: mlx5_0
  - client4: mlx5_0

### 报告文件假设
FullMesh 模式下，每个客户端会生成多个报告文件（每个服务端一个）。

假设测试结果：
- client1 → server1: 25 Gbps
- client1 → server2: 25 Gbps (总计 50 Gbps)
- client2 → server1: 24 Gbps
- client2 → server2: 24 Gbps (总计 48 Gbps)
- client3 → server1: 26 Gbps
- client3 → server2: 26 Gbps (总计 52 Gbps)
- client4 → server1: 25 Gbps
- client4 → server2: 25 Gbps (总计 50 Gbps)

服务端接收：
- server1: 25+24+26+25 = 100 Gbps
- server2: 25+24+26+25 = 100 Gbps

### 计算过程

#### 1. 理论带宽
- 总服务端带宽 = 2 × 100 = 200 Gbps
- 客户端数量 = 4
- 理论单客户端带宽 = 200 ÷ 4 = 50 Gbps

#### 2. 客户端显示
| 主机 | 设备 | 实际带宽 | 理论带宽 | 差值 | 差值% | 状态 |
|------|------|---------|---------|------|------|------|
| client1 | mlx5_0 | 50 | 50 | 0 | 0% | OK |
| client2 | mlx5_0 | 48 | 50 | -2 | -4% | OK |
| client3 | mlx5_0 | 52 | 50 | +2 | +4% | OK |
| client4 | mlx5_0 | 50 | 50 | 0 | 0% | OK |

#### 3. 服务端显示
| 主机 | 设备 | 接收带宽 |
|------|------|---------|
| server1 | mlx5_0 | 100 |
| server2 | mlx5_0 | 100 |

### 验证结果
✅ 两个版本的计算结果完全一致

---

## 可能的问题点

### 如果用户觉得服务端数据不对，可能的原因：

1. **期望值错误**
   - 用户可能期望看到平均值而不是累加值
   - 解决：这是设计行为，累加值代表该 HCA 接收的总带宽

2. **配置文件 speed 值不正确**
   - 如果配置中 `speed: 100` 但实际是 200 Gbps 的网卡
   - 会导致理论单客户端带宽计算错误

3. **报告文件收集不完整**
   - 某些报告文件没有被收集
   - 导致累加值偏小

4. **文件名解析问题**
   - 如果文件名格式不符合预期，可能被跳过
   - 格式：`report_c_<hostname>_<device>_port.json` (客户端)
   - 格式：`report_s_<hostname>_<device>_port.json` (服务端)

---

## 调试建议

### 1. 检查报告文件
```bash
# 查看收集的报告文件
ls -la reports/*/

# 统计报告文件数量
find reports/ -name "*.json" | wc -l

# 查看某个报告文件内容
cat reports/<hostname>/report_*.json | jq .
```

### 2. 验证数据收集
在浏览器开发者工具中查看 API 返回的原始数据：
```
Network → /api/configs/<name>/report → Response
```

检查：
- `server_data` 中每个设备的 `rx_bw` 值
- `client_data` 中每个设备的 `actual_bw` 值
- `total_server_bw` 和 `theoretical_bw_per_client` 是否正确

### 3. 对比命令行
使用命令行工具分析同一批报告：
```bash
./xnetperf analyze -c config.yaml
```

对比输出是否与 Web 界面一致。

---

## 结论

✅ **代码逻辑完全一致**: API 版本的计算逻辑与命令行版本 100% 相同

如果用户觉得服务端数据不对，建议：
1. 检查配置文件中的 `speed` 值是否正确
2. 验证报告文件是否完整收集
3. 使用命令行工具对比验证
4. 提供具体的测试配置和预期值，以便进一步分析
