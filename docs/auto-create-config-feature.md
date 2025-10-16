# Auto-create config.yaml for server command

## 功能说明

当执行 `xnetperf server` 命令时，系统会自动检查当前目录是否存在 `config.yaml` 文件。如果不存在，会自动创建一个包含默认值的配置文件。

## 实现细节

### 1. 新增函数 (config/config.go)

#### SaveConfig
```go
func SaveConfig(filePath string, cfg *Config) error
```
- 将配置对象序列化为 YAML 格式并保存到文件
- 文件权限设置为 `0644`
- 返回错误信息（如果有）

#### EnsureConfigFile
```go
func EnsureConfigFile(filePath string) error
```
- 检查指定路径的配置文件是否存在
- 如果不存在，创建一个包含默认值的配置文件
- 如果已存在，不做任何操作（不会覆盖现有配置）
- 创建成功时会打印提示信息：`Created default config file: <path>`

### 2. 修改 server 命令 (cmd/server.go)

在 `runServer` 函数中添加了配置文件检查：

```go
func runServer(cmd *cobra.Command, args []string) {
	// Ensure config.yaml exists in current directory
	if err := config.EnsureConfigFile("config.yaml"); err != nil {
		fmt.Printf("Warning: %v\n", err)
	}
	
	srv := server.NewServer(serverPort)
	if err := srv.Start(); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}
```

## 默认配置内容

自动创建的 `config.yaml` 包含以下默认值：

```yaml
start_port: 20000
stream_type: incast
qp_num: 10
message_size_bytes: 4096
output_base: ./generated_scripts
waiting_time_seconds: 15
speed: 400
rdma_cm: false
report:
  enable: true
  dir: /root
run:
  infinitely: false
  duration_seconds: 10
ssh:
  private_key: ~/.ssh/id_rsa
server:
  hostname: []
  hca: []
client:
  hostname: []
  hca: []
```

## 使用场景

### 场景 1: 首次使用
```bash
# 在一个空目录中启动 server
cd /path/to/empty/dir
xnetperf server
```

输出：
```
Created default config file: config.yaml
Starting HTTP server on port 8080...
```

### 场景 2: 已有配置文件
```bash
# 在已有 config.yaml 的目录中启动 server
cd /path/to/existing/config
xnetperf server
```

输出：
```
Starting HTTP server on port 8080...
```
（不会创建新文件，也不会覆盖现有配置）

### 场景 3: 自定义端口
```bash
xnetperf server --port 9090
```

输出：
```
Created default config file: config.yaml
Starting HTTP server on port 9090...
```

## 测试

新增了以下测试用例（config/config_save_test.go）：

1. **TestSaveConfig** - 测试保存和加载配置
2. **TestEnsureConfigFile_CreatesNewFile** - 测试自动创建新配置文件
3. **TestEnsureConfigFile_ExistingFile** - 测试不覆盖现有配置文件

运行测试：
```bash
go test ./config -run "TestSave|TestEnsure" -v
```

所有测试通过 ✅

## 优点

1. **用户友好**：新用户不需要手动创建配置文件
2. **零配置启动**：可以直接运行 `xnetperf server` 而无需准备配置
3. **安全**：不会覆盖现有的配置文件
4. **灵活**：自动创建的配置包含所有默认值，用户可以根据需要修改

## 向后兼容性

该功能完全向后兼容：
- 如果用户已经有 `config.yaml`，行为不变
- 只在文件不存在时才创建
- 不改变任何现有命令的行为
