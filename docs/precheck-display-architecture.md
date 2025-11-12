# Precheck 展示层架构设计文档

## 设计理念

采用**多层DTO模式**，清晰分离数据、着色规则和展示逻辑。

## 架构分层

```
┌─────────────────────────────────────────────────────────────┐
│                     数据采集层 (SSH)                          │
│                   HostPrecheckData                           │
│           (原始数据，带面向对象验证方法)                        │
└──────────────────────┬──────────────────────────────────────┘
                       │ convertHostDataToResults()
                       ↓
┌─────────────────────────────────────────────────────────────┐
│                      数据层 DTO                              │
│                   PrecheckResult                             │
│              (清理后的数据，无着色信息)                         │
└──────────────────────┬──────────────────────────────────────┘
                       │ NewPrecheckDisplayData()
                       │ (应用着色规则)
                       ↓
┌─────────────────────────────────────────────────────────────┐
│                     展示层 DTO                               │
│              PrecheckDisplayData                             │
│         (包含颜色信息，着色规则已应用)                          │
│                                                              │
│  ├─ PrecheckDisplayItem (单项展示数据)                        │
│  │   ├─ Hostname, HCA (不着色)                              │
│  │   ├─ Speed: FieldColorInfo (带颜色样式)                   │
│  │   ├─ FwVer: FieldColorInfo (带颜色样式)                   │
│  │   └─ Status: FieldColorInfo (带颜色样式)                  │
│  └─ 统计信息 (HealthyCount, ErrorCount, etc.)                │
└──────────────────────┬──────────────────────────────────────┘
                       │
         ┌─────────────┴─────────────┐
         ↓                           ↓
┌──────────────────┐      ┌──────────────────┐
│   终端展示        │      │   Web UI 展示     │
│                  │      │                  │
│ ApplyColor()     │      │ GetColorClass()  │
│ → ANSI 颜色      │      │ → CSS 类名       │
└──────────────────┘      └──────────────────┘
```

## 核心组件

### 1. FieldColorInfo - 字段颜色信息

```go
type FieldColorInfo struct {
    Value      string     // 原始值
    ColorStyle ColorStyle // 颜色样式枚举
}
```

**职责**：
- 封装单个字段的值和颜色样式
- 提供多渠道的颜色应用方法

**方法**：
- `ApplyColor() string` - 应用终端ANSI颜色
- `GetColorClass() string` - 获取Web CSS类名

### 2. ColorStyle - 颜色样式枚举

```go
type ColorStyle int

const (
    ColorStyleNormal  ColorStyle = iota // 正常，不着色
    ColorStyleSuccess                   // 成功（绿色）
    ColorStyleWarning                   // 警告（黄色）
    ColorStyleError                     // 错误（红色）
)
```

**优势**：
- 统一的颜色语义
- 易于扩展新样式
- 跨平台一致性

### 3. PrecheckDisplayItem - 单项展示数据

```go
type PrecheckDisplayItem struct {
    Hostname     string         // 主机名（不着色）
    HCA          string         // HCA名称（不着色）
    SerialNumber string         // 序列号（不着色）
    PhysState    string         // 物理状态（不着色）
    State        string         // 逻辑状态（不着色）
    Speed        FieldColorInfo // 速度（可能着色）
    FwVer        FieldColorInfo // 固件版本（可能着色）
    BoardId      FieldColorInfo // 板卡ID（可能着色）
    Status       FieldColorInfo // 状态（着色）
}
```

### 4. PrecheckDisplayData - 展示数据集合

```go
type PrecheckDisplayData struct {
    Items []PrecheckDisplayItem
    
    // 统计信息
    HealthyCount   int
    UnhealthyCount int
    ErrorCount     int
    TotalCount     int
}
```

**核心方法**：
- `NewPrecheckDisplayData([]PrecheckResult) *PrecheckDisplayData` - 工厂方法
- `SortByHostAndHCA()` - 排序方法
- `applySpeedColor()` - 应用速度着色规则（私有）
- `applyFwVerColor()` - 应用固件版本着色规则（私有）
- `applyBoardIdColor()` - 应用板卡ID着色规则（私有）

## 着色规则

### Speed（速度）
- ✅ **绿色**：数量最多的速度值（正常）
- ❌ **红色**：数量较少的速度值（异常）
- ⚪ **无色**：唯一值或相同数量

### FW Version（固件版本）
- ⚠️ **黄色**：数量较少的版本（版本不一致）
- ⚪ **无色**：数量最多的版本

### Board ID（板卡ID）
- ⚠️ **黄色**：数量较少的型号（型号不一致）
- ⚪ **无色**：数量最多的型号

### Status（状态）
- ✅ **绿色**：健康（LinkUp + ACTIVE）
- ❌ **红色**：不健康
- ⚠️ **黄色**：错误

## 使用示例

### 终端展示（V2版本）

```go
func DisplayPrecheckResultsV2(results []PrecheckResult) {
    // 1. 创建展示数据（应用着色规则）
    displayData := NewPrecheckDisplayData(results)
    displayData.SortByHostAndHCA()
    
    // 2. 创建表格
    t := table.NewWriter()
    
    // 3. 填充数据（只负责展示）
    for _, item := range displayData.Items {
        t.AppendRow(table.Row{
            item.SerialNumber,
            item.Hostname,
            item.HCA,
            item.Speed.ApplyColor(),   // 应用终端颜色
            item.Status.ApplyColor(),  // 应用终端颜色
        })
    }
    
    // 4. 渲染
    t.Render()
}
```

