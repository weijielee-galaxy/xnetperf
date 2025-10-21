# Incast 模式脚本生成修复文档

## 问题描述

在 v0.2.1 incast 模式的初始实现中，脚本生成逻辑存在严重问题，导致运行时找不到 client 主机的脚本文件。

### 错误信息

```bash
Executing: bash ./generated_scripts_latency/cetus-g88-061_mlx5_0_server_latency.sh
❌ Error running latency scripts: failed to execute server script 
   ./generated_scripts_latency/cetus-g88-061_mlx5_0_server_latency.sh: 
   script does not exist
```

### 根本原因

**错误的实现逻辑**（已修复前）：

```go
// ❌ 错误：按 server 组织脚本
func generateLatencyScriptsIncast(cfg *config.Config) error {
    // 只为 server 主机生成脚本
    for _, serverHost := range cfg.Server.Hostname {
        generateLatencyScriptsForServerIncast(serverHost, cfg, port)
    }
}

func generateLatencyScriptForServerHCAIncast(...) {
    // 脚本文件名使用 server 主机名
    serverScriptFileName := fmt.Sprintf("%s/%s_%s_server_latency.sh",
        outputDir, serverHost, serverHCA)  // ❌ 使用 server 名称
    clientScriptFileName := fmt.Sprintf("%s/%s_%s_client_latency.sh",
        outputDir, serverHost, serverHCA)  // ❌ 使用 server 名称
}
```

**问题分析**：

1. **脚本只为 server 主机生成**：
   - 遍历 `cfg.Server.Hostname`
   - Client 主机完全没有对应的脚本文件

2. **Client 脚本命名错误**：
   - 使用 `{serverHost}_{serverHCA}_client_latency.sh`
   - 应该使用 `{clientHost}_{clientHCA}_client_latency.sh`

3. **执行失败**：
   - 运行时尝试执行 `cetus-g88-061_mlx5_0_server_latency.sh`
   - 该文件不存在（因为没有为 client 主机生成）

## 修复方案

### 核心思想转变

从 **"按 server 组织"** 改为 **"按 client 组织"**：

| 维度 | 错误实现 | 正确实现 |
|------|---------|---------|
| **遍历对象** | Server 主机 | **Client 主机** |
| **脚本文件名** | 以 server 命名 | **以 client 命名** |
| **生成数量** | N_servers × N_server_HCAs | **N_clients × N_client_HCAs** |
| **文件归属** | Server 主机的脚本目录 | **Client 主机的脚本目录** |

### 修复后的实现

```go
// ✅ 正确：按 client 组织脚本
func generateLatencyScriptsIncast(cfg *config.Config) error {
    fmt.Printf("📊 Generating latency scripts in INCAST mode (client → server only)\n")

    // 为每个 client 生成脚本（每个 client 测试所有 servers）
    port := cfg.StartPort
    for _, clientHost := range cfg.Client.Hostname {
        port, err = generateLatencyScriptsForClientIncast(clientHost, cfg, port)
    }
    
    return nil
}

// ✅ 为单个 client 主机生成脚本
func generateLatencyScriptsForClientIncast(clientHost string, cfg *config.Config, startPort int) (int, error) {
    // 获取所有 server 的 IP 地址
    serverIPs := make(map[string]string)
    for _, serverHost := range cfg.Server.Hostname {
        output, err := getHostIP(serverHost, cfg.SSH.PrivateKey, cfg.NetworkInterface)
        serverIPs[serverHost] = strings.TrimSpace(string(output))
    }

    // 为该 client 的每个 HCA 生成脚本
    port := startPort
    for _, clientHCA := range cfg.Client.Hca {
        port, err = generateLatencyScriptForClientHCAIncast(
            clientHost, clientHCA, serverIPs, cfg, port,
        )
    }

    return port, nil
}

// ✅ 为单个 client HCA 生成脚本
func generateLatencyScriptForClientHCAIncast(
    clientHost, clientHCA string,
    serverIPs map[string]string,
    cfg *config.Config,
    startPort int,
) (int, error) {
    outputDir := getLatencyOutputDir(cfg)
    
    // ✅ 正确：使用 client 主机名和 HCA 命名
    serverScriptFileName := fmt.Sprintf("%s/%s_%s_server_latency.sh",
        outputDir, clientHost, clientHCA)  // ✅ 使用 client 名称
    clientScriptFileName := fmt.Sprintf("%s/%s_%s_client_latency.sh",
        outputDir, clientHost, clientHCA)  // ✅ 使用 client 名称

    port := startPort

    // 该 client HCA 测试所有 server HCAs
    for _, serverHost := range cfg.Server.Hostname {
        serverIP := serverIPs[serverHost]
        for _, serverHCA := range cfg.Server.Hca {
            // 生成 server 命令（运行在 server 主机上）
            serverCmd := NewIBWriteBWCommandBuilder().
                Host(serverHost).       // server 主机
                Device(serverHCA).      // server HCA
                Port(port).
                ForLatencyTest(true).
                // ...
                ServerCommand()

            // 生成 client 命令（运行在 client 主机上）
            clientCmd := NewIBWriteBWCommandBuilder().
                Host(clientHost).       // client 主机
                Device(clientHCA).      // client HCA
                TargetIP(serverIP).     // 连接到 server
                Port(port).
                ForLatencyTest(true).
                // ...
                ClientCommand()

            serverScriptContent.WriteString(serverCmd.String() + "\n")
            clientScriptContent.WriteString(clientCmd.String() + "\n")

            port++
        }
    }

    // 写入脚本文件
    os.WriteFile(serverScriptFileName, []byte(serverScriptContent.String()), 0755)
    os.WriteFile(clientScriptFileName, []byte(clientScriptContent.String()), 0755)

    return port, nil
}
```

