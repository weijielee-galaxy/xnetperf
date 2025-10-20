# Serial Number 列功能增强总结

## 概述

本次更新在 `precheck` 和 `analyze` 命令中添加了 Serial Number 列，用于显示系统的序列号信息，并进行了以下重要优化：

1. **Serial Number 处理逻辑**：如果序列号包含 `-` 字符，则按 `-` 分割并获取最后一个值
2. **列位置调整**：Serial Number 列调整到第一列（最左侧）
3. **analyze 结果集成**：在 analyze 命令的所有模式（FullMesh、InCast、P2P）中都添加了 Serial Number 显示

## 功能实现

### 1. Serial Number 处理逻辑

**原始需求**：如果获取到的 Serial Number 包含 `-`，则按 `-` 分割获取最后一个值。

**实现代码**（`cmd/precheck.go`）：
```go
// 处理Serial Number：如果包含-则按-分割获取最后一个值
if strings.Contains(serialNumberStr, "-") {
    parts := strings.Split(serialNumberStr, "-")
    if len(parts) > 0 {
        serialNumberStr = parts[len(parts)-1]
    }
}
```

**示例**：
- 输入：`DELL-ABC-12345` → 输出：`12345`
- 输入：`HP-SERVER-XYZ-67890` → 输出：`67890`
- 输入：`SN123456`（无横杠）→ 输出：`SN123456`

### 2. Precheck 命令列位置调整

**修改前**：
```
┌───────────────┬──────────┬────────────────┬───────────────┬─────────────┬──────────────┬─────────────────┬─────────────────┬───────────────┐
│ Hostname      │ HCA      │ Physical State │ Logical State │ Speed       │ FW Version   │ Board ID        │ Serial Number   │ Status        │
```

**修改后**：
```
┌─────────────────┬──────────────┬──────────┬────────────────┬───────────────┬─────────────────────┬──────────────┬─────────────────┬───────────────┐
│ Serial Number   │ Hostname     │ HCA      │ Physical State │ Logical State │ Speed               │ FW Version   │ Board ID        │ Status        │
```

**列顺序**：Serial Number | Hostname | HCA | Physical State | Logical State | Speed | FW Version | Board ID | Status

### 3. Analyze 命令集成

#### 3.1 数据结构修改

**DeviceData 结构体**（用于 FullMesh/InCast 模式）：
```go
type DeviceData struct {
    Hostname     string
    Device       string
    SerialNumber string  // 新增字段
    BWSum        float64
    Count        int
    IsClient     bool
}
```

**P2PDeviceData 结构体**（用于 P2P 模式）：
```go
type P2PDeviceData struct {
    Hostname     string
    Device       string
    SerialNumber string  // 新增字段
    BWSum        float64
    Count        int
}
```

#### 3.2 Serial Number 获取逻辑

添加了 `getSerialNumberForHost` 函数，通过 SSH 获取主机的系统序列号：

```go
func getSerialNumberForHost(hostname, sshKeyPath string) string {
    cmd := buildSSHCommand(hostname, "cat /sys/class/dmi/id/product_serial", sshKeyPath)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return "N/A"
    }
    
    serialNumber := strings.TrimSpace(string(output))
    
    // 处理带横杠的序列号
    if strings.Contains(serialNumber, "-") {
        parts := strings.Split(serialNumber, "-")
        if len(parts) > 0 {
            serialNumber = parts[len(parts)-1]
        }
    }
    
    if serialNumber == "" {
        return "N/A"
    }
    
    return serialNumber
}
```

#### 3.3 FullMesh/InCast 模式显示

**客户端表格**：
```
CLIENT DATA (TX)
┌───────────────┬─────────────────────┬─────────────┬─────────────┬──────────────┬─────────────────┬──────────┐
│ Serial Number │ Hostname            │ Device      │ TX (Gbps)   │ SPEC (Gbps)  │ DELTA           │ Status   │
├───────────────┼─────────────────────┼─────────────┼─────────────┼──────────────┼─────────────────┼──────────┤
│ SN111111      │ client-001          │ mlx5_0      │     3505.00 │       266.67 │   3238.3(1214%) │ NOT OK   │
│               │                     │ mlx5_1      │     3505.00 │       266.67 │   3238.3(1214%) │ NOT OK   │
```

