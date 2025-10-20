# 同主机不同HCA延迟测试支持

## 问题背景

在之前的实现中，延迟测试只测试不同主机之间的HCA连接，跳过了同一主机内不同HCA之间的测试。这导致在延迟矩阵中，同主机不同HCA的单元格显示为 `-`（无数据）。

### 原有行为（错误）

```
┌────────────┬────────────┬─────────────────────────────┬─────────────────────────────┐
│            │            │ cetus-g88-061               │ cetus-g88-062               │
│            │            ├──────────────┬──────────────┼──────────────┬──────────────┤
│            │            │ mlx5_0       │ mlx5_1       │ mlx5_0       │ mlx5_1       │
├────────────┼────────────┼──────────────┼──────────────┼──────────────┼──────────────┤
│cetus-g88-061│ mlx5_0    │            - │            - │      2.00 μs │      2.92 μs │
│            ├────────────┼──────────────┼──────────────┼──────────────┼──────────────┤
│            │ mlx5_1     │            - │            - │      2.81 μs │      1.96 μs │
└────────────┴────────────┴──────────────┴──────────────┴──────────────┴──────────────┘
                             ↑ 缺少数据 ↑
```

在上面的例子中，`cetus-g88-061:mlx5_0` 到 `cetus-g88-061:mlx5_1` 之间没有延迟数据。

## 需求分析

实际需求是：
- ✅ **需要测试**：同一主机的不同HCA之间的延迟（例如 `mlx5_0` ↔ `mlx5_1`）
- ❌ **不需要测试**：同一主机的同一HCA自测（例如 `mlx5_0` → `mlx5_0`）

**原因**：
1. 同一主机的不同HCA之间的延迟反映了PCIe总线和内存访问性能
2. 对于多卡系统，了解卡间通信延迟很重要
3. 可以识别NUMA域配置问题

## 解决方案

### 代码修改

#### 1. 修改跳过逻辑 (`stream/stream_latency.go`)

**修改前**：
```go
for _, targetHost := range allHosts {
    if targetHost == currentHost {
        continue // Skip self-testing
    }
    
    for _, targetHCA := range cfg.Server.Hca {
        // 生成测试命令
    }
}
```

**修改后**：
```go
for _, targetHost := range allHosts {
    for _, targetHCA := range cfg.Server.Hca {
        // Skip only if same host AND same HCA
        if targetHost == currentHost && targetHCA == currentHCA {
            continue // Skip testing same HCA to itself
        }
        
        // 生成测试命令
    }
}
```

#### 2. 更新端口计算公式

**修改前**：
```go
// For N hosts with H HCAs each:
// Total connections = N × H × (N-1) × H
func calculateTotalLatencyPorts(hosts []string, hcas []string) int {
    numHosts := len(hosts)
    numHcas := len(hcas)
    return numHosts * numHcas * (numHosts - 1) * numHcas
}
```

**修改后**：
```go
// For N hosts with H HCAs each:
// Total HCAs = N × H
// Total connections = (N × H) × (N × H - 1)
func calculateTotalLatencyPorts(hosts []string, hcas []string) int {
    numHosts := len(hosts)
    numHcas := len(hcas)
    totalHCAs := numHosts * numHcas
    // Each HCA tests to all other HCAs (including same host, different HCA)
    // but not to itself
    return totalHCAs * (totalHCAs - 1)
}
```

### 端口数量对比

| 配置 | 修改前 | 修改后 | 增加 |
|------|--------|--------|------|
| 2 hosts, 2 HCAs | 8 | 12 | +50% |
| 3 hosts, 2 HCAs | 24 | 30 | +25% |
| 4 hosts, 3 HCAs | 108 | 132 | +22% |
| 10 hosts, 4 HCAs | 1440 | 1560 | +8.3% |

**分析**：
- 对于少量主机，端口增加比例较高
- 对于大规模集群，端口增加比例较低
- 增加的端口主要用于同主机不同HCA的测试

## 测试验证

### 单元测试更新

更新了以下测试用例以反映新的行为：

1. **`TestCalculateTotalLatencyPorts`** - 验证端口计算公式
2. **`TestCalculateTotalLatencyPortsFormula`** - 验证大规模场景
3. **`TestGenerateLatencyScriptForHCA`** - 验证脚本生成包含同主机测试
4. **`TestGenerateLatencyScriptForHCA_FilenameFormat`** - 验证文件名格式
5. **`TestGenerateLatencyScriptForHCA_PortAllocation`** - 验证端口分配

