# Generate Command - 脚本生成命令

## 功能概述

`xnetperf generate` 命令用于根据配置文件生成网络测试脚本，但不执行它们。这个命令的主要目的是让用户能够预先查看和验证生成的脚本内容，确保配置正确无误后再实际运行测试。

## 使用场景

### 1. 预览脚本内容
在运行测试前，先生成脚本查看具体会执行哪些命令：
```bash
xnetperf generate
```

### 2. 验证配置正确性
修改配置文件后，快速生成脚本验证配置是否符合预期：
```bash
xnetperf generate --config custom-config.yaml
```

### 3. 调试脚本问题
当测试结果不符合预期时，生成脚本手动检查命令参数：
```bash
xnetperf generate
# 然后查看生成的脚本文件
cat generated_scripts_p2p/*.sh
```

## 命令说明

### 基本用法
```bash
xnetperf generate [flags]
```

### 参数
- `-c, --config string`: 指定配置文件路径（默认: `./config.yaml`）
- `-h, --help`: 显示帮助信息

### 示例

#### 使用默认配置生成脚本
```bash
xnetperf generate
```

输出示例：
```
📝 Generating scripts for stream type: p2p
📁 Output directory: ./generated_scripts_p2p

Total Ports Needed: 8
[server1 client1]
Output from server1 : 192.168.1.10
Output from client1 : 192.168.1.20

✅ P2P scripts generated successfully in: ./generated_scripts_p2p

📋 Generated script files:

  Server scripts:
    - server1_mlx5_0_server_p2p.sh
    - server1_mlx5_1_server_p2p.sh

  Client scripts:
    - client1_mlx5_0_client_p2p.sh
    - client1_mlx5_1_client_p2p.sh

💡 Tip: You can review the generated scripts in ./generated_scripts_p2p before running them.
```

#### 使用自定义配置文件
```bash
xnetperf generate --config /path/to/custom-config.yaml
```

#### 生成不同类型的脚本
根据配置文件中的 `stream_type` 字段，生成不同类型的测试脚本：

**Full Mesh 模式**
```yaml
stream_type: fullmesh
```
```bash
xnetperf generate
# 生成目录: ./generated_scripts_fullmesh
```

**InCast 模式**
```yaml
stream_type: incast
```
```bash
xnetperf generate
# 生成目录: ./generated_scripts_incast
```

**P2P 模式**
```yaml
stream_type: p2p
```
```bash
xnetperf generate
# 生成目录: ./generated_scripts_p2p
```

## 输出说明

### 生成的脚本位置
脚本会被保存到配置文件中指定的输出目录：
- 目录格式: `<output_base>_<stream_type>`
- 默认: `./generated_scripts_<stream_type>`

### 脚本命名规则
生成的脚本文件按照以下格式命名：
- 服务器脚本: `<hostname>_<hca>_server_<stream_type>.sh`
- 客户端脚本: `<hostname>_<hca>_client_<stream_type>.sh`

例如：
```
cetus-g88-061_mlx5_0_server_p2p.sh
cetus-g88-061_mlx5_0_client_p2p.sh
cetus-g88-061_mlx5_1_server_p2p.sh
cetus-g88-061_mlx5_1_client_p2p.sh
```

### 脚本内容示例
生成的脚本包含完整的 `ib_write_bw` 命令，例如：

**服务器脚本**:
```bash
#!/bin/bash
ssh -i ~/.ssh/id_rsa server1 'ib_write_bw -d mlx5_0 -D 10 -p 20000 --report_gbits >/dev/null 2>&1 &'; sleep 0.06
ssh -i ~/.ssh/id_rsa server1 'ib_write_bw -d mlx5_0 -D 10 -p 20001 --report_gbits >/dev/null 2>&1 &'; sleep 0.06
```

