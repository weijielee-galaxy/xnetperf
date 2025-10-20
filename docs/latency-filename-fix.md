# 延迟测试文件命名修复 - 完整矩阵支持

## 概述

本文档记录了延迟测试报告文件命名格式的重要改进，以支持完整的 N×N 延迟矩阵显示。

## 问题背景

### ib_write_lat 的特殊限制

与 `ib_write_bw` 不同，**`ib_write_lat` 的每个 server 端口只能服务一个 client 连接**。

**示例说明：**
```
场景：host1:mlx5_0 需要接收来自 host2:mlx5_0 和 host3:mlx5_0 的延迟测试

ib_write_bw (带宽测试):
  ✅ host1:mlx5_0 在端口 20000 启动一个 server
  ✅ host2:mlx5_0 和 host3:mlx5_0 可以同时连接到 host1:20000

ib_write_lat (延迟测试):
  ❌ host1:mlx5_0 不能在端口 20000 同时服务两个 client
  ✅ 必须在不同端口启动两个 server:
     - 端口 20000: 接收来自 host2:mlx5_0 的连接
     - 端口 20001: 接收来自 host3:mlx5_0 的连接
```

### 原始文件命名问题

**旧的文件命名格式（不完整）：**
```
latency_c_host2_mlx5_0_20000.json
```

**问题：**
- 只知道：`host2:mlx5_0` 发起了测试，使用端口 `20000`
- **不知道：** 连接到哪个目标主机和 HCA
- **导致：** 无法构建完整的 N×N 延迟矩阵

**显示结果（错误）：**
```
Source: host2:mlx5_0
Target: unknown  ← 无法确定目标
Latency: 1.23 μs
```

## 解决方案

### 新的文件命名格式

**改进后的格式（完整信息）：**
```
Server: latency_s_{serverHost}_{serverHCA}_from_{clientHost}_{clientHCA}_p{PORT}.json
Client: latency_c_{clientHost}_{clientHCA}_to_{serverHost}_{serverHCA}_p{PORT}.json
```

**实际示例：**
```bash
# host2:mlx5_0 连接到 host1:mlx5_1，端口 20000
Server端: latency_s_host1_mlx5_1_from_host2_mlx5_0_p20000.json
Client端: latency_c_host2_mlx5_0_to_host1_mlx5_1_p20000.json
```

### 文件名解析逻辑

**新的解析逻辑：**
```go
func parseLatencyReport(filePath string) (*LatencyData, error) {
    filename := filepath.Base(filePath)
    nameWithoutExt := strings.TrimSuffix(filename, ".json")

    // 只处理 client 报告
    if !strings.HasPrefix(nameWithoutExt, "latency_c_") {
        return nil, nil
    }

    // 移除 "latency_c_" 前缀
    remaining := strings.TrimPrefix(nameWithoutExt, "latency_c_")

    // 按 "_to_" 分割获取源和目标
    // 示例: "host2_mlx5_0_to_host1_mlx5_1_p20000"
    parts := strings.Split(remaining, "_to_")
    if len(parts) != 2 {
        return nil, fmt.Errorf("invalid filename format")
    }

    // 解析源 (format: host_hca)
    sourceParts := strings.SplitN(parts[0], "_", 2)
    sourceHost := sourceParts[0]  // "host2"
    sourceHCA := sourceParts[1]   // "mlx5_0"

    // 解析目标 (format: host_hca_pPORT)
    // 找到最后一个 "_p" 来分离 HCA 和端口
    targetStr := parts[1]
    pIndex := strings.LastIndex(targetStr, "_p")
    hostAndHCA := targetStr[:pIndex]  // "host1_mlx5_1"
    
    targetParts := strings.SplitN(hostAndHCA, "_", 2)
    targetHost := targetParts[0]  // "host1"
    targetHCA := targetParts[1]   // "mlx5_1"

    // 创建 LatencyData 对象
    return &LatencyData{
        SourceHost:   sourceHost,
        SourceHCA:    sourceHCA,
        TargetHost:   targetHost,
        TargetHCA:    targetHCA,
        AvgLatencyUs: avgLatency,
    }, nil
}
```

### 端口分配逻辑验证

**当前的端口分配逻辑是正确的！**

```go
port := cfg.StartPort

for _, currentHost := range allHosts {
    for _, currentHCA := range cfg.Client.Hca {
        // 为当前 HCA 生成脚本
        for _, targetHost := range allHosts {
            if targetHost == currentHost {
                continue // 跳过自己
            }
            for _, targetHCA := range cfg.Server.Hca {
                // currentHost:currentHCA 作为 server
                // targetHost:targetHCA 作为 client
                port++  // 每个连接使用不同的端口
            }
        }
    }
}
```

