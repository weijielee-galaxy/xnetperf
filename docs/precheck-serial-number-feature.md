# Precheck Serial Number Column Feature

## 概述

该功能在 `xnetperf precheck` 命令中添加了 Serial Number 列，用于显示系统的序列号，提供更好的资产跟踪和系统识别能力。

## 功能特性

### 新增列位置
- **列名**: Serial Number
- **位置**: 在 Board ID 和 Status 列之间
- **数据来源**: `/sys/class/dmi/id/product_serial`

### 表格布局
```
┌─────────────────┬──────────┬────────────────┬───────────────┬─────────────┬──────────────┬─────────────────┬─────────────────┬───────────────┐
│ Hostname        │ HCA      │ Physical State │ Logical State │ Speed       │ FW Version   │ Board ID        │ Serial Number   │ Status        │
├─────────────────┼──────────┼────────────────┼───────────────┼─────────────┼──────────────┼─────────────────┼─────────────────┼───────────────┤
│ server-001      │ mlx5_0   │ LinkUp         │ ACTIVE        │ 200 Gb/sec │ 28.43.2026   │ MT_0000000844   │ ABC123456789    │ [+] HEALTHY   │
└─────────────────┴──────────┴────────────────┴───────────────┴─────────────┴──────────────┴─────────────────┴─────────────────┴───────────────┘
```

## 技术实现

### 数据收集
- 通过 SSH 执行 `cat /sys/class/dmi/id/product_serial` 命令获取序列号
- 支持远程主机和本地主机的序列号收集

### 错误处理
- 当无法访问 `/sys/class/dmi/id/product_serial` 时显示 "N/A"
- 当 SSH 连接失败时显示 "N/A"
- 当文件不存在或权限不足时显示 "N/A"

### 动态列宽
- 自动计算 Serial Number 列的最大宽度
- 最小宽度: 13 字符（列标题 "Serial Number" 长度）
- 根据实际序列号长度动态调整

### 代码修改

#### 1. 结构体更新
```go
type PrecheckResult struct {
    Hostname       string
    HCA           string
    PhysState     string
    State         string
    Speed         string
    FwVer         string
    BoardId       string
    SerialNumber  string  // 新增字段
    IsHealthy     bool
    Error         string
}
```

#### 2. 数据收集逻辑
```go
// 获取系统序列号
serialNumberCmd := "cat /sys/class/dmi/id/product_serial"
serialNumberOutput, err := ExecuteSSHCommand(hostname, serialNumberCmd, true)
var serialNumberStr string
if err == nil && serialNumberOutput != "" {
    serialNumberStr = strings.TrimSpace(serialNumberOutput)
} else {
    serialNumberStr = ""
}
```

#### 3. 表格显示更新
- 更新表格格式从 8 列扩展到 9 列
- 添加动态列宽计算: `maxSerialNumberWidth`
- 更新所有边框和分隔符格式

## 使用示例

### 基本用法
```bash
xnetperf precheck -c config.yaml
```

### 输出示例
```
=== Precheck Results (10:00:04) ===
┌─────────────────┬──────────┬────────────────┬───────────────┬─────────────────────┬──────────────┬─────────────────┬─────────────────┬───────────────┐
│ Hostname        │ HCA      │ Physical State │ Logical State │ Speed               │ FW Version   │ Board ID        │ Serial Number   │ Status        │
├─────────────────┼──────────┼────────────────┼───────────────┼─────────────────────┼──────────────┼─────────────────┼─────────────────┼───────────────┤
│ server-001      │ mlx5_0   │ LinkUp         │ ACTIVE        │ 200 Gb/sec (2X NDR) │ 28.43.2026   │ MT_0000000844   │ ABC123456789    │ [+] HEALTHY   │
│                 │ mlx5_1   │ LinkUp         │ ACTIVE        │ 200 Gb/sec (2X NDR) │ 28.43.2026   │ MT_0000000844   │ ABC123456789    │ [+] HEALTHY   │
├─────────────────┼──────────┼────────────────┼───────────────┼─────────────────────┼──────────────┼─────────────────┼─────────────────┼───────────────┤
│ server-002      │ mlx5_0   │ N/A            │ N/A           │ N/A                 │ N/A          │ N/A             │ N/A             │ [!] ERROR     │
│                 │ mlx5_1   │ Polling        │ INIT          │ 100 Gb/sec (1X HDR) │ 28.43.2025   │ MT_0000000845   │ DEF987654321    │ [X] UNHEALTHY │
└─────────────────┴──────────┴────────────────┴───────────────┴─────────────────────┴──────────────┴─────────────────┴─────────────────┴───────────────┘
Summary: 2 healthy, 1 unhealthy, 1 errors (Total: 4 HCAs)
```

## 应用场景

### 1. 资产管理
- 通过序列号跟踪硬件资产
- 识别特定硬件型号和批次
- 支持硬件维护记录管理

### 2. 系统识别
- 在大规模集群中快速识别特定节点
- 区分相同配置的不同物理机器
- 支持硬件故障追溯

### 3. 合规性检查
- 验证硬件配置符合预期
- 检查是否存在未授权硬件
- 支持审计要求

## 兼容性

### 支持的系统
- ✅ Linux 系统 (通过 `/sys/class/dmi/id/product_serial`)
- ✅ 具有 DMI 信息的 x86_64 系统
- ✅ 虚拟机环境 (如果 hypervisor 提供序列号)

### 限制
- ⚠️ 某些嵌入式系统可能不提供序列号信息
- ⚠️ 需要读取系统 DMI 信息的权限
- ⚠️ 容器环境可能无法访问主机序列号

## 测试覆盖

### 单元测试
- ✅ 序列号字段显示测试
- ✅ 表格格式对齐测试  
- ✅ 错误处理测试 (N/A 显示)
- ✅ 动态列宽计算测试

### 集成测试
- ✅ SSH 连接失败场景
- ✅ 文件不存在场景
- ✅ 权限不足场景
- ✅ 多主机混合状态场景

## 版本历史

- **v0.1.3**: 首次添加 Serial Number 列功能
- 支持动态列宽计算
- 完整的错误处理和测试覆盖

## 注意事项

1. **权限要求**: 读取 `/sys/class/dmi/id/product_serial` 通常需要管理员权限
2. **性能影响**: 每个主机增加一次 SSH 命令执行，对性能影响很小
3. **数据隐私**: 序列号可能包含敏感信息，请注意日志和输出的安全性
4. **网络依赖**: 远程主机需要 SSH 连接正常才能获取序列号