# Incast 延迟测试模式实现文档

## 版本信息
- **功能版本**: v0.2.1
- **实现日期**: 2024
- **相关文件**:
  - `stream/stream_latency.go` - 脚本生成逻辑
  - `cmd/lat.go` - 结果展示和分析
  - `server/config_service.go` - 配置验证

## 功能概述

Incast 模式是一种专门的延迟测试拓扑，适用于 **客户端→服务器** 单向通信场景，如：
- AI 训练（多个 GPU 节点向参数服务器汇聚）
- 存储系统（多客户端同时向存储节点写入）
- 数据聚合场景

### Incast vs Fullmesh 对比

| 特性 | Fullmesh | Incast |
|------|----------|--------|
| **拓扑** | N×N 全连接（双向） | 客户端→服务器（单向） |
| **测试方向** | 所有节点互相测试 | 仅 client → server |
| **主机角色** | 所有主机相同角色 | 严格区分 client/server |
| **主机要求** | 可重叠 | **不允许**重叠 |
| **端口计算** | N_hosts × N_hcas × (N_total - 1) | N_clients × client_HCAs × N_servers × server_HCAs |
| **文件命名** | `latency_fullmesh_c_*` | `latency_incast_c_*` |
| **应用场景** | 通用网络性能评估 | 聚合/汇聚流量模式 |

## 配置示例

### 基本 Incast 配置

```yaml
stream_type: incast  # 指定为 incast 模式

server:
  hostname:
    - server-a
    - server-b
  hca:
    - mlx5_0
    - mlx5_1
  ssh:
    user: root
    key_path: /root/.ssh/id_rsa
  
client:
  hostname:
    - client-1
    - client-2
    - client-3
  hca:
    - mlx5_0
    - mlx5_1
  ssh:
    user: root
    key_path: /root/.ssh/id_rsa

latency:
  duration: 10
  size: 2
  iterations: 5000
  start_port: 20000
```

### 端口计算示例

上述配置的端口需求：
```
端口数 = N_clients × client_HCAs × N_servers × server_HCAs
      = 3 clients × 2 HCAs × 2 servers × 2 HCAs
      = 24 ports
```

因此需要确保端口范围 `20000-20023` 可用。

## 配置验证规则

Incast 模式有严格的配置验证：

### ❌ 错误配置（主机重叠）

```yaml
stream_type: incast
server:
  hostname: [node-a, node-b]
client:
  hostname: [node-a, node-c]  # ❌ node-a 同时在 server 和 client 中
```

**错误信息**:
```
incast 模式下，server 和 client 的主机名不能重复，重复的主机: [node-a]
```

### ✅ 正确配置（主机分离）

```yaml
stream_type: incast
server:
  hostname: [node-a, node-b]
client:
  hostname: [node-c, node-d]  # ✅ 完全不同的主机列表
```

## 文件命名规范

### Incast 模式文件名

**服务端**:
```
latency_incast_s_{serverHost}_{serverHCA}_from_{clientHost}_{clientHCA}_p{port}.json
```

示例:
```
latency_incast_s_server-a_mlx5_0_from_client-1_mlx5_0_p20000.json
latency_incast_s_server-a_mlx5_0_from_client-1_mlx5_1_p20001.json
```

**客户端**:
```
latency_incast_c_{clientHost}_{clientHCA}_to_{serverHost}_{serverHCA}_p{port}.json
```

示例:
```
latency_incast_c_client-1_mlx5_0_to_server-a_mlx5_0_p20000.json
latency_incast_c_client-1_mlx5_1_to_server-a_mlx5_0_p20001.json
```

### 与 Fullmesh 模式对比

| 模式 | 文件名前缀 | 示例 |
|------|-----------|------|
| Fullmesh | `latency_fullmesh_c_` | `latency_fullmesh_c_host1_mlx5_0_to_host2_mlx5_1_p20000.json` |
| Incast | `latency_incast_c_` | `latency_incast_c_client1_mlx5_0_to_server1_mlx5_0_p20000.json` |
| Legacy | `latency_c_` | `latency_c_host1_mlx5_0_to_host2_mlx5_1_p20000.json` |

