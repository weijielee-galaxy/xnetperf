# 测试配置验证 API

## 创建一个有效的配置文件进行测试

```bash
# 先创建一个测试配置
curl -X POST http://localhost:8080/api/configs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "valid-test.yaml",
    "config": {
      "stream_type": "p2p",
      "server": {
        "hostname": ["server-001"],
        "hca": ["mlx5_0"]
      },
      "client": {
        "hostname": ["client-001"],
        "hca": ["mlx5_0"]
      }
    }
  }'

# 验证配置文件
curl -X POST http://localhost:8080/api/configs/valid-test.yaml/validate | jq
```

## 手动创建一个无效的配置文件进行测试

创建 `configs/invalid-test.yaml`:

```yaml
start_port: 70000  # 无效：超出端口范围
stream_type: "invalid"  # 无效：不是 fullmesh/incast/p2p
qp_num: -1  # 无效：负数
message_size_bytes: 0  # 无效：必须大于 0
speed: -100  # 无效：负数
waiting_time_seconds: -5  # 无效：负数
rdma_cm: false

report:
  enable: true
  dir: "/root"

run:
  infinitely: false
  duration_seconds: -10  # 无效：当 infinitely=false 时必须大于 0

server:
  hostname: []  # 无效：不能为空
  hca: []  # 无效：不能为空

client:
  hostname: []  # 无效：不能为空
  hca: []  # 无效：不能为空
```

然后验证：
```bash
curl -X POST http://localhost:8080/api/configs/invalid-test.yaml/validate | jq
```

预期会返回多个验证错误。

## 验证规则列表

| 字段 | 验证规则 |
|------|---------|
| server.hostname | 不能为空数组 |
| server.hca | 不能为空数组 |
| client.hostname | 不能为空数组 |
| client.hca | 不能为空数组 |
| stream_type | 必须是 fullmesh, incast 或 p2p |
| start_port | 必须在 1-65535 之间 |
| qp_num | 必须大于 0 |
| message_size_bytes | 必须大于 0 |
| speed | 必须大于 0 |
| waiting_time_seconds | 不能为负数 |
| run.duration_seconds | 当 run.infinitely=false 时必须大于 0 |

## 使用场景

1. **配置文件创建后验证**：确保配置文件格式正确
2. **配置文件使用前检查**：在运行测试前验证配置
3. **配置文件迁移**：验证从其他环境迁移的配置文件
4. **故障排查**：快速定位配置文件中的问题
