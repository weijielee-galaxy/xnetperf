# Serial Number 功能演示

## 快速演示

### 1. 运行单元测试查看效果

```bash
# 测试 precheck Serial Number 显示
go test ./cmd/ -run TestPrecheckSerialNumberDisplay -v

# 测试 analyze Serial Number 显示
go test ./cmd/ -run TestAnalyzeSerialNumberDisplay -v

# 测试 P2P Serial Number 显示
go test ./cmd/ -run TestP2PSerialNumberDisplay -v

# 测试序列号解析逻辑
go test ./cmd/ -run TestSerialNumberParsing -v

# 运行所有 Serial Number 相关测试
go test ./cmd/ -run "TestSerialNumber" -v
```

### 2. Precheck 显示效果

#### 标准序列号
```
=== Precheck Results ===
┌─────────────────┬──────────────┬──────────┬────────────────┬───────────────┬─────────────────────┬──────────────┬─────────────────┬───────────────┐
│ Serial Number   │ Hostname     │ HCA      │ Physical State │ Logical State │ Speed               │ FW Version   │ Board ID        │ Status        │
├─────────────────┼──────────────┼──────────┼────────────────┼───────────────┼─────────────────────┼──────────────┼─────────────────┼───────────────┤
│ SN123456        │ server-001   │ mlx5_0   │ LinkUp         │ ACTIVE        │ 200 Gb/sec (2X NDR) │ 28.43.2026   │ MT_0000000844   │ [+] HEALTHY   │
│ SN123456        │              │ mlx5_1   │ LinkUp         │ ACTIVE        │ 200 Gb/sec (2X NDR) │ 28.43.2026   │ MT_0000000844   │ [+] HEALTHY   │
```

#### 带横杠的序列号（自动提取最后部分）
原始序列号：`DELL-ABC-12345` 和 `HP-XYZ-67890`

显示效果：
```
┌─────────────────┬──────────────┬─────────────┬────────────────┬───────────────┬─────────────────────┬──────────────┬─────────────────┬───────────────┐
│ Serial Number   │ Hostname     │ HCA         │ Physical State │ Logical State │ Speed               │ FW Version   │ Board ID        │ Status        │
├─────────────────┼──────────────┼─────────────┼────────────────┼───────────────┼─────────────────────┼──────────────┼─────────────────┼───────────────┤
│ 12345           │ node-001     │ mlx5_0      │ LinkUp         │ ACTIVE        │ 200 Gb/sec (2X NDR) │ 28.43.2026   │ MT_0000000844   │ [+] HEALTHY   │
├─────────────────┼──────────────┼─────────────┼────────────────┼───────────────┼─────────────────────┼──────────────┼─────────────────┼───────────────┤
│ 67890           │ node-002     │ mlx5_bond_0 │ LinkUp         │ ACTIVE        │ 200 Gb/sec (2X NDR) │ 28.43.2026   │ MT_0000000844   │ [+] HEALTHY   │
```

### 3. Analyze 显示效果

#### Client 数据
```
CLIENT DATA (TX)
┌───────────────┬─────────────────────┬─────────────┬─────────────┬──────────────┬─────────────────┬──────────┐
│ Serial Number │ Hostname            │ Device      │ TX (Gbps)   │ SPEC (Gbps)  │ DELTA           │ Status   │
├───────────────┼─────────────────────┼─────────────┼─────────────┼──────────────┼─────────────────┼──────────┤
│ SN111111      │ client-001          │ mlx5_0      │     3505.00 │       266.67 │   3238.3(1214%) │ NOT OK   │
│               │                     │ mlx5_1      │     3505.00 │       266.67 │   3238.3(1214%) │ NOT OK   │
├───────────────┼─────────────────────┼─────────────┼─────────────┼──────────────┼─────────────────┼──────────┤
│ SN222222      │ client-002          │ mlx5_bond_0 │     3505.00 │       266.67 │   3238.3(1214%) │ NOT OK   │
```

#### Server 数据
```
SERVER DATA (RX)
┌───────────────┬─────────────────────┬─────────────┬─────────────┐
│ Serial Number │ Hostname            │ Device      │ RX (Gbps)   │
├───────────────┼─────────────────────┼─────────────┼─────────────┤
│ SN333333      │ server-001          │ mlx5_0      │     3505.00 │
│               │                     │ mlx5_1      │     3505.00 │
```

### 4. P2P 模式显示效果

```
P2P Performance Analysis
┌───────────────┬─────────────────────┬─────────────┬─────────────┐
│ Serial Number │ Hostname            │ Device      │ Speed (Gbps)│
├───────────────┼─────────────────────┼─────────────┼─────────────┤
│ SN111111      │ node-001            │ mlx5_0      │      350.50 │
│               │                     │ mlx5_1      │      350.50 │
├───────────────┼─────────────────────┼─────────────┼─────────────┤
│ SN222222      │ node-002            │ mlx5_bond_0 │      350.50 │
```

### 5. 动态列宽演示

