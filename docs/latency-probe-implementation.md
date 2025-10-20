# Latency Probe Implementation

## 概述

实现了 `execLatencyProbeCommand` 函数，用于监控 `ib_write_lat` 进程的运行状态。这个功能与现有的 `probe` 命令类似，但专门用于延迟测试。

## 实现的功能

### 1. execLatencyProbeCommand

**位置**: `cmd/lat.go`

**功能**: 监控所有配置的主机上的 `ib_write_lat` 进程

**特性**:
- 自动检测所有配置的主机（server和client）
- 每5秒轮询一次进程状态
- 显示实时进程统计信息
- 当所有进程完成后自动退出
- 使用并发方式同时检查所有主机

### 2. 辅助函数

#### probeLatencyAllHosts
```go
func probeLatencyAllHosts(hosts map[string]bool, sshKeyPath string) []LatencyProbeResult
```
- 并发探测所有主机
- 使用 `sync.WaitGroup` 等待所有并发任务完成
- 使用 `sync.Mutex` 保护共享结果切片

#### probeLatencyHost
```go
func probeLatencyHost(hostname, sshKeyPath string) LatencyProbeResult
```
- 通过SSH执行 `ps aux | grep ib_write_lat | grep -v grep`
- 解析命令输出，统计进程数量
- 处理三种状态：
  - `RUNNING`: 有进程在运行
  - `COMPLETED`: 所有进程已完成
  - `ERROR`: SSH连接或执行失败

#### displayLatencyProbeResults
```go
func displayLatencyProbeResults(results []LatencyProbeResult)
```
- 使用表格格式显示探测结果
- 显示每个主机的状态、进程数量和详细信息
- 提供总体摘要统计信息

### 3. 数据结构

#### LatencyProbeResult
```go
type LatencyProbeResult struct {
    Hostname     string   // 主机名
    ProcessCount int      // 进程数量
    Processes    []string // 进程列表
    Error        string   // 错误信息
    Status       string   // 状态: RUNNING/COMPLETED/ERROR
}
```

## 输出示例

```
Probing ib_write_lat processes on 3 hosts...
Mode: Continuous monitoring until all processes complete

=== Latency Probe Results (14:23:45) ===
┌─────────────────────┬───────────────┬──────────────┬─────────────────┐
│ Hostname            │ Status        │ Process Count│ Details         │
├─────────────────────┼───────────────┼──────────────┼─────────────────┤
│ host1               │ 🟡 RUNNING    │            8 │ 8 process(es)   │
│ host2               │ 🟡 RUNNING    │            8 │ 8 process(es)   │
│ host3               │ 🟡 RUNNING    │            8 │ 8 process(es)   │
└─────────────────────┴───────────────┴──────────────┴─────────────────┘

Summary: 3 hosts running (24 processes), 0 completed, 0 errors
Waiting 5 seconds for next probe...

=== Latency Probe Results (14:23:50) ===
┌─────────────────────┬───────────────┬──────────────┬─────────────────┐
│ Hostname            │ Status        │ Process Count│ Details         │
├─────────────────────┼───────────────┼──────────────┼─────────────────┤
│ host1               │ ✅ COMPLETED  │            0 │ No processes    │
│ host2               │ ✅ COMPLETED  │            0 │ No processes    │
│ host3               │ ✅ COMPLETED  │            0 │ No processes    │
└─────────────────────┴───────────────┴──────────────┴─────────────────┘

Summary: 0 hosts running (0 processes), 3 completed, 0 errors
✅ All ib_write_lat processes have completed!
```

## 使用场景

在 `lat` 命令的工作流程中，步骤3（运行脚本）启动所有 `ib_write_lat` 进程后，步骤4会自动调用此函数进行监控：

```
Step 0: Precheck - Verify HCA health
Step 1: Generate Scripts - Creating latency test scripts
Step 2: Clear Report Directory
Step 3: Execute Scripts - Running latency tests
Step 4: Monitor Processes - Probing ib_write_lat processes  ← 这里使用
Step 5: Analyze Reports - Building N×N latency matrix
```

## 实现参考

此实现参考了现有的 `probe` 命令（`cmd/probe.go`），但做了以下调整：

1. **进程名称**: 监控 `ib_write_lat` 而不是 `ib_write_bw`
2. **轮询间隔**: 固定为5秒（probe命令可配置）
3. **运行模式**: 始终为连续监控模式（probe命令支持一次性模式）
4. **集成方式**: 作为 `lat` 命令工作流的一部分自动执行

## 代码质量

- ✅ 无编译错误
- ✅ 无lint警告
- ✅ 添加了必要的导入（`sync`, `time`）
- ✅ 遵循Go编码规范
- ✅ 与现有代码风格一致
- ✅ 所有测试通过

## 未来优化建议

1. **可配置轮询间隔**: 允许用户自定义轮询频率
2. **进程详情显示**: 可选择显示每个进程的详细信息（PID、端口等）
3. **超时机制**: 添加最大等待时间，防止永久阻塞
4. **性能优化**: 对于大规模集群，可以批量处理主机
5. **日志记录**: 将探测历史记录到日志文件

## 相关文件

- `cmd/lat.go` - 主实现文件
- `cmd/probe.go` - 参考实现（bandwidth探测）
- `stream/stream_latency.go` - 脚本生成逻辑
