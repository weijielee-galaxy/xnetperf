# xnetperf HTTP Server API 参考文档

> **版本**: v0.2.0  
> **更新日期**: 2025-11-05  
> **基础路径**: `http://localhost:8080`

## 📖 目录

- [概述](#概述)
- [通用说明](#通用说明)
- [配置文件管理 API](#配置文件管理-api)
- [字典管理 API](#字典管理-api)
- [健康检查 API](#健康检查-api)
- [数据结构](#数据结构)
- [错误码](#错误码)
- [使用示例](#使用示例)

---

## 概述

xnetperf HTTP Server 提供了一套 RESTful API，用于管理配置文件、执行网络性能测试和收集测试报告。所有 API 端点都遵循统一的响应格式。

### 启动服务器

```bash
# 使用默认端口 8080
./xnetperf server

# 指定端口
./xnetperf server --port 8080
```

服务器启动后，可以通过以下地址访问：
- **Web UI**: http://localhost:8080
- **API 端点**: http://localhost:8080/api

---

## 通用说明

### 统一响应格式

所有 API 都返回以下格式的 JSON 响应：

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

**字段说明**：
- `code` (int): 状态码，0 表示成功，非 0 表示错误
- `message` (string): 响应消息，成功时为 "success"，失败时为错误描述
- `data` (any): 响应数据，根据具体接口返回不同的数据结构

### 错误响应格式

```json
{
  "code": 400,
  "message": "配置文件名不能为空",
  "data": null
}
```

### Content-Type

- 请求头：`Content-Type: application/json`
- 响应头：`Content-Type: application/json; charset=utf-8`

---

## 配置文件管理 API

### 1. 获取配置文件列表

获取所有可用的配置文件列表。

**接口**：`GET /api/configs`

**请求参数**：无

**响应示例**：

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
      "name": "test1.yaml",
      "path": "configs/test1.yaml",
      "is_default": false,
      "is_deletable": true
    }
  ]
}
```

**响应字段**：
- `name` (string): 配置文件名称
- `path` (string): 配置文件路径
- `is_default` (bool): 是否为默认配置文件（config.yaml）
- `is_deletable` (bool): 是否可删除（默认配置不可删除）

---

### 2. 获取指定配置文件

获取指定配置文件的完整内容。

**接口**：`GET /api/configs/:name`

**路径参数**：
- `name` (string, required): 配置文件名称，如 `config.yaml`

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "start_port": 20000,
    "stream_type": "incast",
    "qp_num": 10,
    "message_size_bytes": 4096,
    "output_base": "./generated_scripts",
    "waiting_time_seconds": 5,
    "speed": 400,
    "rdma_cm": false,
    "gid_index": 3,
    "network_interface": "bond0",
    "report": {
      "enable": true,
      "dir": "/root"
    },
    "run": {
      "infinitely": false,
      "duration_seconds": 20
    },
    "ssh": {
      "user": "root",
      "private_key": "~/.ssh/id_rsa"
    },
    "logger": {
      "log_level": "info",
      "log_format": "text"
    },
    "server": {
      "hostname": ["server1", "server2"],
      "hca": ["mlx5_0", "mlx5_1"]
    },
    "client": {
      "hostname": ["client1", "client2"],
      "hca": ["mlx5_0", "mlx5_1"]
    },
    "version": "v1"
  }
}
```

**配置字段详解**：

| 字段 | 类型 | 说明 | 默认值 |
|------|------|------|--------|
| `start_port` | int | 起始端口号（1-65535） | 20000 |
| `stream_type` | string | 流类型：`fullmesh`、`incast`、`p2p` | incast |
| `qp_num` | int | Queue Pair 数量 | 10 |
| `message_size_bytes` | int | 消息大小（字节） | 4096 |
| `output_base` | string | 脚本输出目录 | ./generated_scripts |
| `waiting_time_seconds` | int | 客户端启动前等待时间（秒） | 5 |
| `speed` | float | 理论带宽速度（Gbps） | 400 |
| `rdma_cm` | bool | 是否使用 RDMA CM | false |
| `gid_index` | int | GID 索引（RoCE v2） | 3 |
| `network_interface` | string | 网络接口名称 | bond0 |
| `report.enable` | bool | 是否启用报告收集 | true |
| `report.dir` | string | 报告保存目录 | /root |
| `run.infinitely` | bool | 是否无限运行 | false |
| `run.duration_seconds` | int | 运行时长（秒） | 20 |
| `ssh.user` | string | SSH 用户名 | root |
| `ssh.private_key` | string | SSH 私钥路径 | ~/.ssh/id_rsa |
| `logger.log_level` | string | 日志级别：`debug`、`info`、`warn`、`error` | info |
| `logger.log_format` | string | 日志格式：`text`、`json` | text |
| `server.hostname` | []string | 服务端主机名列表 | [] |
| `server.hca` | []string | 服务端 HCA 设备列表 | [] |
| `client.hostname` | []string | 客户端主机名列表 | [] |
| `client.hca` | []string | 客户端 HCA 设备列表 | [] |
| `version` | string | 配置文件版本 | v1 |

---

### 3. 预览配置文件（YAML 格式）

以 YAML 格式预览配置文件内容。

**接口**：`GET /api/configs/:name/preview`

**路径参数**：
- `name` (string, required): 配置文件名称

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "yaml": "start_port: 20000\nstream_type: incast\n..."
  }
}
```

**响应字段**：
- `yaml` (string): YAML 格式的配置文件内容

---

### 4. 创建配置文件

创建新的配置文件。

**接口**：`POST /api/configs`

**请求体**：

```json
{
  "name": "test1.yaml",
  "config": {
    "start_port": 20000,
    "stream_type": "incast",
    "server": {
      "hostname": ["server1"],
      "hca": ["mlx5_0"]
    },
    "client": {
      "hostname": ["client1"],
      "hca": ["mlx5_0"]
    }
  }
}
```

**请求字段**：
- `name` (string, required): 配置文件名称，必须以 `.yaml` 或 `.yml` 结尾
- `config` (object, required): 配置对象，参考"获取指定配置文件"中的字段说明

**响应示例**：

```json
{
  "code": 0,
  "message": "配置文件创建成功",
  "data": {
    "name": "test1.yaml",
    "path": "configs/test1.yaml",
    "is_default": false,
    "is_deletable": true
  }
}
```

**注意事项**：
- 不能创建名为 `config.yaml` 的文件（保留给默认配置）
- 未指定的字段会自动使用默认值
- 文件会保存在 `configs/` 目录下

---

### 5. 更新配置文件

更新已存在的配置文件。

**接口**：`PUT /api/configs/:name`

**路径参数**：
- `name` (string, required): 配置文件名称

**请求体**：

```json
{
  "start_port": 20000,
  "stream_type": "fullmesh",
  "speed": 400,
  "server": {
    "hostname": ["server1", "server2"],
    "hca": ["mlx5_0"]
  },
  "client": {
    "hostname": ["client1", "client2"],
    "hca": ["mlx5_0"]
  }
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "配置文件更新成功",
  "data": null
}
```

---

### 6. 删除配置文件

删除指定的配置文件。

**接口**：`DELETE /api/configs/:name`

**路径参数**：
- `name` (string, required): 配置文件名称

**响应示例**：

```json
{
  "code": 0,
  "message": "配置文件删除成功",
  "data": null
}
```

**注意事项**：
- 不能删除默认配置文件 `config.yaml`

---

### 7. 验证配置文件

验证配置文件是否有效。

**接口**：`POST /api/configs/:name/validate`

**路径参数**：
- `name` (string, required): 配置文件名称

**响应示例（成功）**：

```json
{
  "code": 0,
  "message": "配置文件验证成功",
  "data": {
    "valid": true,
    "config": { ... }
  }
}
```

**响应示例（失败）**：

```json
{
  "code": 400,
  "message": "配置文件验证失败",
  "data": {
    "valid": false,
    "errors": [
      "server.hostname 不能为空",
      "client.hca 不能为空",
      "start_port 必须在 1-65535 之间，当前值: 0"
    ]
  }
}
```

**验证规则**：
- `server.hostname` 和 `server.hca` 不能为空
- `client.hostname` 和 `client.hca` 不能为空
- `stream_type` 必须是 `fullmesh`、`incast` 或 `p2p`
- `start_port` 必须在 1-65535 之间
- `qp_num` 必须大于 0
- `message_size_bytes` 必须大于 0
- `speed` 必须大于 0
- `waiting_time_seconds` 不能为负数
- 当 `run.infinitely` 为 false 时，`run.duration_seconds` 必须大于 0

---

### 8. 执行 Precheck 检查

在指定的配置上执行网络预检查。

**接口**：`POST /api/configs/:name/precheck`

**路径参数**：
- `name` (string, required): 配置文件名称

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "summary": {
      "total_hosts": 4,
      "success_hosts": 4,
      "failed_hosts": 0,
      "total_hcas": 8,
      "healthy_hcas": 8,
      "unhealthy_hcas": 0
    },
    "details": [
      {
        "hostname": "server1",
        "serial_number": "MT2116X09876",
        "hcas": [
          {
            "name": "mlx5_0",
            "link_layer": "InfiniBand",
            "state": "ACTIVE",
            "phys_state": "LinkUp",
            "rate": "200 Gb/sec (4X HDR)",
            "is_healthy": true,
            "error": ""
          }
        ],
        "is_healthy": true,
        "error": ""
      }
    ]
  }
}
```

**响应字段详解**：

**summary（汇总信息）**：
- `total_hosts` (int): 总主机数
- `success_hosts` (int): 检查成功的主机数
- `failed_hosts` (int): 检查失败的主机数
- `total_hcas` (int): 总 HCA 设备数
- `healthy_hcas` (int): 健康的 HCA 数量
- `unhealthy_hcas` (int): 不健康的 HCA 数量

**details（详细信息）**：
- `hostname` (string): 主机名
- `serial_number` (string): 序列号
- `hcas` (array): HCA 设备列表
  - `name` (string): HCA 设备名称
  - `link_layer` (string): 链路层类型（InfiniBand/Ethernet）
  - `state` (string): 端口状态（ACTIVE/DOWN）
  - `phys_state` (string): 物理状态（LinkUp/LinkDown）
  - `rate` (string): 传输速率
  - `is_healthy` (bool): 是否健康（state=ACTIVE 且 phys_state=LinkUp）
  - `error` (string): 错误信息（如果有）
- `is_healthy` (bool): 主机是否健康（所有 HCA 都健康）
- `error` (string): 错误信息（如果有）

---

### 9. 运行测试

使用指定配置运行性能测试。

**接口**：`POST /api/configs/:name/run`

**路径参数**：
- `name` (string, required): 配置文件名称

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "status": "running",
    "start_time": "2025-11-05T10:30:00Z",
    "message": "测试已启动"
  }
}
```

**响应字段**：
- `status` (string): 测试状态（`running`、`completed`、`failed`）
- `start_time` (string): 开始时间（ISO 8601 格式）
- `message` (string): 状态消息

---

### 10. 探测测试状态

探测当前测试的运行状态。

**接口**：`POST /api/configs/:name/probe`

**路径参数**：
- `name` (string, required): 配置文件名称

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "summary": {
      "total_hosts": 4,
      "running_hosts": 2,
      "completed_hosts": 2,
      "failed_hosts": 0
    },
    "details": [
      {
        "hostname": "server1",
        "device": "mlx5_0",
        "status": "running",
        "pid": 12345,
        "process_count": 10
      },
      {
        "hostname": "client1",
        "device": "mlx5_0",
        "status": "completed",
        "pid": 0,
        "process_count": 0
      }
    ]
  }
}
```

**响应字段详解**：

**summary（汇总信息）**：
- `total_hosts` (int): 总主机数
- `running_hosts` (int): 正在运行的主机数
- `completed_hosts` (int): 已完成的主机数
- `failed_hosts` (int): 失败的主机数

**details（详细信息）**：
- `hostname` (string): 主机名
- `device` (string): HCA 设备名称
- `status` (string): 状态（`running`、`completed`、`not_running`）
- `pid` (int): 进程 ID（0 表示未运行）
- `process_count` (int): 进程数量

---

### 11. 收集测试报告

从远程主机收集测试报告文件。

**接口**：`POST /api/configs/:name/collect`

**路径参数**：
- `name` (string, required): 配置文件名称

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "summary": {
      "total_files": 20,
      "collected_files": 20,
      "failed_files": 0
    },
    "files": [
      {
        "hostname": "server1",
        "device": "mlx5_0",
        "filename": "server1_mlx5_0_report.txt",
        "size": 2048,
        "collected": true
      }
    ]
  }
}
```

**响应字段详解**：

**summary（汇总信息）**：
- `total_files` (int): 总文件数
- `collected_files` (int): 成功收集的文件数
- `failed_files` (int): 收集失败的文件数

**files（文件列表）**：
- `hostname` (string): 主机名
- `device` (string): HCA 设备名称
- `filename` (string): 文件名
- `size` (int): 文件大小（字节）
- `collected` (bool): 是否成功收集

---

### 12. 获取性能报告

生成并获取性能分析报告。

**接口**：`GET /api/configs/:name/report`

**路径参数**：
- `name` (string, required): 配置文件名称

**响应示例（InCast/FullMesh）**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "stream_type": "incast",
    "theoretical_bw_per_client": 100,
    "total_server_bw": 400,
    "client_count": 4,
    "client_data": {
      "client1": {
        "mlx5_0": {
          "hostname": "client1",
          "device": "mlx5_0",
          "actual_bw": 95.5,
          "theoretical_bw": 100,
          "delta": -4.5,
          "delta_percent": -4.5,
          "status": "OK"
        }
      }
    },
    "server_data": {
      "server1": {
        "mlx5_0": {
          "hostname": "server1",
          "device": "mlx5_0",
          "rx_bw": 391.19,
          "theoretical_bw": 400,
          "delta": -8.81,
          "delta_percent": -2.2,
          "status": "OK"
        }
      }
    }
  }
}
```

