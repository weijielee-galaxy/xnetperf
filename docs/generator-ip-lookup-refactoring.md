# Generator IP Lookup重构文档

## 重构背景

在实现`generator_bw_fullmesh.go`和`generator_bw_incast.go`时,发现原设计将IP查询逻辑(通过SSH调用`LookupServerHostsIP()`)放在了`GenerateScripts()`方法内部,这导致了以下问题:

1. **违反单一职责原则**: Generator既负责脚本生成,又负责网络查询(SSH)
2. **测试困难**: 需要创建`WithServerIPs()`和`WithHostIPs()`等注入方法来绕过SSH调用
3. **代码耦合**: Generator与SSH强耦合,无法进行纯函数式测试
4. **设计不合理**: IP查询应该是调用方的职责,而非脚本生成器的职责

## 重构方案

采用**方案1**:将IP作为参数传入`GenerateScripts()`方法

### 优点
- 简单直接,符合函数式编程思想
- 职责清晰:调用方负责IP查询,generator只负责脚本生成
- 测试简单:直接传入mock IP即可,无需注入机制
- 向后兼容:不影响接口设计

### 设计原则
- **单一职责**: Generator只负责根据配置和IP生成脚本
- **依赖倒置**: 不依赖具体的SSH实现,只依赖IP数据
- **可测试性**: 纯函数,容易mock和测试

## 修改内容

### 1. generator_bw_incast.go

**修改前**:
```go
type bwIncastScriptGenerator struct {
    cfg       *config.Config
    serverIPs map[string]string // Optional: for testing
}

func (g *bwIncastScriptGenerator) GenerateScripts() (*ScriptResult, error) {
    var serverIps map[string]string
    if g.serverIPs != nil {
        serverIps = g.serverIPs
    } else {
        serverIps, err = g.cfg.LookupServerHostsIP() // SSH调用
        // ...
    }
    // ...
}
```

**修改后**:
```go
type bwIncastScriptGenerator struct {
    cfg *config.Config
}

func (g *bwIncastScriptGenerator) GenerateScripts(serverIps map[string]string) (*ScriptResult, error) {
    // 直接使用传入的serverIps
    // ...
}
```

### 2. generator_bw_fullmesh.go

**修改前**:
```go
type bwFullmeshScriptGenerator struct {
    cfg    *config.Config
    allIPs map[string]string // Optional: for testing
}

func (g *bwFullmeshScriptGenerator) GenerateScripts() (*ScriptResult, error) {
    var allIPs map[string]string
    if g.allIPs != nil {
        allIPs = g.allIPs
    } else {
        // 复杂的IP查询逻辑,包含SSH调用
        allIPs = make(map[string]string)
        for _, host := range allHosts {
            if contains(g.cfg.Server.Hostname, host) {
                serverIps, err := g.cfg.LookupServerHostsIP()
                // ...
            }
            if contains(g.cfg.Client.Hostname, host) {
                clientIps, err := g.cfg.LookupClientHostsIP()
                // ...
            }
        }
    }
    // ...
}
```

**修改后**:
```go
type bwFullmeshScriptGenerator struct {
    cfg *config.Config
}

func (g *bwFullmeshScriptGenerator) GenerateScripts(allIPs map[string]string) (*ScriptResult, error) {
    // 直接使用传入的allIPs
    // ...
}
```

### 3. 测试文件更新

**修改前**:
```go
gen := generator.NewBwIncastScriptGenerator(cfg).WithServerIPs(mockIPs)
result, err := gen.GenerateScripts()
```

**修改后**:
```go
gen := generator.NewBwIncastScriptGenerator(cfg)
result, err := gen.GenerateScripts(mockIPs)
```

## 调用方责任

重构后,调用方需要负责IP查询:

```go
// 1. 创建generator
gen := generator.NewBwIncastScriptGenerator(cfg)

// 2. 查询IP (调用方的责任)
serverIPs, err := cfg.LookupServerHostsIP()
if err != nil {
    return err
}

// 3. 生成脚本
result, err := gen.GenerateScripts(serverIPs)
```

对于fullmesh:
```go
gen := generator.NewBwFullmeshScriptGenerator(cfg)

// 查询所有主机IP
allHosts := append(cfg.Server.Hostname, cfg.Client.Hostname...)
allIPs := make(map[string]string)

serverIPs, err := cfg.LookupServerHostsIP()
if err != nil {
    return err
}
clientIPs, err := cfg.LookupClientHostsIP()
if err != nil {
    return err
}

// 合并IP
for host, ip := range serverIPs {
    allIPs[host] = ip
}
for host, ip := range clientIPs {
    allIPs[host] = ip
}

// 生成脚本
result, err := gen.GenerateScripts(allIPs)
```

## 测试验证

重构后所有测试通过:
```
ok      xnetperf/internal/script/generator      0.020s
```

测试覆盖:
- ✅ 基本配置测试
- ✅ 多主机多HCA测试
- ✅ 报告生成测试
- ✅ 端口耗尽测试
- ✅ 命令格式验证
- ✅ 端口分配验证

## 收益总结

1. **代码更简洁**: 移除了`WithServerIPs()`、`WithHostIPs()`等注入方法
2. **职责更清晰**: Generator只负责脚本生成,不涉及网络IO
3. **测试更简单**: 直接传入mock数据,无需复杂的注入机制
4. **设计更合理**: 符合单一职责原则和依赖倒置原则
5. **扩展性更好**: 未来可以支持从其他数据源获取IP (数据库、配置文件等)

## 经验教训

**重要**: 当发现测试需要大量mock和注入机制时,应该停下来思考设计是否合理,而不是继续为不合理的设计编写复杂的测试代码。

正确的开发流程应该是:
1. 发现测试困难 → 
2. 分析设计问题 → 
3. 重构设计 → 
4. 编写简单测试

而不是:
1. 发现测试困难 → 
2. 创建复杂mock机制 → 
3. 继续调试测试 → 
4. 最终花费大量时间