## 使用方法

### 1. 生成脚本

```bash
./xnetperf lat generate --config config_incast.yaml
```

**输出**:
```
📊 Generating latency scripts in INCAST mode (client → server only)
🔢 Total latency ports needed: 24
✅ Total ports to use: 24 (start: 20000, end: 20023)

📝 Generating latency scripts for servers to receive connections...
   Server: server-a
   Server: server-b

✅ Latency scripts generated successfully
   - Script directory: latency_<timestamp>
   - Total ports used: 24
```

### 2. 运行测试

```bash
./xnetperf lat run --config config_incast.yaml
```

### 3. 分析结果

```bash
./xnetperf lat analyze --config config_incast.yaml
```

## 结果展示

### 延迟矩阵表格

Incast 模式显示 **客户端 × 服务器** 矩阵（非方阵）：

```
================================================================================
📊 Latency Matrix - INCAST Mode (Client → Server)
   Average Latency in microseconds
================================================================================
┌────────────┬────────────┬─────────────────────────────────────────┐
│            │            │              server-a    │    server-b  │
│            │            ├──────────────┬───────────┼──────────────┤
│            │            │    mlx5_0    │  mlx5_1   │   mlx5_0     │
├────────────┼────────────┼──────────────┼───────────┼──────────────┤
│ client-1   │ mlx5_0     │     1.23 μs  │  1.25 μs  │    1.28 μs   │
│            ├────────────┼──────────────┼───────────┼──────────────┤
│            │ mlx5_1     │     1.24 μs  │  1.26 μs  │    1.29 μs   │
├────────────┼────────────┼──────────────┼───────────┼──────────────┤
│ client-2   │ mlx5_0     │     1.30 μs  │  1.32 μs  │    1.35 μs   │
│            ├────────────┼──────────────┼───────────┼──────────────┤
│            │ mlx5_1     │     1.31 μs  │  1.33 μs  │    1.36 μs   │
└────────────┴────────────┴──────────────┴───────────┴──────────────┘
```

### 统计信息

#### 1. 全局统计
```
🌐 Global Statistics:
   Total measurements: 8
   Minimum latency:    1.23 μs
   Maximum latency:    1.36 μs
   Average latency:    1.29 μs
```

#### 2. 每服务器平均延迟
```
🖥️  Per-Server Average Latency:
   server-a:mlx5_0                1.27 μs  (2 clients)
   server-a:mlx5_1                1.29 μs  (2 clients)
   server-b:mlx5_0                1.32 μs  (2 clients)
```

#### 3. 每客户端平均延迟
```
💻 Per-Client Average Latency:
   client-1:mlx5_0                1.25 μs  (2 servers)
   client-1:mlx5_1                1.26 μs  (2 servers)
   client-2:mlx5_0                1.32 μs  (2 servers)
   client-2:mlx5_1                1.33 μs  (2 servers)
```

## 实现细节

### 脚本生成流程

```go
// stream/stream_latency.go

func GenerateLatencyScripts(cfg *config.Config) {
    if cfg.StreamType == config.InCast {
        generateLatencyScriptsIncast(cfg)
    } else {
        generateLatencyScriptsFullmesh(cfg)
    }
}

func generateLatencyScriptsIncast(cfg *config.Config) {
    // 1. 为每个 server 生成接收脚本
    for _, serverHost := range cfg.Server.Hostname {
        generateLatencyScriptForServerIncast(cfg, serverHost)
    }
}

func generateLatencyScriptForServerIncast(cfg *config.Config, serverHost string) {
    // 2. 为每个 server HCA 生成脚本
    for _, serverHCA := range cfg.Server.HCA {
        generateLatencyScriptForServerHCAIncast(cfg, serverHost, serverHCA, ...)
    }
}

func generateLatencyScriptForServerHCAIncast(...) {
    // 3. 为所有 client 生成连接命令
    for _, clientHost := range cfg.Client.Hostname {
        for _, clientHCA := range cfg.Client.HCA {
            // 生成 server 监听命令
            // 生成 client 连接命令
            // 文件名: latency_incast_c_{client}_{clientHCA}_to_{server}_{serverHCA}_p{port}.json
        }
    }
}
```

