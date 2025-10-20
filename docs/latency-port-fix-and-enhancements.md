# Latency Testing Port Fix and Enhancements

## 概述

本文档记录了延迟测试（latency testing）的三个重要改进：
1. 修复端口冲突问题
2. 添加 `stoplat` 子命令
3. 在 `lat` 命令中集成 precheck 步骤

## 1. 端口冲突问题修复

### 问题描述

在之前的实现中，每个 HCA 的脚本生成都从 `cfg.StartPort` 开始分配端口号。这会导致在同一台主机上有多个 HCA 时，不同 HCA 的 server 进程会尝试使用相同的端口，造成端口冲突。

**问题代码示例：**
```go
// 在 generateLatencyScriptForHCA 中
port := cfg.StartPort  // ❌ 每个 HCA 都从起始端口开始

for _, targetHost := range allHosts {
    for _, targetHCA := range cfg.Server.Hca {
        // 使用 port 分配给 server 和 client
        port++
    }
}
```

**端口分配冲突示例：**
```
假设：2 台主机，每台 2 个 HCA (mlx5_0, mlx5_1)，起始端口 20000

错误的分配（旧版）：
  host1:mlx5_0 → 使用端口 20000, 20001, 20002, 20003
  host1:mlx5_1 → 使用端口 20000, 20001, 20002, 20003  ❌ 冲突！
  host2:mlx5_0 → 使用端口 20000, 20001, 20002, 20003  ❌ 冲突！
  host2:mlx5_1 → 使用端口 20000, 20001, 20002, 20003  ❌ 冲突！
```

### 解决方案

引入**全局端口计数器**，在生成脚本时跨主机和 HCA 连续分配端口号。

**修改的函数签名：**

1. **GenerateLatencyScripts**: 维护全局端口计数器
   ```go
   port := cfg.StartPort
   for _, currentHost := range allHosts {
       var err error
       port, err = generateLatencyScriptsForHost(currentHost, allHosts, cfg, port)
       // port 现在是下一个可用端口
   }
   ```

2. **generateLatencyScriptsForHost**: 接收起始端口，返回下一个可用端口
   ```go
   func generateLatencyScriptsForHost(
       currentHost string, 
       allHosts []string, 
       cfg *config.Config, 
       startPort int,  // 新增：起始端口
   ) (int, error) {   // 修改：返回下一个可用端口
       port := startPort
       for _, currentHCA := range cfg.Client.Hca {
           port, err = generateLatencyScriptForHCA(
               currentHost, currentHostIP, currentHCA, allHosts, cfg, port,
           )
       }
       return port, nil  // 返回下一个可用端口
   }
   ```

3. **generateLatencyScriptForHCA**: 接收起始端口，返回下一个可用端口
   ```go
   func generateLatencyScriptForHCA(
       currentHost, currentHostIP, currentHCA string,
       allHosts []string,
       cfg *config.Config,
       startPort int,  // 新增：起始端口
   ) (int, error) {   // 修改：返回下一个可用端口
       port := startPort  // 从传入的端口开始
       
       for _, targetHost := range allHosts {
           for _, targetHCA := range cfg.Server.Hca {
               // 使用当前 port
               port++
           }
       }
       
       return port, nil  // 返回下一个可用端口
   }
   ```

**正确的端口分配（新版）：**
```
假设：2 台主机，每台 2 个 HCA (mlx5_0, mlx5_1)，起始端口 20000

正确的分配：
  host1:mlx5_0 → 使用端口 20000, 20001, 20002, 20003  (4个测试)
  host1:mlx5_1 → 使用端口 20004, 20005, 20006, 20007  (4个测试) ✅
  host2:mlx5_0 → 使用端口 20008, 20009, 20010, 20011  (4个测试) ✅
  host2:mlx5_1 → 使用端口 20012, 20013, 20014, 20015  (4个测试) ✅
  
总共使用端口：20000-20015 (16个端口，无冲突)
```

### 改进后的输出