## 修复前后对比

### 配置示例

```yaml
stream_type: incast
server:
  hostname: [server-a, server-b]
  hca: [mlx5_0, mlx5_1]
client:
  hostname: [client-1, client-2]
  hca: [mlx5_0]
```

### 生成的脚本文件

| 修复前（错误） | 修复后（正确） |
|--------------|--------------|
| ❌ `server-a_mlx5_0_server_latency.sh` | ✅ `client-1_mlx5_0_server_latency.sh` |
| ❌ `server-a_mlx5_0_client_latency.sh` | ✅ `client-1_mlx5_0_client_latency.sh` |
| ❌ `server-a_mlx5_1_server_latency.sh` | ✅ `client-2_mlx5_0_server_latency.sh` |
| ❌ `server-a_mlx5_1_client_latency.sh` | ✅ `client-2_mlx5_0_client_latency.sh` |
| ❌ `server-b_mlx5_0_server_latency.sh` | （共 4 个文件，每个 client HCA 一对） |
| ❌ `server-b_mlx5_0_client_latency.sh` |  |
| ❌ `server-b_mlx5_1_server_latency.sh` |  |
| ❌ `server-b_mlx5_1_client_latency.sh` |  |
| （共 8 个文件，每个 server HCA 一对） |  |

### 脚本内容对比

#### 修复前 - server-a_mlx5_0_server_latency.sh
```bash
# ❌ 问题：该脚本应该在 client-1 上运行，但文件名用的是 server-a
ssh server-a 'ib_write_lat -d mlx5_0 ...'   # server-a:mlx5_0 监听
ssh server-a 'ib_write_lat -d mlx5_1 ...'   # server-a:mlx5_1 监听
# ... 所有针对 server-a 的监听命令
```

#### 修复后 - client-1_mlx5_0_server_latency.sh
```bash
# ✅ 正确：脚本在 client-1 上运行，内容包含所有需要启动的 server 监听命令
ssh server-a 'ib_write_lat -d mlx5_0 ...'   # server-a:mlx5_0 监听
ssh server-a 'ib_write_lat -d mlx5_1 ...'   # server-a:mlx5_1 监听
ssh server-b 'ib_write_lat -d mlx5_0 ...'   # server-b:mlx5_0 监听
ssh server-b 'ib_write_lat -d mlx5_1 ...'   # server-b:mlx5_1 监听
```

#### 修复后 - client-1_mlx5_0_client_latency.sh
```bash
# ✅ 正确：从 client-1:mlx5_0 发起到所有 servers 的连接
ssh client-1 'ib_write_lat -d mlx5_0 -p 20000 <server-a-ip> ...'  # → server-a:mlx5_0
ssh client-1 'ib_write_lat -d mlx5_0 -p 20001 <server-a-ip> ...'  # → server-a:mlx5_1
ssh client-1 'ib_write_lat -d mlx5_0 -p 20002 <server-b-ip> ...'  # → server-b:mlx5_0
ssh client-1 'ib_write_lat -d mlx5_0 -p 20003 <server-b-ip> ...'  # → server-b:mlx5_1
```

## 技术细节

### 端口分配策略

修复后的端口分配保持一致：