**响应示例（P2P）**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "stream_type": "p2p",
    "p2p_data": {
      "host1": {
        "mlx5_0": {
          "hostname": "host1",
          "device": "mlx5_0",
          "avg_speed": 195.5,
          "count": 10
        }
      }
    },
    "p2p_summary": {
      "total_pairs": 20,
      "avg_speed": 198.3
    }
  }
}
```

**响应字段详解**：

**通用字段**：
- `stream_type` (string): 流类型（`incast`、`fullmesh`、`p2p`）

**InCast/FullMesh 模式**：
- `theoretical_bw_per_client` (float): 单客户端理论带宽（Gbps）
- `total_server_bw` (float): 服务端总带宽（Gbps）
- `client_count` (int): 客户端数量

**client_data（客户端数据）**：
- `hostname` (string): 主机名
- `device` (string): HCA 设备名称
- `actual_bw` (float): 实际发送带宽（Gbps）
- `theoretical_bw` (float): 理论带宽（Gbps）
- `delta` (float): 差值 = actual_bw - theoretical_bw
- `delta_percent` (float): 差值百分比 = (delta / theoretical_bw) × 100
- `status` (string): 状态（`OK` 或 `NOT OK`，|delta_percent| > 20% 时为 NOT OK）

**server_data（服务端数据）**：
- `hostname` (string): 主机名
- `device` (string): HCA 设备名称
- `rx_bw` (float): 实际接收带宽（Gbps）
- `theoretical_bw` (float): 理论带宽（Gbps，即配置的 speed）
- `delta` (float): 差值 = rx_bw - theoretical_bw
- `delta_percent` (float): 差值百分比 = (delta / theoretical_bw) × 100
- `status` (string): 状态（`OK` 或 `NOT OK`）

**P2P 模式**：

**p2p_data（P2P 数据）**：
- `hostname` (string): 主机名
- `device` (string): HCA 设备名称
- `avg_speed` (float): 平均速度（Gbps）
- `count` (int): 连接对数

**p2p_summary（P2P 汇总）**：
- `total_pairs` (int): 总连接对数
- `avg_speed` (float): 平均速度（Gbps）

---

## 字典管理 API

字典管理用于维护主机名和 HCA 设备的预定义列表，方便在 Web UI 中快速选择。

### 1. 获取主机名列表

获取预定义的主机名列表。

**接口**：`GET /api/dictionary/hostnames`

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": [
    "server1",
    "server2",
    "client1",
    "client2"
  ]
}
```