现在脚本生成时会显示每个 HCA 使用的端口范围：
```
✅ Generated latency scripts for host1:mlx5_0 (ports 20000-20003)
✅ Generated latency scripts for host1:mlx5_1 (ports 20004-20007)
✅ Generated latency scripts for host2:mlx5_0 (ports 20008-20011)
✅ Generated latency scripts for host2:mlx5_1 (ports 20012-20015)
```

### 代码变更总结

| 文件 | 修改内容 |
|------|---------|
| `stream/stream_latency.go` | 添加 `startPort` 参数到两个函数，实现端口连续分配 |
| `stream/stream_latency.go` | 更新返回值类型为 `(int, error)` |
| `stream/stream_latency.go` | 添加端口范围输出到日志 |

### 验证

端口分配公式验证：
```go
// 对于 N 台主机，每台 H 个 HCA
// 总端口数 = N × H × (N-1) × H
// 
// 示例：2 台主机，2 个 HCA
// 总端口 = 2 × 2 × 1 × 2 = 8 ✅
//
// 示例：3 台主机，2 个 HCA  
// 总端口 = 3 × 2 × 2 × 2 = 24 ✅
```

所有单元测试通过：
```bash
$ go test ./stream/ -v -run "TestCalculateTotalLatencyPorts"
PASS
```

---

## 2. 添加 `stoplat` 子命令

### 背景

在延迟测试过程中，如果出现错误或需要中断测试，需要手动 SSH 到每台主机并 kill `ib_write_lat` 进程，非常不便。类似于已有的 `stop` 命令（用于停止 `ib_write_bw`），我们需要一个专门的命令来停止延迟测试。

### 实现

创建 `cmd/stoplat.go` 文件，实现以下功能：

**命令特性：**
- 命令名：`xnetperf stoplat`
- 功能：在所有配置的主机上执行 `killall ib_write_lat`
- 并发执行：使用 goroutine 同时在所有主机上执行
- 智能错误处理：区分"进程未运行"和真正的错误

**代码结构：**
```go
package cmd

import (
    "fmt"
    "os"
    "strings"
    "sync"
    "xnetperf/config"
    "github.com/spf13/cobra"
)

const COMMAND_STOP_LAT = "killall ib_write_lat"

var stopLatCmd = &cobra.Command{
    Use:   "stoplat",
    Short: "Stop all ib_write_lat processes (latency tests)",
    Long: `Stop all running ib_write_lat processes on all configured hosts.
This is useful when latency tests encounter errors or need to be terminated manually.`,
    Run: func(cmd *cobra.Command, args []string) {
        cfg, err := config.LoadConfig(cfgFile)
        if err != nil {
            fmt.Printf("Error reading config: %v\n", err)
            os.Exit(1)
        }
        handleStopLatCommand(cfg)
    },
}

func handleStopLatCommand(cfg *config.Config) {
    // 并发在所有主机上执行 killall ib_write_lat
    // 处理三种情况：
    // 1. 成功 kill 进程
    // 2. 进程本来就不存在
    // 3. 真正的错误（如 SSH 连接失败）
}
```

### 使用示例

**基本使用：**
```bash
# 停止所有延迟测试进程
$ xnetperf stoplat

[INFO] 'stoplat' command initiated. Sending 'killall ib_write_lat' to 4 hosts...

-> Contacting host1...
-> Contacting host2...
-> Contacting host3...
-> Contacting host4...
   [SUCCESS] ✅ On host1: ib_write_lat processes killed.
   [OK] ✅ On host2: No ib_write_lat process was running.
   [SUCCESS] ✅ On host3: ib_write_lat processes killed.
   [ERROR] ❌ On host4: connection refused
      └── Output: ssh: connect to host host4 port 22: Connection refused

[INFO] All 'stoplat' operations complete.
```

**使用场景：**

1. **测试出错时快速清理：**
   ```bash
   $ xnetperf lat
   # 如果测试卡住或报错
   $ xnetperf stoplat  # 立即停止所有进程
   ```