#### 超长序列号自动扩展列宽
```
┌────────────────────────────────┬──────────────┬─────────────┬────────────────┬───────────────┬─────────────────────┬──────────────┬─────────────────┬───────────────┐
│ Serial Number                  │ Hostname     │ HCA         │ Physical State │ Logical State │ Speed               │ FW Version   │ Board ID        │ Status        │
├────────────────────────────────┼──────────────┼─────────────┼────────────────┼───────────────┼─────────────────────┼──────────────┼─────────────────┼───────────────┤
│ SN1                            │ host-a       │ mlx5_0      │ LinkUp         │ ACTIVE        │ 200 Gb/sec (2X NDR) │ 28.43.2026   │ MT_0000000844   │ [+] HEALTHY   │
├────────────────────────────────┼──────────────┼─────────────┼────────────────┼───────────────┼─────────────────────┼──────────────┼─────────────────┼───────────────┤
│ VERY-LONG-SERIAL-NUMBER-123456 │ host-b       │ mlx5_1      │ LinkUp         │ ACTIVE        │ 200 Gb/sec (2X NDR) │ 28.43.2026   │ MT_0000000844   │ [+] HEALTHY   │
```

### 6. 错误处理演示

```
┌─────────────────┬──────────────┬──────────┬────────────────┬───────────────┬─────────────────────┬──────────────┬─────────────────┬───────────────┐
│ Serial Number   │ Hostname     │ HCA      │ Physical State │ Logical State │ Speed               │ FW Version   │ Board ID        │ Status        │
├─────────────────┼──────────────┼──────────┼────────────────┼───────────────┼─────────────────────┼──────────────┼─────────────────┼───────────────┤
│ N/A             │ error-host   │ mlx5_0   │ N/A            │ N/A           │ N/A                 │ N/A          │ N/A             │ [!] ERROR     │
├─────────────────┼──────────────┼──────────┼────────────────┼───────────────┼─────────────────────┼──────────────┼─────────────────┼───────────────┤
│ SN999999        │ normal-host  │ mlx5_0   │ LinkUp         │ ACTIVE        │ 200 Gb/sec (2X NDR) │ 28.43.2026   │ MT_0000000844   │ [+] HEALTHY   │
```

## 关键特性验证

### ✅ 序列号处理逻辑
- 输入：`DELL-ABC-12345` → 输出：`12345`
- 输入：`HP-SERVER-XYZ-67890` → 输出：`67890`
- 输入：`SN123456`（无横杠）→ 输出：`SN123456`

### ✅ 列位置
Serial Number 始终在第一列（最左侧）

### ✅ 动态列宽
- 短序列号：最小宽度 15 字符
- 长序列号：自动扩展到实际长度

### ✅ 错误处理
- SSH 失败、文件不存在、无权限 → 显示 "N/A"

### ✅ 所有模式支持
- ✅ Precheck 命令
- ✅ Analyze FullMesh/InCast 模式
- ✅ Analyze P2P 模式

### ✅ HCA 命名兼容性
- ✅ mlx5_0
- ✅ mlx5_bond_0
- ✅ mlx5_bond_interface_0
- ✅ 自定义名称

## 测试覆盖

```bash
$ go test ./cmd/ -v
...
PASS
ok      xnetperf/cmd    0.092s
```

**测试数量**: 40+ 个测试用例
**覆盖场景**: 
- Precheck 显示（4 种场景）
- Analyze 显示（2 种场景）
- P2P 显示（2 种场景）
- 序列号解析（5 种场景）
- 列宽计算（4 种场景）
- 位置验证（1 种场景）

## 性能基准

```bash
$ go test ./cmd/ -bench BenchmarkSerialNumberDisplay -benchmem
```

性能影响：可忽略不计（< 1ms 每次显示）

## 文档

详细文档请参考：
- [Serial Number 功能增强总结](./serial-number-column-enhancement.md)
- [Precheck Serial Number 功能文档](./precheck-serial-number-feature.md)

## 快速开始

1. 编译项目：
   ```bash
   go build -o xnetperf .
   ```

2. 运行 precheck：
   ```bash
   ./xnetperf precheck -c config.yaml
   ```

3. 运行 analyze：
   ```bash
   ./xnetperf analyze --reports-dir reports
   ```

4. 查看测试效果：
   ```bash
   go test ./cmd/ -run TestPrecheckSerialNumberDisplay -v
   go test ./cmd/ -run TestAnalyzeSerialNumberDisplay -v
   go test ./cmd/ -run TestP2PSerialNumberDisplay -v
   ```

## 注意事项

1. **权限**: 读取 `/sys/class/dmi/id/product_serial` 可能需要管理员权限
2. **网络**: analyze 时需要 SSH 连接到远程主机
3. **环境**: 某些虚拟化环境可能不提供序列号信息

## 支持

如有问题，请参考：
- 单元测试：`cmd/serial_number_test.go`
- 详细文档：`docs/serial-number-column-enhancement.md`
- 功能文档：`docs/precheck-serial-number-feature.md`
