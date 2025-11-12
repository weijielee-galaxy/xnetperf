# 连通性检查功能

## 概述

连通性检查功能用于测试网络中所有 HCA（Host Channel Adapter）之间的双向连通性。该功能通过执行两次 incast 延迟测试来检查所有客户端到服务器以及服务器到客户端的连接状态。

## 使用方法

### CLI 命令

```bash
# 使用默认配置文件
xnetperf check-conn

# 使用自定义配置文件
xnetperf check-conn -c /path/to/config.yaml

# 查看帮助
xnetperf check-conn --help
```

### API 接口

```bash
# 执行连通性检查
curl -X POST http://localhost:8080/api/configs/config.yaml/connectivity
```

## 设计原理

### 核心思想

- **代码复用**：完全复用 `latency_runner.ParseLatencyReportsFromDir()` 来解析测试报告
- **测试策略**：使用 2 次 incast 测试代替 N×N 次独立测试
  - 第 1 次：client → server (所有客户端到所有服务器)
  - 第 2 次：server → client (交换角色后，所有服务器到所有客户端)
- **短时测试**：使用 5 秒的短测试持续时间，快速检查连通性

### 架构设计

```
connectivity.Checker
├── CheckConnectivity()          # 主入口：执行双向连通性检查
├── runConnectivityTest()        # 运行单次测试（复用 script.Executor）
├── collectReports()             # 收集报告（复用 collect.Collector）
├── parseConnectivityResults()   # 解析结果（复用 lat.ParseLatencyReportsFromDir）
├── swapClientServer()           # 交换客户端/服务器角色
└── buildSummary()               # 构建汇总结果
```

## 数据结构

### ConnectivityResult

单个 HCA 对的连通性结果：

```go
type ConnectivityResult struct {
    SourceHost   string  // 源主机名
    SourceHCA    string  // 源 HCA
    TargetHost   string  // 目标主机名
    TargetHCA    string  // 目标 HCA
    Connected    bool    // 是否连通
    AvgLatencyUs float64 // 平均延迟（微秒）
    MinLatencyUs float64 // 最小延迟（微秒）
    MaxLatencyUs float64 // 最大延迟（微秒）
    Error        string  // 错误信息（如有）
}
```

### ConnectivitySummary

连通性检查的汇总结果：

```go
type ConnectivitySummary struct {
    TotalPairs        int                   // 总测试对数
    ConnectedPairs    int                   // 成功连接的对数
    DisconnectedPairs int                   // 未连接的对数
    ErrorPairs        int                   // 错误的对数
    Results           []ConnectivityResult  // 详细结果列表
}
```

## API 接口

### 端点

```
POST /api/configs/:name/connectivity
```

### 请求示例

```bash
curl -X POST http://localhost:8080/api/configs/config.yaml/connectivity
```

### 响应示例

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total_pairs": 16,
    "connected_pairs": 15,
    "disconnected_pairs": 0,
    "error_pairs": 1,
    "results": [
      {
        "source_host": "host1",
        "source_hca": "mlx5_0",
        "target_host": "host2",
        "target_hca": "mlx5_0",
        "connected": true,
        "avg_latency_us": 2.5,
        "min_latency_us": 2.1,
        "max_latency_us": 3.2,
        "error": ""
      },
      ...
    ]
  }
}
```

## 工作流程

1. **保存原始配置**
   - 保存 `stream_type`、`duration_seconds`、`infinitely` 等配置

2. **配置测试参数**
   - 设置 `stream_type = incast`
   - 设置 `duration_seconds = 5`（短测试）
   - 设置 `infinitely = false`

3. **执行 Client → Server 测试**
   - 运行 incast 延迟测试
   - 监控进程状态（使用 `MonitorProgressWithTimeout`，600秒超时）
   - 收集测试报告
   - 解析连通性结果

4. **执行 Server → Client 测试**
   - 交换 client/server 角色
   - 运行 incast 延迟测试
   - 监控进程状态（使用 `MonitorProgressWithTimeout`，600秒超时）
   - 收集测试报告
   - 解析连通性结果
   - 恢复 client/server 角色

5. **合并结果并返回**
   - 合并两次测试的结果
   - 构建汇总统计信息
   - 返回完整的连通性报告

6. **恢复原始配置**
   - 恢复所有原始配置参数

## 代码复用

### 复用的组件

1. **script.Executor**
   - 用于执行测试脚本生成和运行
   - 测试类型：`script.TestTypeLatency`

2. **collect.Collector**
   - 用于从远程主机收集测试报告
   - 自动清理远程文件

3. **lat.ParseLatencyReportsFromDir()**
   - 解析延迟测试 JSON 报告
   - 提取延迟数据（avg/min/max）
   - 解析文件名（host/HCA 信息）

### 优势

- **无重复代码**：所有核心逻辑均复用现有组件
- **一致性**：与延迟测试使用相同的测试工具和解析逻辑
- **可维护性**：修改延迟测试逻辑时，连通性检查自动受益

## 使用场景

1. **网络部署验证**
   - 在新部署的集群中快速验证所有 HCA 之间的连通性

2. **故障排查**
   - 检测哪些 HCA 对无法建立连接
   - 识别延迟异常的连接

3. **自动化测试**
   - 在性能测试前自动验证网络连通性
   - 确保所有必要的连接都可用

## 注意事项

1. **配置要求**
   - 必须启用报告生成：`report.enable = true`
   - 建议使用 incast 模式的配置

2. **测试时长**
   - 连通性检查会强制使用 5 秒测试时长
   - 原始配置会在测试后恢复

3. **资源占用**
   - 会执行 2 次完整的 incast 测试
   - 测试期间会占用网络带宽和计算资源

## 相关文档

- [延迟测试指南](./latency-testing-guide.md)
- [HTTP Server API](./http-server-api.md)
- [配置文件说明](./README.md)
