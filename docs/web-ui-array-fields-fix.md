# Web UI Server/Client 数组字段修复

## 问题描述

在 Web UI 中保存配置时，`server` 和 `client` 的 `hostname` 和 `hca` 数组字段被保存为空数组 `[]`，即使在界面上添加了值。

## 问题原因

### 数据流中的大小写转换问题

1. **API 返回数据**（GET）- PascalCase：
```json
{
  "data": {
    "Server": {
      "Hostname": ["server1", "server2"],
      "Hca": ["mlx5_0", "mlx5_1"]
    },
    "Client": {
      "Hostname": ["client1"],
      "Hca": ["mlx5_0"]
    }
  }
}
```

2. **前端存储** - 直接使用 API 返回的 PascalCase：
```javascript
this.formData = result.data;  // formData.Server, formData.Client
```

3. **添加/删除标签** - 使用 snake_case 查找：
```javascript
// ❌ 错误代码
addTag(type, value) {
    const [section, field] = type.split('_');  // ['server', 'hostname']
    this.formData[section][field].push(value);  // this.formData.server.hostname - undefined!
}
```

4. **收集表单数据** - 使用错误的小写键：
```javascript
// ❌ 错误代码
server: this.formData.server || { hostname: [], hca: [] }  // undefined
```

### 完整的问题链

```
用户添加 hostname → 
addTag('server_hostname', 'host1') → 
尝试访问 this.formData.server.hostname → 
undefined（实际是 this.formData.Server.Hostname）→ 
创建新数组但没保存到正确位置 → 
collectFormData() 读取 this.formData.server → 
返回空数组
```

## 解决方案

### 1. 修复 addTag() 函数

```javascript
addTag(type, value) {
    if (!value || !value.trim()) return;
    
    // type 格式: 'server_hostname' 或 'client_hca'
    const [sectionLower, fieldLower] = type.split('_');
    
    // 转换为 PascalCase 以匹配 API 返回的数据格式
    const section = sectionLower.charAt(0).toUpperCase() + sectionLower.slice(1); // 'Server'
    const field = fieldLower.charAt(0).toUpperCase() + fieldLower.slice(1); // 'Hostname'
    
    if (!this.formData[section]) {
        this.formData[section] = {};
    }
    if (!this.formData[section][field]) {
        this.formData[section][field] = [];
    }
    
    this.formData[section][field].push(value.trim());
    this.renderTagList(`${type}_list`, this.formData[section][field], type);
}
```

### 2. 修复 removeTag() 函数

```javascript
removeTag(type, index) {
    const [sectionLower, fieldLower] = type.split('_');
    
    // 转换为 PascalCase
    const section = sectionLower.charAt(0).toUpperCase() + sectionLower.slice(1);
    const field = fieldLower.charAt(0).toUpperCase() + fieldLower.slice(1);
    
    this.formData[section][field].splice(index, 1);
    this.renderTagList(`${type}_list`, this.formData[section][field], type);
}
```

### 3. 修复 collectFormData() 函数

```javascript
collectFormData() {
    return {
        // ... 其他字段 ...
        
        // 使用 PascalCase 从 formData 读取，转换为 snake_case 发送
        server: {
            hostname: this.formData.Server?.Hostname || [],
            hca: this.formData.Server?.Hca || []
        },
        client: {
            hostname: this.formData.Client?.Hostname || [],
            hca: this.formData.Client?.Hca || []
        }
    };
}
```

## 数据映射关系

| 操作 | 前端内部存储 | API 发送/接收 |
|------|-------------|--------------|
| API 返回 (GET) | `formData.Server.Hostname` | `"Server": {"Hostname": [...]}` |
| 添加标签 | `formData.Server.Hostname.push()` | - |
| 删除标签 | `formData.Server.Hostname.splice()` | - |
| 保存配置 (PUT) | `formData.Server.Hostname` | `"server": {"hostname": [...]}` |

注意：
- **前端内部使用 PascalCase**（与 Go 结构体字段名一致）
- **发送给后端使用 snake_case**（与 YAML 字段名一致）

## 测试步骤

1. 启动服务器：`./xnetperf server --port 8080`
2. 打开浏览器：http://localhost:8080
3. 选择配置文件（如 `prod-config.yaml`）
4. **添加 server hostname**：
   - 在 "服务器配置 (server)" → "主机名列表 (hostname)" 输入框输入 `server-001`
   - 点击"添加"按钮
   - 验证标签显示出来 ✅
5. **添加 server hca**：
   - 在 "HCA 设备列表 (hca)" 输入框输入 `mlx5_0`
   - 点击"添加"
   - 验证标签显示 ✅
6. **添加 client 配置**：
   - 类似地添加 client 的 hostname 和 hca
7. **保存配置**：
   - 点击右上角"💾 保存"按钮
   - 验证提示"保存成功" ✅
8. **验证保存结果**：
   - 重新选择该配置文件
   - 验证所有添加的值都正确显示 ✅
9. **检查 YAML 文件**：
```bash
cat build/configs/prod-config.yaml
```
应该看到：
```yaml
server:
    hostname:
        - server-001
    hca:
        - mlx5_0
client:
    hostname:
        - client-001
    hca:
        - mlx5_1
```

## 相关文件

- `web/static/app.js` - 前端 JavaScript
  - `addTag()` - 添加数组元素
  - `removeTag()` - 删除数组元素
  - `collectFormData()` - 收集表单数据
  - `fillForm()` - 填充表单