### 测试结果示例

```bash
=== RUN   TestGenerateLatencyScriptForHCA
✅ Generated latency scripts for host1:mlx5_0 (ports 20000-20004)
   Server script preview (first command): 
   ssh -i /home/user/.ssh/id_rsa host1 'ib_write_lat -d mlx5_0 -D 5 -p 20000 -R -x 3 \
   --out_json --out_json_file /tmp/latency_s_host1_mlx5_0_from_host1_mlx5_1_p20000.json \
   >/dev/null 2>&1 &'; sleep 0.02
                                          ↑ 同主机不同HCA测试 ↑
   Client script preview (first command): 
   ssh -i /home/user/.ssh/id_rsa host1 'ib_write_lat -d mlx5_1 -D 5 -p 20000 -R -x 3 \
   192.168.1.1 --out_json --out_json_file /tmp/latency_c_host1_mlx5_1_to_host1_mlx5_0_p20000.json \
   >/dev/null 2>&1 &'; sleep 0.06
                      ↑ 同主机，不同设备 ↑
--- PASS: TestGenerateLatencyScriptForHCA (0.00s)
```

## 预期效果

### 修复后的延迟矩阵

```
┌────────────┬────────────┬─────────────────────────────┬─────────────────────────────┐
│            │            │ cetus-g88-061               │ cetus-g88-062               │
│            │            ├──────────────┬──────────────┼──────────────┬──────────────┤
│            │            │ mlx5_0       │ mlx5_1       │ mlx5_0       │ mlx5_1       │
├────────────┼────────────┼──────────────┼──────────────┼──────────────┼──────────────┤
│cetus-g88-061│ mlx5_0    │            - │      1.23 μs │      2.00 μs │      2.92 μs │
│            ├────────────┼──────────────┼──────────────┼──────────────┼──────────────┤
│            │ mlx5_1     │      1.24 μs │            - │      2.81 μs │      1.96 μs │
├────────────┼────────────┼──────────────┼──────────────┼──────────────┼──────────────┤
│cetus-g88-062│ mlx5_0    │      2.03 μs │      2.79 μs │            - │      1.25 μs │
│            ├────────────┼──────────────┼──────────────┼──────────────┼──────────────┤
│            │ mlx5_1     │      2.75 μs │      2.00 μs │      1.26 μs │            - │
└────────────┴────────────┴──────────────┴──────────────┴──────────────┴──────────────┘
                             ↑ 现在有数据了 ↑
```

### 性能洞察

通过新增的同主机HCA间延迟数据，可以分析：

1. **PCIe性能**：同主机HCA间延迟反映PCIe总线性能
2. **NUMA影响**：跨NUMA节点的HCA延迟通常更高
3. **卡间通信**：多卡训练/推理场景的通信瓶颈
4. **系统配置**：识别BIOS设置或硬件配置问题

### 示例分析

假设测试结果显示：
```
host1:mlx5_0 → host1:mlx5_1: 1.2 μs   (同主机，可能同NUMA)
host1:mlx5_0 → host1:mlx5_2: 2.5 μs   (同主机，可能跨NUMA)
host1:mlx5_0 → host2:mlx5_0: 2.0 μs   (跨主机)
```

**结论**：
- 同NUMA节点的HCA延迟最低 (1.2 μs)
- 跨NUMA节点的HCA延迟反而高于某些跨主机连接 (2.5 μs vs 2.0 μs)
- **建议**：调整应用以优先使用同NUMA节点的HCA

## 向后兼容性

- ✅ 配置文件格式不变
- ✅ 命令行接口不变
- ✅ 输出格式不变（仅增加数据）
- ✅ 现有脚本继续工作
- ⚠️ 端口需求增加（见上表）

## 使用建议

1. **检查端口范围**：确保 `start_port` 到 `start_port + total_ports` 的范围内端口可用
2. **NUMA拓扑**：关注同主机HCA间的延迟差异，可能反映NUMA配置
3. **网络规划**：对于多卡系统，考虑HCA与PCIe插槽的物理位置关系

## 相关文件

- `stream/stream_latency.go` - 脚本生成逻辑修改
- `stream/stream_latency_test.go` - 测试用例更新
- `cmd/lat.go` - 延迟矩阵显示（支持合并单元格）
- `docs/latency-matrix-merged-cells.md` - 延迟矩阵显示格式

## 版本历史

- **v0.2.1** (2025-01-20): 添加同主机不同HCA延迟测试支持
- **v0.2.0**: 初始延迟测试实现（仅跨主机）
