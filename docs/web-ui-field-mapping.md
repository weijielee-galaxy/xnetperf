# Web UI 字段映射问题修复

## 问题描述

Web UI 在加载配置时，除了 `stream_type` 外，其他字段都无法正确显示。

## 根本原因

### 1. JSON 字段命名不一致

Go 后端的 `Config` 结构体只定义了 `yaml` 标签，没有定义 `json` 标签：

```go
type Config struct {
    StartPort          int     `yaml:"start_port"`      // JSON 序列化后保持 "StartPort"
    StreamType         string  `yaml:"stream_type"`     // JSON 序列化后保持 "StreamType"
    QpNum              int     `yaml:"qp_num"`          // JSON 序列化后保持 "QpNum"
    MessageSizeBytes   int     `yaml:"message_size_bytes"` // JSON 序列化后保持 "MessageSizeBytes"
    // ...
}
```

**Go 的 JSON 序列化规则**：
- 如果没有 `json` 标签，Go 会使用字段名本身（PascalCase）
- **保持原样**，不做任何转换
- 例如：`StartPort` → `StartPort` (不是 startPort!)

### 2. 前端字段名错误

前端代码中使用了 snake_case 来读取数据：

```javascript
// ❌ 错误：使用 snake_case 或 camelCase
document.getElementById('start_port').value = data.start_port;  // undefined!
document.getElementById('start_port').value = data.startPort;   // undefined!

// ✅ 正确：使用 PascalCase（与 Go 字段名完全一致）
document.getElementById('start_port').value = data.StartPort;   // 20000 ✅
document.getElementById('qp_num').value = data.QpNum;           // 10 ✅
```

### 3. 数字 0 的处理问题

JavaScript 的 `||` 运算符会将 `0` 视为 falsy：

```javascript
// ❌ 错误
value = data.speed || '';  // 如果 speed = 0，结果是 ''

// ✅ 正确：使用空值合并运算符
value = data.speed ?? '';  // 如果 speed = 0，结果是 0
```

## 解决方案

### 方案 1：修改前端适配（已采用）

修改 `web/static/app.js` 中的字段映射：

```javascript
// fillForm() - 读取数据时使用 PascalCase（与 Go 结构体字段名一致）
document.getElementById('start_port').value = data.StartPort ?? '';
document.getElementById('stream_type').value = data.StreamType || 'fullmesh';
document.getElementById('qp_num').value = data.QpNum ?? '';
document.getElementById('message_size_bytes').value = data.MessageSizeBytes ?? '';
document.getElementById('output_base').value = data.OutputBase || '';
document.getElementById('waiting_time_seconds').value = data.WaitingTimeSeconds ?? '';
document.getElementById('rdma_cm').checked = data.RdmaCm || false;
document.getElementById('run_duration_seconds').value = data.Run?.DurationSeconds ?? '';

// collectFormData() - 发送数据时使用 snake_case（后端期望）
return {
    start_port: parseInt(document.getElementById('start_port').value) || 0,
    stream_type: document.getElementById('stream_type').value,
    qp_num: parseInt(document.getElementById('qp_num').value) || 0,
    // ...
};
```

### 方案 2：修改后端（备选）

在 Go 结构体中添加 `json` 标签：

```go
type Config struct {
    StartPort          int     `yaml:"start_port" json:"start_port"`
    StreamType         string  `yaml:"stream_type" json:"stream_type"`
    QpNum              int     `yaml:"qp_num" json:"qp_num"`
    MessageSizeBytes   int     `yaml:"message_size_bytes" json:"message_size_bytes"`
    // ...
}
```

## 字段映射对照表

| HTML Input ID | Go 结构体字段 | YAML 字段 | JSON 字段 (实际) | 前端读取 | 前端发送 |
|---------------|---------------|-----------|-----------------|---------|---------|
| start_port | StartPort | start_port | StartPort | data.StartPort | start_port |
| stream_type | StreamType | stream_type | StreamType | data.StreamType | stream_type |
| qp_num | QpNum | qp_num | QpNum | data.QpNum | qp_num |
| message_size_bytes | MessageSizeBytes | message_size_bytes | MessageSizeBytes | data.MessageSizeBytes | message_size_bytes |
| output_base | OutputBase | output_base | OutputBase | data.OutputBase | output_base |
| waiting_time_seconds | WaitingTimeSeconds | waiting_time_seconds | WaitingTimeSeconds | data.WaitingTimeSeconds | waiting_time_seconds |
| speed | Speed | speed | Speed | data.Speed | speed |
| rdma_cm | RdmaCm | rdma_cm | RdmaCm | data.RdmaCm | rdma_cm |
| run_duration_seconds | DurationSeconds | duration_seconds | DurationSeconds | data.Run.DurationSeconds | duration_seconds |

## 测试验证

1. 启动服务器：`./xnetperf server --port 8081`
2. 访问 http://localhost:8081
3. 选择配置文件 `a.yaml`
4. 验证所有字段都正确显示：
   - ✅ start_port: 20000
   - ✅ stream_type: incast
   - ✅ qp_num: 10
   - ✅ message_size_bytes: 4096
   - ✅ speed: 400
   - ✅ 等等...
5. 修改字段并保存
6. 验证保存成功并重新加载正确

## 相关文件

- `web/static/app.js` - 前端 JavaScript 代码
- `config/config.go` - Go 配置结构体定义
- `server/config_service.go` - 配置服务 API