---

### 2. 更新主机名列表

更新预定义的主机名列表。

**接口**：`PUT /api/dictionary/hostnames`

**请求体**：

```json
{
  "hostnames": [
    "server1",
    "server2",
    "client1",
    "client2",
    "node1",
    "node2"
  ]
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "主机名列表更新成功",
  "data": [
    "server1",
    "server2",
    "client1",
    "client2",
    "node1",
    "node2"
  ]
}
```

**注意事项**：
- 自动去重
- 自动去除空值
- 保存到 `dictionary/hostnames.txt` 文件

---

### 3. 获取 HCA 列表

获取预定义的 HCA 设备列表。

**接口**：`GET /api/dictionary/hcas`

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": [
    "mlx5_0",
    "mlx5_1",
    "mlx5_2",
    "mlx5_3"
  ]
}
```

---

### 4. 更新 HCA 列表

更新预定义的 HCA 设备列表。

**接口**：`PUT /api/dictionary/hcas`

**请求体**：

```json
{
  "hcas": [
    "mlx5_0",
    "mlx5_1",
    "mlx5_2",
    "mlx5_3",
    "mlx5_4"
  ]
}
```

**响应示例**：

```json
{
  "code": 0,
  "message": "HCA 列表更新成功",
  "data": [
    "mlx5_0",
    "mlx5_1",
    "mlx5_2",
    "mlx5_3",
    "mlx5_4"
  ]
}
```

**注意事项**：
- 自动去重
- 自动去除空值
- 保存到 `dictionary/hcas.txt` 文件

---

## 健康检查 API

### 健康检查

检查 HTTP Server 是否正常运行。

**接口**：`GET /health`

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "status": "ok"
  }
}
```