**客户端脚本**:
```bash
#!/bin/bash
ssh -i ~/.ssh/id_rsa client1 'ib_write_bw -d mlx5_0 -D 10 -m 4096 -p 20000 192.168.1.10 --report_gbits --out_json --out_json_file report_c_client1.json >/dev/null 2>&1 &'; sleep 0.06
ssh -i ~/.ssh/id_rsa client1 'ib_write_bw -d mlx5_0 -D 10 -m 4096 -p 20001 192.168.1.10 --report_gbits --out_json --out_json_file report_c_client1.json >/dev/null 2>&1 &'; sleep 0.06
```

## 与 run 命令的区别

| 特性 | generate | run |
|------|----------|-----|
| 生成脚本 | ✅ | ✅ |
| 执行 precheck | ❌ | ✅ |
| 分发脚本到远程主机 | ❌ | ✅ |
| 执行测试 | ❌ | ✅ |
| 清理旧报告文件 | ❌ | ✅ |
| 适用场景 | 预览和验证 | 运行测试 |

## 工作流程建议

推荐的测试工作流程：

1. **编辑配置文件**
   ```bash
   vim config.yaml
   ```

2. **生成并查看脚本**
   ```bash
   xnetperf generate
   ls -lh generated_scripts_p2p/
   cat generated_scripts_p2p/server1_mlx5_0_server_p2p.sh
   ```

3. **确认脚本无误后运行测试**
   ```bash
   xnetperf run
   ```

## 实现细节

### 代码位置
- 命令实现: `cmd/generate.go`
- 测试文件: `cmd/generate_test.go`

### 核心函数
```go
func execGenerateCommand(cfg *config.Config) {
    // 根据 stream_type 调用相应的生成函数
    switch cfg.StreamType {
    case config.FullMesh:
        stream.GenerateFullMeshScript(cfg)
    case config.InCast:
        stream.GenerateIncastScripts(cfg)
    case config.P2P:
        err := stream.GenerateP2PScripts(cfg)
        // ...
    }
    
    // 显示生成的脚本列表
    displayGeneratedScripts(cfg)
}
```

### 脚本列表显示
`displayGeneratedScripts()` 函数会：
1. 扫描输出目录
2. 分类显示服务器脚本和客户端脚本
3. 提供查看提示

## 错误处理

### 配置文件错误
```bash
$ xnetperf generate --config invalid.yaml
Error reading config: failed to read config file 'invalid.yaml': open invalid.yaml: no such file or directory
```

### 无效的 stream_type
```bash
$ xnetperf generate
❌ Invalid stream_type 'invalid' in config. Supported types: fullmesh, incast, p2p
```

### SSH 连接问题
如果配置的主机无法 SSH 连接，会显示详细错误信息：
```bash
Error executing command on server1: exit status 255
Output: ssh: connect to host server1 port 22: Connection refused
```

## 测试

运行 generate 命令的测试：
```bash
go test ./cmd -run TestGenerate -v
go test ./cmd -run "TestContains|TestIndexOf" -v
```

所有测试通过 ✅

## 提示和最佳实践

1. **先生成，后运行**: 始终先使用 `generate` 命令查看生成的脚本，确认无误后再用 `run` 命令执行
2. **版本控制**: 可以将生成的脚本提交到 Git，方便追踪配置变化
3. **手动测试**: 对于复杂配置，可以先手动执行生成的脚本中的某一条命令，验证连通性
4. **批量验证**: 使用 `grep` 或 `awk` 检查生成的脚本中的参数是否符合预期

```bash
# 检查所有脚本中的端口号
grep -h "ib_write_bw" generated_scripts_p2p/*.sh | grep -oP "\-p \d+" | sort -u

# 检查所有脚本中的设备名
grep -h "ib_write_bw" generated_scripts_p2p/*.sh | grep -oP "\-d \w+" | sort -u

# 统计生成的命令数量
grep -c "ib_write_bw" generated_scripts_p2p/*.sh
```

## 未来增强

可能的功能增强方向：
- [ ] 添加 `--dry-run` 模式，不实际创建文件，只显示将要生成的脚本信息
- [ ] 添加 `--output` 参数，允许指定自定义输出目录
- [ ] 支持生成脚本的同时进行语法检查
- [ ] 添加脚本模板自定义功能
