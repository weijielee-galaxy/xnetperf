# 延迟测试目录修复

## 问题描述

在执行 `xnetperf lat` 命令时，出现以下问题：

1. **目录混淆**: 提示 "Cleared stream script directory: ./generated_scripts_incast"，使用了 incast 目录而不是专用的延迟测试目录
2. **配置混淆**: 延迟测试受到 `stream_type` 配置的影响，实际应该忽略该配置
3. **脚本执行不透明**: 无法看到正在执行的脚本命令
4. **结果丢失**: 由于目录问题，测试结果无法正确收集

## 根本原因

延迟测试代码错误地使用了 `cfg.OutputDir()`，该函数根据 `stream_type` 生成目录名：
```go
func (c *Config) OutputDir() string {
    return fmt.Sprintf("%s_%s", c.OutputBase, c.StreamType)
}
```

当 `stream_type: incast` 时，目录变成 `./generated_scripts_incast`，导致：
- 延迟测试脚本和 incast 脚本混在一起
- 收集报告时找不到正确的文件
- 测试结果互相覆盖

## 修复方案

### 1. 创建专用目录函数

新增 `getLatencyOutputDir()` 函数，为延迟测试提供独立目录：

```go
// getLatencyOutputDir returns the output directory for latency tests
func getLatencyOutputDir(cfg *config.Config) string {
    return fmt.Sprintf("%s_latency", cfg.OutputBase)
}
```

这确保延迟测试始终使用 `./generated_scripts_latency` 目录，不受 `stream_type` 影响。

### 2. 独立的目录清理函数

新增 `clearLatencyScriptDir()` 函数：

```go
func clearLatencyScriptDir(cfg *config.Config) {
    dir := getLatencyOutputDir(cfg)
    fmt.Printf("Clearing latency script directory: %s\n", dir)
    // ... 清理逻辑
    fmt.Printf("Cleared latency script directory: %s\n", dir)
}
```

### 3. 更新所有目录引用

将所有 `cfg.OutputDir()` 替换为 `getLatencyOutputDir(cfg)`：

**修改的位置：**
- `GenerateLatencyScripts()` - 脚本生成
- `generateLatencyScriptForHCA()` - 单个 HCA 脚本生成
- `RunLatencyScripts()` - 脚本执行

### 4. 改进提示信息

**之前：**
```
⚠️  Warning: Latency testing currently only supports fullmesh mode. Current mode: incast
Continuing with latency test generation...
```

**现在：**
```
⚠️  Note: Config stream_type is 'incast', but latency testing uses full-mesh topology
Clearing latency script directory: ./generated_scripts_latency
Cleared latency script directory: ./generated_scripts_latency
```

更清晰地说明延迟测试会忽略 `stream_type` 配置。

### 5. 添加脚本执行打印

在生成和执行脚本时，打印具体命令：

**生成时：**
```go
fmt.Printf("   Server script preview (first command): %s\n", serverLines[0])
fmt.Printf("   Client script preview (first command): %s\n", clientLines[0])
```

**执行时：**
```go
fmt.Printf("  Executing: bash %s\n", serverScript)
fmt.Printf("    → Running: bash %s\n", scriptPath)
```

### 6. 修复脚本执行逻辑

更新 `executeScript()` 函数以真正执行脚本：

```go
func executeScript(scriptPath string) error {
    // Check if script exists
    if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
        return fmt.Errorf("script does not exist: %s", scriptPath)
    }

    // Print script path for debugging
    fmt.Printf("    → Running: bash %s\n", scriptPath)
    
    // Read script content
    content, err := os.ReadFile(scriptPath)
    if err != nil {
        return fmt.Errorf("failed to read script %s: %v", scriptPath, err)
    }
    
    // Execute via bash
    cmd := exec.Command("bash", "-c", string(content))
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("failed to start script %s: %v", scriptPath, err)
    }
    
    return nil
}
```

## 文件变更

### 修改的文件

**stream/stream_latency.go**
- ✅ 新增 `getLatencyOutputDir()` 函数
- ✅ 新增 `clearLatencyScriptDir()` 函数
- ✅ 更新 `GenerateLatencyScripts()` - 使用独立目录
- ✅ 更新 `generateLatencyScriptForHCA()` - 使用独立目录
- ✅ 更新 `RunLatencyScripts()` - 使用独立目录并打印执行信息
- ✅ 更新 `executeScript()` - 真正执行脚本
- ✅ 添加脚本预览打印
- ✅ 导入 `os/exec` 包

## 修复效果

### 修复前
```bash
$ xnetperf lat
Cleared stream script directory: ./generated_scripts_incast  # ❌ 错误目录
⚠️  Warning: Latency testing currently only supports fullmesh mode. Current mode: incast
# ... 脚本生成到错误目录
# 无法看到执行的命令
# 结果收集失败
```

### 修复后
```bash
$ xnetperf lat
⚠️  Note: Config stream_type is 'incast', but latency testing uses full-mesh topology
Clearing latency script directory: ./generated_scripts_latency  # ✅ 正确目录
Cleared latency script directory: ./generated_scripts_latency
Total ports needed for latency testing: 24
Host host1 IP: 192.168.1.10
✅ Generated latency scripts for host1:mlx5_0
   Server script preview (first command): ssh host1 'ib_write_lat ...'  # ✅ 可见命令
   Client script preview (first command): ssh host2 'ib_write_lat ...'
...
Phase 1: Starting all server processes...
  Executing: bash ./generated_scripts_latency/host1_mlx5_0_server_latency.sh  # ✅ 清晰执行
    → Running: bash ./generated_scripts_latency/host1_mlx5_0_server_latency.sh
...
✅ Successfully generated latency test scripts in ./generated_scripts_latency
```

## 目录结构

修复后的目录结构：

```
./
├── generated_scripts_fullmesh/     # 全连接测试脚本
├── generated_scripts_incast/       # Incast 测试脚本
├── generated_scripts_p2p/          # P2P 测试脚本
├── generated_scripts_latency/      # ✅ 延迟测试专用目录（新增）
│   ├── host1_mlx5_0_server_latency.sh
│   ├── host1_mlx5_0_client_latency.sh
│   ├── host2_mlx5_0_server_latency.sh
│   ├── host2_mlx5_0_client_latency.sh
│   └── ...
└── reports/                        # 所有测试报告
    ├── latency_s_*.json           # 延迟服务端报告
    ├── latency_c_*.json           # 延迟客户端报告
    ├── report_s_*.json            # 带宽服务端报告
    └── report_c_*.json            # 带宽客户端报告
```

## 兼容性

- ✅ **完全向后兼容** - 不影响现有的带宽测试
- ✅ **独立目录** - 延迟测试和带宽测试完全隔离
- ✅ **清晰提示** - 用户知道发生了什么
- ✅ **可调试** - 可以看到执行的命令

## 测试验证

```bash
# 编译测试
✅ go build . 成功

# 单元测试
✅ go test ./stream/ -v 通过

# 功能测试
✅ xnetperf lat 使用正确目录
✅ 脚本生成到 ./generated_scripts_latency
✅ 可以看到执行的命令
✅ 不受 stream_type 配置影响
```

## 后续改进建议

1. **统一执行逻辑** - 考虑将脚本执行逻辑统一到一个函数
2. **并发执行** - 可以并发执行多个服务器/客户端脚本
3. **进度显示** - 显示脚本执行进度百分比
4. **错误收集** - 收集所有脚本执行错误统一显示
5. **日志文件** - 将脚本输出保存到日志文件

---
**修复日期**: 2024-10-20  
**版本**: v0.2.0  
**影响范围**: 延迟测试功能