---

## 数据结构

### Config（配置对象）

完整的配置对象结构，用于创建和更新配置文件。

```typescript
interface Config {
  start_port: number              // 起始端口号
  stream_type: string             // 流类型：fullmesh|incast|p2p
  qp_num: number                  // Queue Pair 数量
  message_size_bytes: number      // 消息大小（字节）
  output_base: string             // 输出目录
  waiting_time_seconds: number    // 等待时间（秒）
  speed: number                   // 理论带宽（Gbps）
  rdma_cm: boolean                // 是否使用 RDMA CM
  gid_index: number               // GID 索引
  network_interface: string       // 网络接口名称
  report: {
    enable: boolean               // 启用报告
    dir: string                   // 报告目录
  }
  run: {
    infinitely: boolean           // 无限运行
    duration_seconds: number      // 运行时长（秒）
  }
  ssh: {
    user: string                  // SSH 用户名
    private_key: string           // SSH 私钥路径
  }
  logger: {
    log_level: string             // 日志级别：debug|info|warn|error
    log_format: string            // 日志格式：text|json
  }
  server: {
    hostname: string[]            // 服务端主机名列表
    hca: string[]                 // 服务端 HCA 列表
  }
  client: {
    hostname: string[]            // 客户端主机名列表
    hca: string[]                 // 客户端 HCA 列表
  }
  version: string                 // 配置版本
}
```

