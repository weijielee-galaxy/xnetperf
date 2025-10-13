# HTTP Server API Documentation

## 启动服务器

```bash
# 使用默认端口 8080
./xnetperf server

# 指定端口
./xnetperf server --port 9000
```

## API 端点

### 1. 健康检查

```
GET /health
```

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "status": "ok"
  }
}
```

### 2. 获取配置文件列表

```
GET /api/configs
```

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "name": "config.yaml",
      "path": "config.yaml",
      "is_default": true,
      "is_deletable": false
    },
    {
      "name": "test-config.yaml",
      "path": "configs/test-config.yaml",
      "is_default": false,
      "is_deletable": true
    }
  ]
}
```

### 3. 查看指定配置文件

```
GET /api/configs/:name
```

**示例：**
```bash
GET /api/configs/config.yaml
GET /api/configs/test-config.yaml
```

**响应示例：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "server": {
      "hostname": ["server-001", "server-002"],
      "hca": ["mlx5_0", "mlx5_1"]
    },
    "client": {
      "hostname": ["client-001"],
      "hca": ["mlx5_0"]
    },
    "config": {
      "duration": 10,
      "msg_size": 8388608
    }
  }
}
```

### 4. 创建配置文件

```
POST /api/configs
Content-Type: application/json
```

**请求体：**
```json
{
  "name": "test-config.yaml",
  "config": {
    "server": {
      "hostname": ["server-001", "server-002"],
      "hca": ["mlx5_0", "mlx5_1"]
    },
    "client": {
      "hostname": ["client-001", "client-002"],
      "hca": ["mlx5_0", "mlx5_1"]
    },
    "config": {
      "duration": 10,
      "msg_size": 8388608
    }
  }
}
```

**响应示例：**
```json
{
  "code": 0,
  "message": "配置文件创建成功",
  "data": {
    "name": "test-config.yaml",
    "path": "configs/test-config.yaml",
    "is_default": false,
    "is_deletable": true
  }
}
```

**注意事项：**
- 文件名必须以 `.yaml` 或 `.yml` 结尾
- 不能创建名为 `config.yaml` 的文件
- 如果文件已存在会返回错误

### 5. 更新配置文件

```
PUT /api/configs/:name
Content-Type: application/json
```

**请求体：**
```json
{
  "server": {
    "hostname": ["server-001", "server-002", "server-003"],
    "hca": ["mlx5_0", "mlx5_1", "mlx5_2"]
  },
  "client": {
    "hostname": ["client-001", "client-002"],
    "hca": ["mlx5_0", "mlx5_1"]
  },
  "config": {
    "duration": 20,
    "msg_size": 8388608
  }
}
```

**响应示例：**
```json
{
  "code": 0,
  "message": "配置文件更新成功",
  "data": null
}
```

**注意事项：**
- 可以更新默认配置文件 `config.yaml`
- 文件必须已存在才能更新

### 6. 删除配置文件

```
DELETE /api/configs/:name
```

**示例：**
```bash
DELETE /api/configs/test-config.yaml
```

**响应示例：**
```json
{
  "code": 0,
  "message": "配置文件删除成功",
  "data": null
}
```

**注意事项：**
- 不能删除默认配置文件 `config.yaml`
- 文件必须存在才能删除

### 7. 验证配置文件

```
POST /api/configs/:name/validate
```

**示例：**
```bash
POST /api/configs/config.yaml/validate
POST /api/configs/test-config.yaml/validate
```

**验证成功响应示例：**
```json
{
  "code": 0,
  "message": "配置文件验证成功",
  "data": {
    "valid": true,
    "config": {
      "start_port": 20000,
      "stream_type": "p2p",
      "qp_num": 10,
      "message_size_bytes": 4096,
      "output_base": "./generated_scripts",
      "waiting_time_seconds": 15,
      "speed": 400,
      "rdma_cm": false,
      "report": {
        "enable": true,
        "dir": "/root"
      },
      "run": {
        "infinitely": true,
        "duration_seconds": 10
      },
      "server": {
        "hostname": ["server-001"],
        "hca": ["mlx5_0"]
      },
      "client": {
        "hostname": ["client-001"],
        "hca": ["mlx5_0"]
      }
    }
  }
}
```

**验证失败响应示例：**
```json
{
  "code": 400,
  "message": "配置文件验证失败",
  "data": {
    "valid": false,
    "errors": [
      "server.hostname 不能为空",
      "client.hca 不能为空",
      "start_port 必须在 1-65535 之间，当前值: 70000"
    ]
  }
}
```

**验证规则：**
- 必填字段：
  - `server.hostname` - 服务器主机名列表不能为空
  - `server.hca` - 服务器 HCA 列表不能为空
  - `client.hostname` - 客户端主机名列表不能为空
  - `client.hca` - 客户端 HCA 列表不能为空
- 字段值验证：
  - `stream_type` - 必须是 `fullmesh`, `incast` 或 `p2p`
  - `start_port` - 必须在 1-65535 之间
  - `qp_num` - 必须大于 0
  - `message_size_bytes` - 必须大于 0
  - `speed` - 必须大于 0
  - `waiting_time_seconds` - 不能为负数
  - `run.duration_seconds` - 当 `run.infinitely` 为 false 时必须大于 0

**注意事项：**
- 此 API 会检查配置文件是否能正常解析
- 会验证所有必填字段和字段值的合法性
- 验证成功时会返回完整的配置对象
- 可用于在使用配置文件前进行预检查

## 错误响应格式

```json
{
  "code": 400,
  "message": "错误描述信息",
  "data": null
}
```

**常见错误码：**
- `400`: 请求参数错误
- `404`: 配置文件不存在
- `500`: 服务器内部错误

## 使用 REST Client 测试

在 VS Code 中安装 REST Client 插件后，打开 `apitests/config.http` 文件，点击每个请求上方的"Send Request"按钮即可测试API。

## 目录结构

```
.
├── config.yaml          # 默认配置文件（不可删除）
└── configs/             # 自定义配置文件目录
    ├── test-config.yaml
    └── prod-config.yaml
```