2. **开发调试时：**
   ```bash
   # 启动测试后发现配置错误
   $ xnetperf stoplat
   # 修改配置
   $ xnetperf lat
   ```

3. **集成到自动化脚本：**
   ```bash
   #!/bin/bash
   # 清理环境
   xnetperf stoplat
   
   # 运行测试
   xnetperf lat
   
   # 测试完成后再次清理
   xnetperf stoplat
   ```

### 与 `stop` 命令的对比

| 特性 | `stop` | `stoplat` |
|------|--------|-----------|
| 停止的进程 | `ib_write_bw` | `ib_write_lat` |
| 适用场景 | 带宽测试 | 延迟测试 |
| 命令 | `killall ib_write_bw` | `killall ib_write_lat` |
| 实现文件 | `cmd/stop.go` | `cmd/stoplat.go` |

### 验证

```bash
$ go build .
$ ./xnetperf stoplat --help
Stop all running ib_write_lat processes on all configured hosts.
This is useful when latency tests encounter errors or need to be terminated manually.

Usage:
  xnetperf stoplat [flags]

Flags:
  -h, --help   help for stoplat

Global Flags:
  -c, --config string   config file (default "./config.yaml")
```

---

## 3. 在 `lat` 命令中集成 Precheck 步骤

### 背景

`run` 命令（带宽测试）在执行测试前会进行网卡状态检查（precheck），确保所有网卡处于健康状态。但 `lat` 命令（延迟测试）缺少这个重要的检查步骤，可能导致在网卡故障的情况下浪费时间运行测试。

### 实现

在 `cmd/lat.go` 的 `runLat` 函数中添加 precheck 步骤作为**第 0 步**。

**修改前的工作流程：**
```
1. Generate latency test scripts
2. Run latency tests
3. Monitor test progress
4. Collect latency report files
5. Analyze results and display N×N latency matrix
```

**修改后的工作流程：**
```
0. Precheck - Verify network card status on all hosts  ← 新增
1. Generate latency test scripts
2. Run latency tests
3. Monitor test progress
4. Collect latency report files
5. Analyze results and display N×N latency matrix
```

**代码变更：**
```go
func runLat(cmd *cobra.Command, args []string) {
    fmt.Println("🚀 Starting xnetperf latency testing workflow...")
    fmt.Println(strings.Repeat("=", 60))

    // Load configuration
    cfg, err := config.LoadConfig(cfgFile)
    if err != nil {
        fmt.Printf("❌ Error loading config: %v\n", err)
        os.Exit(1)
    }

    // Step 0: Precheck - Verify network card status before starting tests
    fmt.Println("\n🔍 Step 0/5: Performing network card precheck...")
    if !execPrecheckCommand(cfg) {
        fmt.Printf("❌ Precheck failed! Network cards are not ready. Please fix the issues before running latency tests.\n")
        os.Exit(1)
    }
    fmt.Println("✅ Precheck passed! All network cards are healthy. Proceeding with latency tests...")

    // Step 1: Generate latency scripts
    fmt.Println("\n📋 Step 1/5: Generating latency test scripts...")
    // ... 继续原有流程
}
```

### Precheck 功能说明

Precheck 步骤会检查以下内容：

1. **网卡状态（ibstat）**
   - 检查所有配置的 HCA 是否存在
   - 检查端口链路状态（是否为 Active）
   - 检查物理状态（是否为 LinkUp）

2. **序列号匹配（可选）**
   - 如果配置了序列号，验证网卡序列号是否匹配
   - 防止误用错误的网卡

**Precheck 成功示例：**
```
🔍 Step 0/5: Performing network card precheck...

Checking host1...
  [✓] mlx5_0: Port 1 Active (SN: MT1234567890)
  [✓] mlx5_1: Port 1 Active (SN: MT1234567891)

Checking host2...
  [✓] mlx5_0: Port 1 Active (SN: MT1234567892)
  [✓] mlx5_1: Port 1 Active (SN: MT1234567893)

✅ All network cards are healthy (4/4 ports Active)
✅ Precheck passed! All network cards are healthy. Proceeding with latency tests...
```