**服务端表格**：
```
SERVER DATA (RX)
┌───────────────┬─────────────────────┬─────────────┬─────────────┐
│ Serial Number │ Hostname            │ Device      │ RX (Gbps)   │
├───────────────┼─────────────────────┼─────────────┼─────────────┤
│ SN333333      │ server-001          │ mlx5_0      │     3505.00 │
│               │                     │ mlx5_1      │     3505.00 │
```

#### 3.4 P2P 模式显示

```
P2P Performance Analysis
┌───────────────┬─────────────────────┬─────────────┬─────────────┐
│ Serial Number │ Hostname            │ Device      │ Speed (Gbps)│
├───────────────┼─────────────────────┼─────────────┼─────────────┤
│ SN111111      │ node-001            │ mlx5_0      │      350.50 │
│               │                     │ mlx5_1      │      350.50 │
├───────────────┼─────────────────────┼─────────────┼─────────────┤
│ SN222222      │ node-002            │ mlx5_bond_0 │      350.50 │
```

## 动态列宽计算

### Precheck 命令

```go
// 计算 Serial Number 列的最大宽度
maxSerialNumberWidth := len("Serial Number")
for _, result := range results {
    serialNumber := result.SerialNumber
    if serialNumber == "" {
        serialNumber = "N/A"
    }
    if len(serialNumber) > maxSerialNumberWidth {
        maxSerialNumberWidth = len(serialNumber)
    }
}
if maxSerialNumberWidth < 15 {
    maxSerialNumberWidth = 15
}
```

### Analyze 命令

添加了 `calculateMaxSerialNumberLength` 函数：

```go
func calculateMaxSerialNumberLength(dataMap map[string]map[string]*DeviceData) int {
    maxLen := len("Serial Number")
    for _, devices := range dataMap {
        for _, data := range devices {
            if len(data.SerialNumber) > maxLen {
                maxLen = len(data.SerialNumber)
            }
        }
    }
    if maxLen < 15 {
        maxLen = 15
    }
    return maxLen
}
```

## 单元测试

创建了全面的单元测试 `cmd/serial_number_test.go`，包括：

### 1. TestPrecheckSerialNumberDisplay
测试 precheck 命令的 Serial Number 显示效果：
- 标准格式序列号
- 带横杠的序列号（提取最后部分）
- 不同长度的序列号混合
- 错误情况下的 N/A 显示

### 2. TestAnalyzeSerialNumberDisplay
测试 analyze 命令（FullMesh/InCast 模式）的 Serial Number 显示效果：
- 带序列号的 FullMesh 结果展示
- 不同长度序列号的对齐效果

### 3. TestP2PSerialNumberDisplay
测试 P2P 模式的 Serial Number 显示效果：
- P2P 模式下的序列号显示
- 不同长度序列号在 P2P 模式下的对齐

### 4. TestSerialNumberParsing
测试序列号的解析逻辑：
- 简单序列号（无横杠）
- 单个横杠的序列号
- 多个横杠的序列号
- 空序列号
- 以横杠结尾的序列号

### 5. TestSerialNumberColumnWidth
测试序列号列宽度计算：
- 短序列号（使用最小宽度）
- 中等长度序列号
- 超长序列号（扩展列宽）
- 混合长度

### 6. TestSerialNumberColumnPosition
验证 Serial Number 列在第一列的位置

### 测试结果
```bash
$ go test ./cmd/ -v
...
PASS
ok      xnetperf/cmd    0.092s
```

所有测试通过 ✅

## 代码修改文件列表

### 核心功能文件
1. **cmd/precheck.go**
   - 添加 Serial Number 处理逻辑（按 `-` 分割）
   - 调整表格列顺序（Serial Number 放第一列）
   - 更新所有表格格式化代码

2. **cmd/analyze.go**
   - 修改 `DeviceData` 和 `P2PDeviceData` 结构体
   - 添加 `getSerialNumberForHost` 函数
   - 添加 `calculateMaxSerialNumberLength` 函数
   - 更新 `collectReportData` 和 `collectP2PReportData` 函数
   - 修改所有表格显示函数
   - 添加 `exec` 包导入

### 测试文件
3. **cmd/serial_number_test.go**（新增）
   - 6 个测试套件，覆盖所有场景
   - 包含性能基准测试

### 文档文件
4. **docs/serial-number-column-enhancement.md**（本文档）

## 使用示例