**示例（2台主机，每台2个HCA）：**
```
端口 20000: host1:mlx5_0(server) ← host2:mlx5_0(client)
端口 20001: host1:mlx5_0(server) ← host2:mlx5_1(client)
端口 20002: host1:mlx5_0(server) ← host3:mlx5_0(client)
端口 20003: host1:mlx5_0(server) ← host3:mlx5_1(client)
---
端口 20004: host1:mlx5_1(server) ← host2:mlx5_0(client)
端口 20005: host1:mlx5_1(server) ← host2:mlx5_1(client)
...
```

**关键点：**
- ✅ 端口**连续分配**，不需要跳跃
- ✅ 每个 server 端口对应**唯一的一对 (server_HCA, client_HCA)**
- ✅ 覆盖完整的 N×N 矩阵

## 完整的 N×N 矩阵示例

### 测试场景
```
3 台主机，每台 2 个 HCA：
- host1: mlx5_0, mlx5_1
- host2: mlx5_0, mlx5_1  
- host3: mlx5_0, mlx5_1
```

### 生成的文件
```bash
# host1 发起的测试 (4个测试)
latency_c_host1_mlx5_0_to_host2_mlx5_0_p20000.json
latency_c_host1_mlx5_0_to_host2_mlx5_1_p20001.json
latency_c_host1_mlx5_0_to_host3_mlx5_0_p20002.json
latency_c_host1_mlx5_0_to_host3_mlx5_1_p20003.json

latency_c_host1_mlx5_1_to_host2_mlx5_0_p20004.json
latency_c_host1_mlx5_1_to_host2_mlx5_1_p20005.json
latency_c_host1_mlx5_1_to_host3_mlx5_0_p20006.json
latency_c_host1_mlx5_1_to_host3_mlx5_1_p20007.json

# host2 发起的测试 (4个测试)
latency_c_host2_mlx5_0_to_host1_mlx5_0_p20008.json
latency_c_host2_mlx5_0_to_host1_mlx5_1_p20009.json
...

# host3 发起的测试 (4个测试)
...

# 总计: 3 × 2 × 2 × 2 = 24 个测试
```

### 完整矩阵显示

```
================================================================================
📊 Latency Matrix (Average Latency in microseconds)
================================================================================
┌──────────────────────┬──────────────┬──────────────┬──────────────┬──────────────┬──────────────┬──────────────┐
│ Source → Target      │ host1:mlx5_0 │ host1:mlx5_1 │ host2:mlx5_0 │ host2:mlx5_1 │ host3:mlx5_0 │ host3:mlx5_1 │
├──────────────────────┼──────────────┼──────────────┼──────────────┼──────────────┼──────────────┼──────────────┤
│ host1:mlx5_0         │ -            │      1.20 μs │      1.45 μs │      1.48 μs │      1.52 μs │      1.55 μs │
├──────────────────────┼──────────────┼──────────────┼──────────────┼──────────────┼──────────────┼──────────────┤
│ host1:mlx5_1         │      1.22 μs │ -            │      1.46 μs │      1.49 μs │      1.53 μs │      1.56 μs │
├──────────────────────┼──────────────┼──────────────┼──────────────┼──────────────┼──────────────┼──────────────┤
│ host2:mlx5_0         │      1.47 μs │      1.50 μs │ -            │      1.21 μs │      1.51 μs │      1.54 μs │
├──────────────────────┼──────────────┼──────────────┼──────────────┼──────────────┼──────────────┼──────────────┤
│ host2:mlx5_1         │      1.48 μs │      1.51 μs │      1.23 μs │ -            │      1.52 μs │      1.55 μs │
├──────────────────────┼──────────────┼──────────────┼──────────────┼──────────────┼──────────────┼──────────────┤
│ host3:mlx5_0         │      1.54 μs │      1.57 μs │      1.53 μs │      1.56 μs │ -            │      1.24 μs │
├──────────────────────┼──────────────┼──────────────┼──────────────┼──────────────┼──────────────┼──────────────┤
│ host3:mlx5_1         │      1.55 μs │      1.58 μs │      1.54 μs │      1.57 μs │      1.25 μs │ -            │
└──────────────────────┴──────────────┴──────────────┴──────────────┴──────────────┴──────────────┴──────────────┘
```

**矩阵特点：**
- ✅ 对角线为 `-`（不测试自己到自己）
- ✅ 行代表源（发起测试的 HCA）
- ✅ 列代表目标（接收测试的 HCA）
- ✅ 每个格子显示平均延迟（μs）