**Precheck 失败示例：**
```
🔍 Step 0/5: Performing network card precheck...

Checking host1...
  [✓] mlx5_0: Port 1 Active (SN: MT1234567890)
  [✗] mlx5_1: Port 1 Down (Physical state: Polling)

Checking host2...
  [✓] mlx5_0: Port 1 Active (SN: MT1234567892)
  [✓] mlx5_1: Port 1 Active (SN: MT1234567893)

❌ Network card check failed! 1 port(s) are not in Active state.
❌ Precheck failed! Network cards are not ready. Please fix the issues before running latency tests.
```

### 使用体验改进

**修改前：**
```bash
$ xnetperf lat
🚀 Starting xnetperf latency testing workflow...
📋 Step 1/5: Generating latency test scripts...
▶️  Step 2/5: Running latency tests...
# 测试运行到一半发现网卡故障，浪费时间 ❌
```

**修改后：**
```bash
$ xnetperf lat
🚀 Starting xnetperf latency testing workflow...
🔍 Step 0/5: Performing network card precheck...
❌ Precheck failed! Network cards are not ready.
# 立即发现问题，节省时间 ✅
```

### 帮助文档更新

```bash
$ xnetperf lat --help
Execute the latency testing workflow for measuring network latency between all HCA pairs:

0. Precheck - Verify network card status on all hosts
1. Generate latency test scripts using ib_write_lat (instead of ib_write_bw)
2. Run latency tests
3. Monitor test progress
4. Collect latency report files
5. Analyze results and display N×N latency matrix

Note: Latency testing currently only supports fullmesh mode. If your config uses
a different stream_type, a warning will be shown but testing will continue.

Examples:
  # Execute latency test with default config
  xnetperf lat

  # Execute with custom config file
  xnetperf lat -c /path/to/config.yaml
```

### 验证

所有测试通过，包括：
```bash
$ go test ./cmd/ -v -run "TestParseLatencyReport|TestDisplayLatencyMatrix"
=== RUN   TestParseLatencyReport
--- PASS: TestParseLatencyReport (0.00s)
=== RUN   TestDisplayLatencyMatrix
--- PASS: TestDisplayLatencyMatrix (0.00s)
PASS
ok      xnetperf/cmd    0.024s
```

---

## 总结

### 变更文件

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `stream/stream_latency.go` | 修改 | 修复端口冲突，添加全局端口计数器 |
| `cmd/stoplat.go` | 新增 | 创建 stoplat 子命令 |
| `cmd/lat.go` | 修改 | 添加 precheck 步骤 |

### 测试验证

✅ 所有单元测试通过
✅ 编译成功
✅ 命令行帮助正确显示
✅ 端口分配逻辑验证通过

### 使用建议

1. **端口配置**：确保 `start_port` 和总端口数不会超过 65535
   ```yaml
   start_port: 20000
   # 对于大规模集群，计算总端口数：N × H × (N-1) × H
   ```

2. **测试流程**：
   ```bash
   # 完整的测试流程
   xnetperf lat          # 自动包含 precheck
   
   # 如果需要中断
   xnetperf stoplat      # 快速停止所有延迟测试
   ```

3. **故障排查**：
   - 如果 precheck 失败，先使用 `xnetperf precheck` 详细查看问题
   - 如果测试卡住，使用 `xnetperf stoplat` 清理进程
   - 检查端口范围输出确认没有端口冲突

### 后续改进建议

1. **并行执行**：考虑并行运行延迟测试以缩短总时间
2. **端口池管理**：更智能的端口分配策略
3. **自动重试**：测试失败时自动重试机制
4. **进度显示**：实时显示测试进度百分比

---

## 相关文档

- [延迟测试功能指南](latency-testing-guide.md)
- [延迟表格显示改进](latency-table-improvement.md)
- [延迟目录修复](latency-directory-fix.md)