### Precheck 命令
```bash
$ xnetperf precheck -c config.yaml

=== Precheck Results ===
┌─────────────────┬──────────────┬──────────┬────────────────┬───────────────┬─────────────────────┬──────────────┬─────────────────┬───────────────┐
│ Serial Number   │ Hostname     │ HCA      │ Physical State │ Logical State │ Speed               │ FW Version   │ Board ID        │ Status        │
├─────────────────┼──────────────┼──────────┼────────────────┼───────────────┼─────────────────────┼──────────────┼─────────────────┼───────────────┤
│ 12345           │ server-001   │ mlx5_0   │ LinkUp         │ ACTIVE        │ 200 Gb/sec (2X NDR) │ 28.43.2026   │ MT_0000000844   │ [+] HEALTHY   │
│ 12345           │              │ mlx5_1   │ LinkUp         │ ACTIVE        │ 200 Gb/sec (2X NDR) │ 28.43.2026   │ MT_0000000844   │ [+] HEALTHY   │
├─────────────────┼──────────────┼──────────┼────────────────┼───────────────┼─────────────────────┼──────────────┼─────────────────┼───────────────┤
│ 67890           │ server-002   │ mlx5_0   │ LinkUp         │ ACTIVE        │ 200 Gb/sec (2X NDR) │ 28.43.2025   │ MT_0000000845   │ [+] HEALTHY   │
└─────────────────┴──────────────┴──────────┴────────────────┴───────────────┴─────────────────────┴──────────────┴─────────────────┴───────────────┘
```

### Analyze 命令
```bash
$ xnetperf analyze --reports-dir reports

=== Network Performance Analysis ===
CLIENT DATA (TX)
┌───────────────┬─────────────────────┬─────────────┬─────────────┬──────────────┬─────────────────┬──────────┐
│ Serial Number │ Hostname            │ Device      │ TX (Gbps)   │ SPEC (Gbps)  │ DELTA           │ Status   │
├───────────────┼─────────────────────┼─────────────┼─────────────┼──────────────┼─────────────────┼──────────┤
│ 12345         │ client-001          │ mlx5_0      │      350.50 │       400.00 │     -49.5(-12%) │ OK       │
│               │                     │ mlx5_1      │      350.50 │       400.00 │     -49.5(-12%) │ OK       │
```

## 技术要点

### 1. 数据获取时机
- **Precheck**: 在执行 precheck 时同步获取 Serial Number
- **Analyze**: 在收集报告数据时通过 SSH 获取 Serial Number

### 2. 错误处理
- SSH 连接失败 → 显示 "N/A"
- 文件不存在或无权限 → 显示 "N/A"
- 空序列号 → 显示 "N/A"

### 3. 性能优化
- 使用 SSH 命令批量获取，减少连接次数
- 缓存序列号避免重复查询（同一主机的多个设备共享序列号）

### 4. 兼容性
- 支持所有 HCA 命名格式（mlx5_0、mlx5_bond_0、自定义名称等）
- 支持所有 stream 类型（FullMesh、InCast、P2P）
- 向后兼容旧配置和旧报告文件

## 应用场景

### 1. 资产管理
通过序列号快速识别硬件资产：
```
Serial Number: 12345 → Dell Server A
Serial Number: 67890 → HP Server B
```

### 2. 故障追踪
通过序列号关联问题机器：
```
序列号 12345 的服务器 HCA 状态异常 → 定位到特定硬件
```

### 3. 批量管理
相同序列号表示同一主机的不同 HCA：
```
Serial Number: 12345
  ├─ mlx5_0 (HEALTHY)
  └─ mlx5_1 (HEALTHY)
```

## 版本历史

- **v0.1.3**: 添加 Serial Number 列功能
  - 实现序列号解析逻辑（按 `-` 分割）
  - 调整列位置到第一列
  - 集成到 analyze 命令所有模式
  - 完整的单元测试覆盖

## 后续计划

1. 支持自定义序列号来源（配置文件、数据库等）
2. 添加序列号过滤功能
3. 支持序列号导出和报表生成
4. Web UI 集成序列号显示

## 注意事项

1. **权限要求**: 读取 `/sys/class/dmi/id/product_serial` 可能需要管理员权限
2. **网络依赖**: analyze 时需要 SSH 连接到远程主机
3. **数据隐私**: 序列号可能包含敏感信息，注意日志安全
4. **环境支持**: 某些虚拟化环境可能不提供序列号信息