---

## 错误码

| 错误码 | 说明 | 示例 |
|--------|------|------|
| 0 | 成功 | 请求成功处理 |
| 400 | 请求参数错误 | 配置文件名不能为空 |
| 404 | 资源不存在 | 配置文件不存在 |
| 500 | 服务器内部错误 | 配置文件读取失败 |

---

## 使用示例

### 示例 1：创建并运行测试

```bash
# 1. 创建配置文件
curl -X POST http://localhost:8080/api/configs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test1.yaml",
    "config": {
      "stream_type": "incast",
      "speed": 400,
      "server": {
        "hostname": ["server1"],
        "hca": ["mlx5_0"]
      },
      "client": {
        "hostname": ["client1", "client2"],
        "hca": ["mlx5_0"]
      }
    }
  }'

# 2. 验证配置
curl -X POST http://localhost:8080/api/configs/test1.yaml/validate

# 3. 执行 Precheck
curl -X POST http://localhost:8080/api/configs/test1.yaml/precheck

# 4. 运行测试
curl -X POST http://localhost:8080/api/configs/test1.yaml/run

# 5. 探测状态
curl -X POST http://localhost:8080/api/configs/test1.yaml/probe

# 6. 收集报告
curl -X POST http://localhost:8080/api/configs/test1.yaml/collect

# 7. 获取性能报告
curl http://localhost:8080/api/configs/test1.yaml/report
```