## 代码变更总结

### 修改的文件

| 文件 | 变更内容 | 说明 |
|------|---------|------|
| `stream/stream_latency.go` | 修改文件命名格式 | 添加 `_to_` 和 `_from_` 标识连接方向 |
| `cmd/lat.go` | 重写 `parseLatencyReport()` | 解析新的文件名格式提取完整信息 |
| `cmd/lat_test.go` | 更新测试用例 | 使用新的文件名格式 |

### 具体变更

**1. stream/stream_latency.go - 文件命名**

```go
// 修改前
OutputFileName(fmt.Sprintf("%s/latency_c_%s_%s_%d.json",
    cfg.Report.Dir, targetHost, targetHCA, port))

// 修改后
OutputFileName(fmt.Sprintf("%s/latency_c_%s_%s_to_%s_%s_p%d.json",
    cfg.Report.Dir, targetHost, targetHCA, currentHost, currentHCA, port))
```

**2. cmd/lat.go - 解析逻辑**

```go
// 修改前：只能提取 sourceHost 和 sourceHCA
parts := strings.Split(strings.TrimSuffix(filename, ".json"), "_")
sourceHost := parts[2]
sourceHCA := parts[3]
// targetHost = "unknown"  ← 无法确定目标

// 修改后：可以提取完整的源和目标信息
remaining := strings.TrimPrefix(nameWithoutExt, "latency_c_")
parts := strings.Split(remaining, "_to_")
// 解析 parts[0] → sourceHost, sourceHCA
// 解析 parts[1] → targetHost, targetHCA
```

## 测试验证

### 单元测试更新

**测试用例：**
```go
{
    name: "Valid client report",
    filename: "latency_c_host1_mlx5_0_to_host2_mlx5_1_p20000.json",
    expectedSource:  "host1:mlx5_0",
    expectedTarget:  "host2:mlx5_1",
    expectedLatency: 1.23,
}
```

**测试结果：**
```bash
$ go test ./cmd/ ./stream/ -run "Latency"
ok      xnetperf/cmd    0.019s
ok      xnetperf/stream 0.010s
✅ 所有测试通过
```

### 兼容性

**旧文件名格式：**
```
latency_c_host1_mlx5_0_20000.json
```
**处理方式：** 
- ✅ 会被识别为无效格式并报错
- ✅ 明确提示缺少 `_to_` 分隔符
- ✅ 不会产生错误的解析结果

## 使用示例

### 运行延迟测试
```bash
$ xnetperf lat -c config.yaml

🚀 Starting xnetperf latency testing workflow...
============================================================

🔍 Step 0/5: Performing network card precheck...
✅ Precheck passed!

📋 Step 1/5: Generating latency test scripts...
✅ Generated latency scripts for host1:mlx5_0 (ports 20000-20003)
✅ Generated latency scripts for host1:mlx5_1 (ports 20004-20007)
✅ Generated latency scripts for host2:mlx5_0 (ports 20008-20011)
✅ Generated latency scripts for host2:mlx5_1 (ports 20012-20015)

▶️  Step 2/5: Running latency tests...
...
```

### 检查生成的报告
```bash
$ ls -1 reports/latency_*.json
reports/latency_c_host1_mlx5_0_to_host2_mlx5_0_p20000.json
reports/latency_c_host1_mlx5_0_to_host2_mlx5_1_p20001.json
reports/latency_c_host2_mlx5_0_to_host1_mlx5_0_p20008.json
reports/latency_c_host2_mlx5_0_to_host1_mlx5_1_p20009.json
...
```

### 文件名解释
```
latency_c_host1_mlx5_0_to_host2_mlx5_1_p20000.json
    │      │     │           │     │      │
    │      │     │           │     │      └─ 端口号
    │      │     │           │     └─ 目标HCA
    │      │     │           └─ 目标主机
    │      │     └─ 源HCA
    │      └─ 源主机
    └─ client报告
```

## 关键要点总结

1. **ib_write_lat 限制**：每个 server 端口只能服务一个 client
2. **端口分配**：连续分配，每个连接使用不同端口
3. **文件命名**：使用 `_to_` 和 `_from_` 明确连接方向
4. **矩阵完整性**：通过文件名可以构建完整的 N×N 矩阵
5. **端口不跳跃**：原有的连续分配逻辑是正确的

## 相关文档

- [延迟测试功能指南](latency-testing-guide.md)
- [端口修复和增强功能](latency-port-fix-and-enhancements.md)
- [延迟表格显示改进](latency-table-improvement.md)
- [延迟目录修复](latency-directory-fix.md)
