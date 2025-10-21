# 静态编译指南

## 概述

为了解决不同 GLIBC 版本之间的兼容性问题，xnetperf 默认使用静态编译。静态编译的二进制文件可以在任何 Linux 发行版上运行，无需担心依赖库版本不匹配的问题。

## 编译选项

### 1. 静态编译（推荐，默认）

```bash
make                  # 默认使用静态编译
make build            # 同上
make build-static     # 明确指定静态编译
```

**特点**：
- ✅ 无 GLIBC 版本依赖
- ✅ 可在任何 Linux 系统运行
- ✅ 无需安装额外依赖
- ⚠️ 二进制文件较大（约 22MB）

**编译参数**：
```makefile
CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-s -w -extldflags '-static'" -o build/xnetperf .
```

**参数说明**：
- `CGO_ENABLED=0`：禁用 CGO，允许完全静态链接
- `GOOS=linux`：目标操作系统为 Linux
- `-a`：强制重新编译所有包
- `-ldflags="-s -w -extldflags '-static'"`：
  - `-s`：去除符号表（减小文件大小）
  - `-w`：去除 DWARF 调试信息（减小文件大小）
  - `-extldflags '-static'`：强制静态链接

### 2. 动态编译

```bash
make build-dynamic
```

**特点**：
- ⚠️ 依赖系统 GLIBC 版本
- ⚠️ 可能在不同系统上无法运行
- ✅ 二进制文件较小（约 15MB）

**适用场景**：
- 仅在本地开发和测试
- 确定目标系统的 GLIBC 版本

### 3. 可移植编译

```bash
make build-portable
```

**特点**：
- ✅ 禁用 CGO
- ⚠️ 不保证完全静态链接
- ✅ 较好的兼容性

**适用场景**：
- CGO 功能不需要时使用
- 需要在多个系统上运行，但不需要完全静态

## 验证静态链接

### 使用 ldd 命令

```bash
ldd build/xnetperf
```

**预期输出**（静态编译）：
```
not a dynamic executable
```

**如果是动态编译，会显示**：
```
linux-vdso.so.1 (0x00007ffd1234abcd)
libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6 (0x00007f1234567890)
/lib64/ld-linux-x86-64.so.2 (0x00007f9876543210)
```

### 使用 file 命令

```bash
file build/xnetperf
```

**预期输出**（静态编译）：
```
build/xnetperf: ELF 64-bit LSB executable, x86-64, version 1 (SYSV), 
statically linked, BuildID[sha1]=..., stripped
```

关键词：`statically linked`, `stripped`

## 其他 Makefile 命令

### 开发相关

```bash
make run              # 直接运行（不编译二进制文件）
make test             # 运行所有测试
make test-verbose     # 运行测试（详细输出）
make test-coverage    # 运行测试并显示覆盖率
```

### 清理

```bash
make clean            # 清理构建产物
```

### 帮助

```bash
make help             # 显示所有可用命令
```

## 常见问题

### Q1: 为什么静态编译的文件这么大？

**A**: 静态编译将所有依赖库都打包到二进制文件中，所以文件较大。但这是为了兼容性的权衡：
- 静态编译：约 22MB，可在任何 Linux 系统运行
- 动态编译：约 15MB，但可能无法在不同系统运行

### Q2: 如何减小静态编译的文件大小？

**A**: Makefile 已经使用了以下优化：
- `-s`：去除符号表
- `-w`：去除调试信息

如果需要进一步压缩，可以使用 UPX：
```bash
upx --best --lzma build/xnetperf
```

⚠️ 注意：压缩后可能会被某些防病毒软件误报。

### Q3: 静态编译的二进制在所有 Linux 系统都能运行吗？

**A**: 理论上是的，只要是 x86-64 架构的 Linux 系统。但需要注意：
- ✅ 不同 Linux 发行版（Ubuntu, CentOS, Debian 等）
- ✅ 不同 GLIBC 版本
- ⚠️ 不同 CPU 架构（需要交叉编译）
- ⚠️ 特殊的内核配置可能有问题

### Q4: 为什么禁用 CGO？

**A**: CGO 允许 Go 调用 C 代码，但会引入动态链接依赖：
- 启用 CGO：可能依赖 glibc, gcc 等 C 库
- 禁用 CGO：可以完全静态链接，无外部依赖

xnetperf 不需要 C 库功能，所以禁用 CGO 是安全的。

### Q5: 如何为其他架构编译？

**A**: 修改 `GOOS` 和 `GOARCH` 环境变量：

```bash
# ARM64 (例如：树莓派 4, AWS Graviton)
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -ldflags="-s -w" -o build/xnetperf-arm64 .

# ARM (32-bit, 例如：树莓派 3)
CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -a -ldflags="-s -w" -o build/xnetperf-arm .

# Windows
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -a -ldflags="-s -w" -o build/xnetperf.exe .

# macOS
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -a -ldflags="-s -w" -o build/xnetperf-darwin .
```

## 部署建议

### 生产环境

```bash
# 1. 编译静态二进制
make build-static

# 2. 验证静态链接
ldd build/xnetperf  # 应该显示 "not a dynamic executable"

# 3. 复制到目标系统
scp build/xnetperf user@remote-host:/path/to/install/

# 4. 在目标系统上运行（无需安装依赖）
./xnetperf --version
```

### 开发环境

```bash
# 快速开发测试（无需编译）
make run

# 或者动态编译（编译更快）
make build-dynamic
./build/xnetperf
```

## 性能对比

| 编译方式 | 文件大小 | 编译时间 | 兼容性 | 性能 |
|---------|---------|---------|--------|------|
| 静态编译 | ~22 MB | 约 30s | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| 动态编译 | ~15 MB | 约 20s | ⭐⭐ | ⭐⭐⭐⭐⭐ |
| 可移植编译 | ~20 MB | 约 25s | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |

**注意**：运行时性能基本相同，主要差异在文件大小和兼容性。

## 持续集成 (CI/CD)

在 CI/CD 流程中使用静态编译：

### GitHub Actions 示例

```yaml
name: Build
on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Build static binary
        run: make build-static
      
      - name: Verify static linking
        run: |
          if ldd build/xnetperf 2>&1 | grep -q "not a dynamic executable"; then
            echo "✅ Static linking verified"
          else
            echo "❌ Binary is not statically linked"
            exit 1
          fi
      
      - name: Upload artifact
        uses: actions/upload-artifact@v3
        with:
          name: xnetperf-linux-amd64
          path: build/xnetperf
```

## 相关资源

- [Go 编译选项文档](https://pkg.go.dev/cmd/go)
- [Go 交叉编译指南](https://golang.org/doc/install/source#environment)
- [CGO 文档](https://golang.org/cmd/cgo/)

## 总结

**推荐做法**：
1. ✅ 生产环境使用 `make build-static`（默认）
2. ✅ 开发环境使用 `make run` 或 `make build-dynamic`
3. ✅ 部署前验证静态链接：`ldd build/xnetperf`
4. ✅ 定期测试在不同 Linux 发行版上的兼容性

**关键优势**：
- 无 GLIBC 版本冲突
- 一次编译，到处运行
- 无需安装运行时依赖
- 简化部署流程