```
端口计算：N_clients × N_client_HCAs × N_servers × N_server_HCAs
示例：2 clients × 1 HCA × 2 servers × 2 HCAs = 8 ports

端口分配顺序：
- client-1:mlx5_0 → server-a:mlx5_0  (port 20000)
- client-1:mlx5_0 → server-a:mlx5_1  (port 20001)
- client-1:mlx5_0 → server-b:mlx5_0  (port 20002)
- client-1:mlx5_0 → server-b:mlx5_1  (port 20003)
- client-2:mlx5_0 → server-a:mlx5_0  (port 20004)
- client-2:mlx5_0 → server-a:mlx5_1  (port 20005)
- client-2:mlx5_0 → server-b:mlx5_0  (port 20006)
- client-2:mlx5_0 → server-b:mlx5_1  (port 20007)
```

### Server IP 获取优化

修复后一次性获取所有 server IP，避免重复查询：

```go
// ✅ 优化：在外层获取所有 server IPs
serverIPs := make(map[string]string)
for _, serverHost := range cfg.Server.Hostname {
    output, err := getHostIP(serverHost, cfg.SSH.PrivateKey, cfg.NetworkInterface)
    serverIPs[serverHost] = strings.TrimSpace(string(output))
}

// 传递给内层函数使用
generateLatencyScriptForClientHCAIncast(clientHost, clientHCA, serverIPs, cfg, port)
```

## 测试验证

### 单元测试

所有现有测试通过：

```bash
$ go test ./stream/... -v
--- PASS: TestCalculateTotalLatencyPortsFullmesh (0.00s)
--- PASS: TestGenerateLatencyScriptForHCA (0.00s)
--- PASS: TestGenerateIncastScriptsV2 (0.00s)
PASS
ok      xnetperf/stream 0.021s
```

### 集成测试

修复后的脚本生成输出：

```
📊 Generating latency scripts in INCAST mode (client → server only)
Total latency ports needed: 8 (from 20000 to 20007)
Server server-a IP: 192.168.1.10
Server server-b IP: 192.168.1.11
✅ Generated incast latency scripts for client-1:mlx5_0 (ports 20000-20003)
   Server script preview: ssh server-a 'ib_write_lat -d mlx5_0 -D 5 -p 20000 ...'
   Client script preview: ssh client-1 'ib_write_lat -d mlx5_0 -D 5 -p 20000 192.168.1.10 ...'
✅ Generated incast latency scripts for client-2:mlx5_0 (ports 20004-20007)
   Server script preview: ssh server-a 'ib_write_lat -d mlx5_0 -D 5 -p 20004 ...'
   Client script preview: ssh client-2 'ib_write_lat -d mlx5_0 -D 5 -p 20004 192.168.1.10 ...'
✅ Successfully generated incast latency test scripts in generated_scripts_latency
```

## 与 Fullmesh 模式的一致性

修复后，incast 和 fullmesh 模式保持一致的脚本组织逻辑：

| 维度 | Fullmesh | Incast |
|------|---------|--------|
| **遍历对象** | 所有主机 | Client 主机 |
| **脚本命名** | `{host}_{hca}_*_latency.sh` | `{client}_{hca}_*_latency.sh` |
| **文件数量** | N_hosts × N_HCAs × 2 | N_clients × N_client_HCAs × 2 |
| **脚本归属** | 每个主机拥有自己的脚本 | 每个 client 拥有自己的脚本 |
| **执行位置** | 脚本在对应主机上执行 | 脚本在对应 client 上执行 |

## 相关文件变更

- **修改文件**: `stream/stream_latency.go`
  - `generateLatencyScriptsIncast()` - 改为遍历 client 主机
  - `generateLatencyScriptsForClientIncast()` - 新函数，替代原有的 `ForServerIncast`
  - `generateLatencyScriptForClientHCAIncast()` - 新函数，替代原有的 `ForServerHCAIncast`

- **测试状态**: 所有测试通过
- **编译状态**: ✅ 成功

## 总结

这次修复解决了 incast 模式脚本生成的根本性设计缺陷：

1. **问题本质**：错误地以 server 为中心组织脚本，导致 client 主机找不到自己的脚本文件
2. **修复思路**：改为以 client 为中心，每个 client 生成一对脚本，包含到所有 servers 的测试命令
3. **效果**：脚本生成逻辑清晰，文件命名正确，执行时能找到对应的脚本
4. **一致性**：与 fullmesh 模式保持相同的脚本组织模式

修复后的实现完全符合 incast 模式的语义：**Client 发起测试，连接到 Servers**。