### 端口计算公式

```go
func calculateTotalLatencyPortsIncast(cfg *config.Config) int {
    numServers := len(cfg.Server.Hostname)
    numServerHcas := len(cfg.Server.HCA)
    numClients := len(cfg.Client.Hostname)
    numClientHcas := len(cfg.Client.HCA)
    
    return numServers * numServerHcas * numClients * numClientHcas
}
```

### 配置验证

```go
// server/config_service.go

func ValidateConfig(cfg *config.Config) []string {
    if cfg.StreamType == config.InCast {
        // 检查 server 和 client 主机名是否重叠
        serverHostMap := make(map[string]bool)
        for _, host := range cfg.Server.Hostname {
            serverHostMap[host] = true
        }
        
        var duplicateHosts []string
        for _, host := range cfg.Client.Hostname {
            if serverHostMap[host] {
                duplicateHosts = append(duplicateHosts, host)
            }
        }
        
        if len(duplicateHosts) > 0 {
            return []string{fmt.Sprintf(
                "incast 模式下，server 和 client 的主机名不能重复，重复的主机: %v",
                duplicateHosts,
            )}
        }
    }
    return nil
}
```

## 应用场景

### 1. AI 训练参数聚合
```yaml
# 多 GPU 节点向参数服务器汇聚梯度
stream_type: incast
server:
  hostname: [param-server-1, param-server-2]
client:
  hostname: [gpu-worker-1, gpu-worker-2, gpu-worker-3, gpu-worker-4]
```

### 2. 分布式存储
```yaml
# 多客户端同时向存储节点写入
stream_type: incast
server:
  hostname: [storage-node-1, storage-node-2, storage-node-3]
client:
  hostname: [app-server-1, app-server-2, ..., app-server-10]
```

### 3. 数据库主从复制压力测试
```yaml
# 多从节点同步主节点数据
stream_type: incast
server:
  hostname: [db-master]
client:
  hostname: [db-slave-1, db-slave-2, db-slave-3]
```

## 与其他模式的集成

### 模式切换

在同一套基础设施上测试不同模式：

1. **Fullmesh 模式** - 评估整体网络健康度
   ```bash
   ./xnetperf lat generate --config config_fullmesh.yaml
   ```

2. **Incast 模式** - 评估汇聚场景性能
   ```bash
   ./xnetperf lat generate --config config_incast.yaml
   ```

### 结果对比

可以同时保留两种模式的结果进行对比分析：
- Fullmesh: 发现网络瓶颈和异常节点
- Incast: 评估特定汇聚场景下的性能表现

## 注意事项

1. **主机配置严格性**: Incast 模式下 server 和 client 主机列表必须完全不重叠
2. **端口需求**: Incast 端口数 = clients × client_HCAs × servers × server_HCAs
3. **文件识别**: 通过文件名前缀 `latency_incast_` 识别，确保不与 fullmesh 结果混淆
4. **单向测试**: 只测试 client → server，不测试反向

## 未来扩展

- [ ] 支持多 server 端口复用（减少端口需求）
- [ ] 添加 incast 模式的网络拥塞检测
- [ ] 支持分组 incast（按组测试不同 server 集群）
- [ ] 添加 incast 模式的 QoS 测试

## 相关文档

- [延迟测试用户指南](latency-testing-guide.md)
- [配置文件验证](config-validation.md)
- [v0.2.1 版本总结](v0.2.1-summary.md)
