# 配置文件验证 API - 功能说明

## 概述

新增配置文件验证 API，用于测试配置文件是否能正常解析并验证其内容的合法性。

## API 端点

```
POST /api/configs/:name/validate
```

## 功能特性

### 1. 文件解析验证
- 检查 YAML 文件是否能正常解析
- 检查文件结构是否符合 Config 结构体定义

### 2. 必填字段验证
确保以下字段不为空：
- `server.hostname` - 服务器主机名列表
- `server.hca` - 服务器 HCA 设备列表
- `client.hostname` - 客户端主机名列表
- `client.hca` - 客户端 HCA 设备列表

### 3. 字段值验证
对配置值进行合法性检查：

| 字段 | 验证规则 | 错误示例 |
|------|---------|---------|
| stream_type | 必须是 fullmesh, incast 或 p2p | "invalid" |
| start_port | 1-65535 | 70000, 0, -1 |
| qp_num | > 0 | 0, -1 |
| message_size_bytes | > 0 | 0, -1 |
| speed | > 0 | 0, -100 |
| waiting_time_seconds | >= 0 | -5 |
| run.duration_seconds | 当 run.infinitely=false 时 > 0 | 0, -10 |

## 响应格式

### 验证成功

```json
{
  "code": 0,
  "message": "配置文件验证成功",
  "data": {
    "valid": true,
    "config": {
      // 完整的配置对象
    }
  }
}
```

### 验证失败（解析错误）

```json
{
  "code": 400,
  "message": "配置文件解析失败: yaml: unmarshal errors...",
  "data": null
}
```

### 验证失败（字段验证错误）

```json
{
  "code": 400,
  "message": "配置文件验证失败",
  "data": {
    "valid": false,
    "errors": [
      "server.hostname 不能为空",
      "start_port 必须在 1-65535 之间，当前值: 70000",
      "stream_type 必须是 fullmesh, incast 或 p2p，当前值: invalid"
    ]
  }
}
```

### 文件不存在

```json
{
  "code": 404,
  "message": "配置文件不存在",
  "data": null
}
```

## 使用示例

### 示例 1: 验证默认配置

```bash
curl -X POST http://localhost:8080/api/configs/config.yaml/validate
```

### 示例 2: 验证自定义配置

```bash
curl -X POST http://localhost:8080/api/configs/test-config.yaml/validate
```

### 示例 3: REST Client (config.http)

```http
### Validate config file
POST {{apiUrl}}/configs/config.yaml/validate HTTP/1.1
```

## 实现细节

### 代码位置
- `server/config_service.go` - `ValidateConfig()` 方法
- `server/server.go` - 路由配置

### 验证流程

1. **参数检查**：验证配置文件名是否提供
2. **文件存在性检查**：确认文件存在
3. **解析配置文件**：使用 `config.LoadConfig()` 加载并解析
4. **字段验证**：逐个检查必填字段和值的合法性
5. **返回结果**：
   - 全部通过：返回成功和完整配置
   - 有错误：返回错误列表

### 验证规则实现

```go
// 必填字段检查
if len(cfg.Server.Hostname) == 0 {
    validationErrors = append(validationErrors, "server.hostname 不能为空")
}

// 值范围检查
if cfg.StartPort <= 0 || cfg.StartPort > 65535 {
    validationErrors = append(validationErrors, 
        fmt.Sprintf("start_port 必须在 1-65535 之间，当前值: %d", cfg.StartPort))
}

// 枚举值检查
if cfg.StreamType != config.FullMesh && 
   cfg.StreamType != config.InCast && 
   cfg.StreamType != config.P2P {
    validationErrors = append(validationErrors, 
        fmt.Sprintf("stream_type 必须是 fullmesh, incast 或 p2p，当前值: %s", cfg.StreamType))
}
```

## 应用场景

### 1. 开发阶段
- 创建配置文件后立即验证
- 确保配置格式正确

### 2. 运行前检查
```bash
# 验证配置
curl -X POST http://localhost:8080/api/configs/my-test.yaml/validate

# 如果验证通过，再运行测试
./xnetperf run --config configs/my-test.yaml
```

### 3. CI/CD 流程
```bash
#!/bin/bash
# 在部署前验证所有配置文件

for config in configs/*.yaml; do
    name=$(basename "$config")
    echo "Validating $name..."
    
    result=$(curl -s -X POST "http://localhost:8080/api/configs/$name/validate")
    valid=$(echo "$result" | jq -r '.data.valid')
    
    if [ "$valid" != "true" ]; then
        echo "ERROR: $name validation failed!"
        echo "$result" | jq '.data.errors'
        exit 1
    fi
done

echo "All configs validated successfully!"
```

### 4. 配置迁移
- 从其他环境迁移配置文件
- 验证配置兼容性

### 5. 故障排查
- 快速定位配置问题
- 获取清晰的错误提示

## 测试覆盖

### REST Client 测试 (apitests/config.http)
```http
### Validate default config (should succeed)
POST {{apiUrl}}/configs/config.yaml/validate HTTP/1.1

### Validate custom config (should succeed)
POST {{apiUrl}}/configs/test-config.yaml/validate HTTP/1.1

### Validate non-existent config (should return 404)
POST {{apiUrl}}/configs/nonexistent.yaml/validate HTTP/1.1
```

### curl 测试脚本 (apitests/config.curls)
```bash
# Test 15: Validate default config
curl -X POST "${API_URL}/configs/config.yaml/validate" | jq

# Test 16: Validate custom config
curl -X POST "${API_URL}/configs/test-config.yaml/validate" | jq

# Test 17: Validate non-existent config
curl -X POST "${API_URL}/configs/nonexistent.yaml/validate" | jq
```

## 优势总结

1. ✅ **预防性检查**：在使用配置前发现问题
2. ✅ **清晰的错误信息**：提供具体的错误描述和当前值
3. ✅ **完整的验证规则**：覆盖所有关键字段和值范围
4. ✅ **易于集成**：简单的 REST API，支持自动化流程
5. ✅ **快速反馈**：无需实际运行测试即可验证配置

## 后续改进建议

1. **警告级别验证**：添加警告信息（非错误但可能影响性能）
2. **配置建议**：提供最佳实践建议
3. **批量验证**：一次验证多个配置文件
4. **配置对比**：对比两个配置的差异
5. **配置模板验证**：验证配置是否符合特定模板要求