### 示例 2：管理字典

```bash
# 获取主机名列表
curl http://localhost:8080/api/dictionary/hostnames

# 更新主机名列表
curl -X PUT http://localhost:8080/api/dictionary/hostnames \
  -H "Content-Type: application/json" \
  -d '{
    "hostnames": ["server1", "server2", "client1"]
  }'

# 获取 HCA 列表
curl http://localhost:8080/api/dictionary/hcas

# 更新 HCA 列表
curl -X PUT http://localhost:8080/api/dictionary/hcas \
  -H "Content-Type: application/json" \
  -d '{
    "hcas": ["mlx5_0", "mlx5_1", "mlx5_2"]
  }'
```

### 示例 3：使用 JavaScript（Fetch API）

```javascript
// 获取配置文件列表
async function getConfigs() {
  const response = await fetch('http://localhost:8080/api/configs')
  const result = await response.json()
  console.log(result.data)
}

// 创建配置文件
async function createConfig() {
  const response = await fetch('http://localhost:8080/api/configs', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      name: 'test1.yaml',
      config: {
        stream_type: 'incast',
        speed: 400,
        server: {
          hostname: ['server1'],
          hca: ['mlx5_0']
        },
        client: {
          hostname: ['client1'],
          hca: ['mlx5_0']
        }
      }
    })
  })
  const result = await response.json()
  console.log(result)
}

// 获取性能报告
async function getReport(configName) {
  const response = await fetch(`http://localhost:8080/api/configs/${configName}/report`)
  const result = await response.json()
  console.log(result.data)
}
```

---

## 相关文档

- [用户指南](traffic-test-guide.md) - 流量测试完整指南
- [Web UI 快速开始](web-ui-quickstart.md) - Web 界面使用入门
- [配置验证功能](config-validation-feature.md) - 配置文件验证说明

---

## 版本历史

| 版本 | 日期 | 变更说明 |
|------|------|----------|
| v0.2.0 | 2025-11-05 | 添加 Logger 配置支持 |
| v0.1.2 | 2024-12-15 | 添加 SSH 私钥配置 |
| v0.1.1 | 2024-12-01 | 初始版本，基础 API 实现 |

---

**支持与反馈**：如有问题或建议，请提交 Issue 到 GitHub 仓库。