### Web UI 展示（示例）

```go
func RenderPrecheckHTML(results []PrecheckResult) string {
    // 1. 创建展示数据
    displayData := NewPrecheckDisplayData(results)
    displayData.SortByHostAndHCA()
    
    // 2. 生成HTML
    var html strings.Builder
    html.WriteString("<table class='table'>")
    
    for _, item := range displayData.Items {
        html.WriteString("<tr>")
        html.WriteString(fmt.Sprintf("<td>%s</td>", item.Hostname))
        
        // 应用CSS类
        speedClass := item.Speed.GetColorClass()
        html.WriteString(fmt.Sprintf("<td class='%s'>%s</td>", 
            speedClass, item.Speed.Value))
        
        statusClass := item.Status.GetColorClass()
        html.WriteString(fmt.Sprintf("<td class='%s'>%s</td>", 
            statusClass, item.Status.Value))
        
        html.WriteString("</tr>")
    }
    
    html.WriteString("</table>")
    return html.String()
}
```

### API JSON 响应（示例）

```go
type PrecheckAPIResponse struct {
    Items []struct {
        Hostname string `json:"hostname"`
        Speed    struct {
            Value string `json:"value"`
            Style string `json:"style"` // "success", "warning", "error", "normal"
        } `json:"speed"`
        Status struct {
            Value string `json:"value"`
            Style string `json:"style"`
        } `json:"status"`
    } `json:"items"`
    Summary struct {
        Healthy   int `json:"healthy"`
        Unhealthy int `json:"unhealthy"`
        Errors    int `json:"errors"`
    } `json:"summary"`
}

func ToPrecheckAPIResponse(displayData *PrecheckDisplayData) PrecheckAPIResponse {
    // 前端可以根据 style 字段应用相应的样式
    // ...
}
```

## 优势总结

### 1. **单一职责原则** ✅
- `PrecheckResult`: 只负责数据
- `PrecheckDisplayData`: 只负责着色规则
- `DisplayPrecheckResultsV2`: 只负责渲染

### 2. **开闭原则** ✅
- 新增颜色样式：扩展 `ColorStyle` 枚举
- 新增着色规则：添加新的 `apply*Color` 方法
- 新增展示渠道：实现新的 `FieldColorInfo` 方法

### 3. **复用性** ✅
- 同一份 `PrecheckDisplayData` 可用于：
  - 终端展示（ANSI颜色）
  - Web UI（CSS类名）
  - API响应（颜色样式字段）
  - Excel导出（单元格背景色）

### 4. **可测试性** ✅
```go
func TestSpeedColorRule(t *testing.T) {
    results := []PrecheckResult{
        {Speed: "200Gb/sec"},
        {Speed: "200Gb/sec"},
        {Speed: "100Gb/sec"}, // 应该标红
    }
    
    displayData := NewPrecheckDisplayData(results)
    
    // 验证着色规则
    assert.Equal(t, ColorStyleSuccess, displayData.Items[0].Speed.ColorStyle)
    assert.Equal(t, ColorStyleError, displayData.Items[2].Speed.ColorStyle)
}
```

### 5. **可维护性** ✅
- 着色规则集中在一处（`NewPrecheckDisplayData`）
- 展示逻辑与着色规则解耦
- 易于理解和修改

## 测试方法

```bash
# 原版本（手动拼接）
./xnetperf precheck

# V1版本（go-pretty/table 基础版）
PRECHECK_UI=v1 ./xnetperf precheck

# V2版本（优雅分层，推荐）
PRECHECK_UI=v2 ./xnetperf precheck
```

## 未来扩展

### 1. 主题支持
```go
type ColorTheme interface {
    Success() string
    Warning() string
    Error() string
}

type DarkTheme struct{}
func (t *DarkTheme) Success() string { return "\033[92m" } // 亮绿

type LightTheme struct{}
func (t *LightTheme) Success() string { return "\033[32m" } // 深绿
```

### 2. 国际化支持
```go
func (f *FieldColorInfo) GetLocalizedValue(lang string) string {
    // 根据语言返回本地化文本
}
```

### 3. 自定义着色规则
```go
type ColorRuleFunc func(value string, count, maxCount int) ColorStyle

func NewPrecheckDisplayDataWithRules(
    results []PrecheckResult,
    speedRule ColorRuleFunc,
    fwVerRule ColorRuleFunc,
) *PrecheckDisplayData {
    // 允许自定义着色规则
}
```

## 总结

这个设计完美体现了你提出的 **OOP + DDD + DTO** 设计思想：

1. **OOP**：对象封装数据和行为（`FieldColorInfo.ApplyColor()`）
2. **DDD**：清晰的领域边界（数据层 → 展示层）
3. **DTO**：数据的生产和展示分离，同一份数据可以展示在不同的地方

**着色和展示分离**使得代码更加优雅、可维护、可扩展！🎨
