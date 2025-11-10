# xnetperf Web UI 架构文档

> **版本**: v0.2.0  
> **更新日期**: 2025-11-05  
> **技术栈**: React 18 + Chakra UI + Vite

## 📖 目录

- [概述](#概述)
- [技术架构](#技术架构)
- [目录结构](#目录结构)
- [API 层设计](#api-层设计)
- [页面组件](#页面组件)
- [UI 组件](#ui-组件)
- [状态管理](#状态管理)
- [工作流设计](#工作流设计)
- [数据流](#数据流)
- [最佳实践](#最佳实践)

---

## 概述

xnetperf Web UI 是一个基于 React 的单页应用（SPA），提供了图形化界面来管理配置文件、执行网络性能测试和查看测试报告。前端通过 RESTful API 与后端 Go HTTP Server 通信。

### 核心特性

- **配置管理** - 可视化编辑和管理 YAML 配置文件
- **字典管理** - 维护主机名和 HCA 设备的预定义列表
- **流量测试** - 完整的测试工作流：Precheck → Run → Probe → Collect → Report
- **实时反馈** - 实时显示测试进度和状态
- **响应式设计** - 适配不同屏幕尺寸

---

## 技术架构

### 技术栈

| 层次 | 技术 | 用途 |
|------|------|------|
| **前端框架** | React 18 | UI 渲染和状态管理 |
| **UI 组件库** | Chakra UI | 统一的设计系统和 UI 组件 |
| **构建工具** | Vite | 快速开发和打包 |
| **HTTP 客户端** | Fetch API | 与后端 API 通信 |
| **路由管理** | Chakra Tabs | 页面导航（简化版 SPA） |

### 架构模式

```
┌─────────────────────────────────────────────────────────┐
│                      Browser                             │
├─────────────────────────────────────────────────────────┤
│                                                           │
│  ┌─────────────────────────────────────────────────┐   │
│  │              App.jsx (主容器)                      │   │
│  │  ┌────────────────────────────────────────────┐ │   │
│  │  │         Tabs (页面导航)                      │ │   │
│  │  │  • ConfigPage (配置管理)                     │ │   │
│  │  │  • DictionaryPage (字典管理)                 │ │   │
│  │  │  • TrafficTestPage (流量测试)                │ │   │
│  │  └────────────────────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────┘   │
│                         ↕                                │
│  ┌─────────────────────────────────────────────────┐   │
│  │              api.js (API 层)                      │   │
│  │  • fetchJSON (统一封装)                           │   │
│  │  • 配置管理 API                                    │   │
│  │  • 字典管理 API                                    │   │
│  │  • 测试执行 API                                    │   │
│  └─────────────────────────────────────────────────┘   │
│                         ↕                                │
└─────────────────────────────────────────────────────────┘
                          ↕ HTTP
┌─────────────────────────────────────────────────────────┐
│                 Go HTTP Server                           │
│              (Gin + REST API)                            │
└─────────────────────────────────────────────────────────┘
```

---

## 目录结构

```
web/src/
├── main.jsx              # 应用入口，初始化 React 和 ChakraProvider
├── App.jsx               # 主应用组件，管理全局状态和路由
├── api.js                # API 封装层，所有 HTTP 请求
├── pages/                # 页面级组件
│   ├── ConfigPage.jsx        # 配置管理页面（布局容器）
│   ├── DictionaryPage.jsx    # 字典管理页面
│   └── TrafficTestPage.jsx   # 流量测试页面（工作流引擎）
└── components/           # 可复用 UI 组件
    ├── ConfigList.jsx        # 配置文件列表（侧边栏）
    ├── ConfigEditor.jsx      # 配置文件编辑器（表单）
    ├── ProbeResults.jsx      # 探测结果展示
    └── ReportResults.jsx     # 性能报告展示
```

---

## API 层设计

### api.js 架构

`api.js` 是前端与后端通信的唯一入口，提供统一的 API 封装。

#### 核心设计原则

1. **统一封装** - `fetchJSON()` 统一处理响应和错误
2. **错误处理** - 自动处理 HTTP 错误和业务错误
3. **类型安全** - 返回解包后的 `data` 字段
4. **特殊处理** - 验证 API 特殊处理验证错误详情

#### fetchJSON 核心函数

```javascript
async function fetchJSON(url, options = {}) {
  // 1. 发送 HTTP 请求
  const response = await fetch(url, options)
  
  // 2. 检查 HTTP 状态
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`)
  }
  
  // 3. 解析 JSON
  const text = await response.text()
  if (!text || text.trim() === '') {
    throw new Error('Empty response from server')
  }
  const result = JSON.parse(text)
  
  // 4. 检查业务状态码
  if (result.code !== 0) {
    throw new Error(result.message || 'Unknown error')
  }
  
  // 5. 返回数据
  return result.data
}
```

#### API 分组

```javascript
// 配置管理 API (12个)
fetchConfigs()              // GET /api/configs
fetchConfig(name)           // GET /api/configs/:name
createConfig(name, config)  // POST /api/configs
updateConfig(name, config)  // PUT /api/configs/:name
deleteConfig(name)          // DELETE /api/configs/:name
validateConfig(name)        // POST /api/configs/:name/validate
previewConfig(name)         // GET /api/configs/:name/preview

// 测试执行 API (5个)
runPrecheck(name)           // POST /api/configs/:name/precheck
runTest(name)               // POST /api/configs/:name/run
probeTest(name)             // POST /api/configs/:name/probe
collectReports(name)        // POST /api/configs/:name/collect
getReport(name)             // GET /api/configs/:name/report

// 字典管理 API (4个)
fetchHostnames()            // GET /api/dictionary/hostnames
updateHostnames(hostnames)  // PUT /api/dictionary/hostnames
fetchHCAs()                 // GET /api/dictionary/hcas
updateHCAs(hcas)            // PUT /api/dictionary/hcas
```

---

## 页面组件

### App.jsx - 主应用容器

**职责**：
- 全局状态管理（配置列表、当前配置、配置数据）
- Tab 导航管理
- 提供全局 Toast 通知
- 状态提升和回调传递

**核心状态**：

```javascript
const [configs, setConfigs] = useState([])              // 配置文件列表
const [currentConfig, setCurrentConfig] = useState(null) // 当前选中配置
const [configData, setConfigData] = useState(null)       // 当前配置数据
const [originalData, setOriginalData] = useState(null)   // 原始数据（用于取消）
const [loading, setLoading] = useState(false)            // 加载状态
```

**数据流**：

```
App.jsx (顶层状态)
  ├── loadConfigs() → fetchConfigs() → setConfigs()
  ├── selectConfig() → fetchConfig() → setConfigData()
  └── Props 向下传递
       ├── ConfigPage (配置、回调)
       ├── DictionaryPage (无依赖)
       └── TrafficTestPage (configs)
```

---

### ConfigPage.jsx - 配置管理页面

**职责**：
- 布局容器（左右分栏）
- 连接 ConfigList 和 ConfigEditor

**布局**：

```
┌─────────────────────────────────────────┐
│    ConfigPage                            │
├──────────────┬───────────────────────────┤
│              │                           │
│  ConfigList  │      ConfigEditor        │
│  (280px)     │      (flex: 1)           │
│              │                           │
│  • config1   │  ┌─────────────────────┐ │
│  • config2   │  │  基础配置           │ │
│  • config3   │  ├─────────────────────┤ │
│              │  │  服务端配置         │ │
│              │  ├─────────────────────┤ │
│              │  │  客户端配置         │ │
│              │  └─────────────────────┘ │
│              │                           │
└──────────────┴───────────────────────────┘
```

---

### DictionaryPage.jsx - 字典管理页面

**职责**：
- 管理主机名和 HCA 字典
- 支持批量文本编辑（每行一条）
- 自动去重和去空

**交互流程**：

```
用户编辑文本框
  ↓
点击"保存"按钮
  ↓
解析文本（按行分割、去空、去重）
  ↓
调用 updateHostnames() 或 updateHCAs()
  ↓
后端保存到 dictionary/*.txt
  ↓
重新加载字典
  ↓
Toast 提示成功
```

**UI 布局**：

```
┌─────────────────────────────────────────────────────┐
│  字典管理                                            │
├──────────────────────────┬──────────────────────────┤
│  主机名字典               │  HCA 字典                │
│  ┌──────────────────────┐│ ┌──────────────────────┐│
│  │ server1              ││ │ mlx5_0               ││
│  │ server2              ││ │ mlx5_1               ││
│  │ client1              ││ │ mlx5_2               ││
│  │ ...                  ││ │ ...                  ││
│  └──────────────────────┘│ └──────────────────────┘│
│  [保存主机名]             │  [保存 HCA 列表]         │
└──────────────────────────┴──────────────────────────┘
```

---

### TrafficTestPage.jsx - 流量测试页面

**职责**：
- 工作流引擎（5个步骤的编排）
- 实时状态展示（时间线 UI）
- 自动化测试执行
- 结果可视化

#### 工作流状态机

```javascript
const STEPS = {
  IDLE: 'idle',           // 待执行
  PRECHECK: 'precheck',   // 硬件检查
  RUN: 'run',             // 运行测试
  PROBE: 'probe',         // 探测状态
  COLLECT: 'collect',     // 收集报告
  REPORT: 'report',       // 生成报告
  COMPLETED: 'completed', // 完成
  ERROR: 'error',         // 错误
}
```

#### 执行流程

```
用户选择配置
  ↓
点击"开始测试"
  ↓
executeWorkflow()
  ├─> 步骤1: executePrecheckStep()
  │    ├─ 调用 runPrecheck()
  │    ├─ 检查健康状态
  │    └─ 显示 Precheck 结果表格
  │
  ├─> 步骤2: executeRunStep()
  │    ├─ 调用 runTest()
  │    └─ 分发脚本并启动进程
  │
  ├─> 步骤3: executeProbeStep()
  │    ├─ 循环调用 probeTest() (每2秒)
  │    ├─ 实时更新 ProbeResults
  │    └─ 直到 all_completed = true
  │
  ├─> 步骤4: executeCollectStep()
  │    ├─ 调用 collectReports()
  │    └─ 从远程主机收集报告文件
  │
  └─> 步骤5: executeReportStep()
       ├─ 调用 getReport()
       ├─ 生成性能分析报告
       └─ 显示 ReportResults
```

#### 时间线 UI 设计

```
步骤 1/5  ● PreCheck 检查           [成功]
          |
          ├─ 汇总信息：总计 8 | 健康 8 | 异常 0
          └─ 详细表格：主机名、HCA、状态...
          
步骤 2/5  ● 运行测试               [成功]
          |
          └─ 测试脚本已成功分发到所有节点
          
步骤 3/5  ◐ 探测状态               [执行中]
          |
          ├─ 实时统计：运行中 2 | 完成 2 | 错误 0
          └─ 主机详情表格（实时刷新）
          
步骤 4/5  ○ 收集报告               [待执行]
          
步骤 5/5  ○ 生成报告               [待执行]
```

#### 核心特性

1. **自动滚动** - 当前步骤自动滚动到视口中心
2. **实时更新** - Probe 步骤每 2 秒更新一次数据
3. **错误处理** - 任何步骤失败都会停止并显示错误
4. **状态持久** - 所有步骤结果保留，可随时查看
5. **配置预览** - 开始前可预览配置详情

---

## UI 组件

### ConfigList.jsx - 配置文件列表

**职责**：
- 显示所有配置文件
- 支持创建、选择、删除配置
- 区分默认配置（⭐标记）

**交互**：

```javascript
// 创建配置
onOpen() → 输入文件名 → createConfig() → onRefresh()

// 选择配置
onClick(name) → onSelect(name) → App.selectConfig()

// 删除配置
onClick(delete) → confirm() → deleteConfig() → onRefresh()
```

---

### ConfigEditor.jsx - 配置文件编辑器

**职责**：
- 表单化编辑配置字段
- 集成字典（主机名、HCA）快速选择
- 支持验证、预览、保存

**核心功能**：

1. **字段编辑**
   - 基础配置：端口、流类型、QP数量、消息大小等
   - 服务端/客户端：主机名列表、HCA 列表
   - 高级配置：RDMA CM、GID Index、报告设置等
   - SSH 配置：用户名、私钥路径
   - Logger 配置：日志级别、日志格式

2. **标签管理**（主机名、HCA）
   - 从字典快速选择（下拉菜单）
   - 手动输入添加
   - 点击 × 删除
   - 自动去重

3. **操作按钮**
   - **验证** - 调用 `validateConfig()`，显示验证错误列表
   - **预览** - 调用 `previewConfig()`，显示 YAML 格式
   - **保存** - 调用 `updateConfig()`，提交修改
   - **取消** - 恢复 `originalData`

**表单布局**：

```
┌────────────────────────────────────────────────┐
│  配置编辑器 - config.yaml                       │
├────────────────────────────────────────────────┤
│  [验证] [预览] [保存] [取消]                    │
├────────────────────────────────────────────────┤
│                                                 │
│  基础配置                                       │
│  ├─ 起始端口: [20000]                          │
│  ├─ 流类型: [InCast ▼]                         │
│  ├─ 理论速度: [400] Gbps                       │
│  └─ ...                                         │
│                                                 │
│  服务端配置                                     │
│  ├─ 主机名: [server1] [server2] [+添加]        │
│  └─ HCA: [mlx5_0] [mlx5_1] [+添加]             │
│                                                 │
│  客户端配置                                     │
│  ├─ 主机名: [client1] [client2] [+添加]        │
│  └─ HCA: [mlx5_0] [+添加]                      │
│                                                 │
│  高级配置                                       │
│  ├─ RDMA CM: [✓ 启用]                          │
│  └─ ...                                         │
│                                                 │
└────────────────────────────────────────────────┘
```

---

### ProbeResults.jsx - 探测结果展示

**职责**：
- 显示测试进程状态
- 实时刷新（TrafficTestPage 每 2 秒调用）

**数据结构**：

```javascript
{
  running_hosts: 2,
  completed_hosts: 2,
  error_hosts: 0,
  total_processes: 40,
  all_completed: false,
  timestamp: "2025-11-05T10:30:00Z",
  results: [
    {
      hostname: "server1",
      process_count: 10,
      status: "RUNNING",
      error: ""
    }
  ]
}
```

**UI 元素**：
- 汇总统计卡片（运行中、已完成、错误、总进程）
- 主机详情表格（主机名、进程数、状态、错误）
- 状态徽章（运行中 🔵、已完成 ✅、错误 ❌）

---

### ReportResults.jsx - 性能报告展示

**职责**：
- 展示性能测试结果
- 支持两种模式：P2P / FullMesh-InCast

#### FullMesh/InCast 模式

**汇总信息**：
- 服务端总带宽
- 客户端数量
- 理论单客户端带宽

**客户端表格（TX）**：

| 主机名 | 设备 | 实际带宽 | 理论带宽 | 差值 | 差值% | 状态 |
|--------|------|----------|----------|------|-------|------|
| client1 | mlx5_0 | 95.5 | 100 | -4.5 | -4.5% | ✅ OK |

**服务端表格（RX）**：

| 主机名 | 设备 | 接收带宽 | 理论带宽 | 差值 | 差值% | 状态 |
|--------|------|----------|----------|------|-------|------|
| server1 | mlx5_0 | 391.19 | 400 | -8.81 | -2.2% | ✅ OK |

**状态判定逻辑**：
- `|delta_percent| <= 20%` → ✅ OK（绿色）
- `|delta_percent| > 20%` → ❌ NOT OK（红色）

#### P2P 模式

**汇总信息**：
- 总连接对数
- 平均速度

**详细表格**：

| 主机名 | 设备 | 平均速度 | 连接数 |
|--------|------|----------|--------|
| host1 | mlx5_0 | 195.5 Gbps | 10 |

---

## 状态管理

### 状态提升策略

xnetperf Web UI 使用 **Props Drilling** 模式进行状态管理，适合中小型应用。

```
App.jsx (顶层状态)
  ├─ configs (配置列表)
  ├─ currentConfig (当前配置)
  ├─ configData (配置数据)
  ├─ originalData (原始数据)
  └─ loading (加载状态)
       ↓ Props
┌──────┴──────┬──────────────┬──────────────┐
│             │              │              │
ConfigPage    DictionaryPage TrafficTestPage
│             (独立状态)      (独立状态)
├─ ConfigList
└─ ConfigEditor
```

### 局部状态 vs 全局状态

| 状态 | 层级 | 说明 |
|------|------|------|
| `configs` | 全局 (App) | 所有页面共享 |
| `currentConfig` | 全局 (App) | ConfigPage 和 TrafficTestPage 共享 |
| `configData` | 全局 (App) | ConfigEditor 需要 |
| `hostnameDict` | 局部 (ConfigEditor) | 仅编辑器内部使用 |
| `hcaDict` | 局部 (ConfigEditor) | 仅编辑器内部使用 |
| `probeData` | 局部 (TrafficTestPage) | 仅流量测试使用 |
| `reportData` | 局部 (TrafficTestPage) | 仅流量测试使用 |

### 数据流向

```
用户操作
  ↓
组件事件处理器
  ↓
调用 api.js 函数
  ↓
发送 HTTP 请求
  ↓
后端处理
  ↓
返回 JSON 响应
  ↓
api.js 解包 data
  ↓
更新组件状态 (setState)
  ↓
React 重新渲染
  ↓
UI 更新
```

---

## 工作流设计

### 配置管理工作流

```
1. 用户访问 Web UI
   ↓
2. App.jsx 加载配置列表
   ├─ useEffect(() => loadConfigs(), [])
   ├─ fetchConfigs() → GET /api/configs
   └─ setConfigs(data)
   ↓
3. 用户选择配置
   ├─ ConfigList.onClick(name)
   ├─ onSelect(name) → selectConfig(name)
   ├─ fetchConfig(name) → GET /api/configs/:name
   ├─ setConfigData(data)
   └─ setOriginalData(clone(data))
   ↓
4. 用户编辑配置
   ├─ ConfigEditor 表单编辑
   ├─ onChange(newData) → setConfigData(newData)
   └─ 本地状态更新，未保存
   ↓
5. 用户保存配置
   ├─ ConfigEditor.handleSave()
   ├─ updateConfig(name, data) → PUT /api/configs/:name
   ├─ onSave() → loadConfigs() & selectConfig()
   └─ Toast 提示成功
```

---

### 流量测试工作流

```
用户选择配置 → 预览配置（可选） → 开始测试
                                   ↓
┌──────────────────────────────────────────────────────┐
│                  executeWorkflow()                    │
├──────────────────────────────────────────────────────┤
│                                                       │
│  步骤 1: executePrecheckStep()                        │
│    ├─ setCurrentStep(PRECHECK)                       │
│    ├─ updateStepStatus(PRECHECK, 'running')          │
│    ├─ runPrecheck() → POST /api/configs/:name/precheck│
│    ├─ setPrecheckData(data)                          │
│    ├─ 检查健康状态                                    │
│    ├─ updateStepStatus(PRECHECK, 'success')          │
│    └─ return true (继续) / false (停止)              │
│                                                       │
│  步骤 2: executeRunStep()                             │
│    ├─ setCurrentStep(RUN)                            │
│    ├─ runTest() → POST /api/configs/:name/run        │
│    └─ return true / false                            │
│                                                       │
│  步骤 3: executeProbeStep()                           │
│    ├─ setCurrentStep(PROBE)                          │
│    ├─ while (!allCompleted && probeCount < 300) {    │
│    │    ├─ probeTest() → POST /api/configs/:name/probe│
│    │    ├─ setProbeData(data) (实时更新 UI)          │
│    │    ├─ 检查 all_completed                        │
│    │    └─ await sleep(2000) (等待 2 秒)             │
│    │  }                                               │
│    └─ return true / false                            │
│                                                       │
│  步骤 4: executeCollectStep()                         │
│    ├─ setCurrentStep(COLLECT)                        │
│    ├─ collectReports() → POST /api/configs/:name/collect│
│    └─ return true / false                            │
│                                                       │
│  步骤 5: executeReportStep()                          │
│    ├─ setCurrentStep(REPORT)                         │
│    ├─ getReport() → GET /api/configs/:name/report    │
│    ├─ setReportData(data)                            │
│    ├─ setCurrentStep(COMPLETED)                      │
│    └─ Toast("流量测试完成")                           │
│                                                       │
└──────────────────────────────────────────────────────┘
```

**关键特性**：
1. **串行执行** - 每个步骤依次执行，前一步失败则停止
2. **实时反馈** - Probe 步骤实时刷新数据
3. **自动滚动** - 当前步骤自动滚动到视口
4. **错误处理** - 任何步骤失败都会设置错误状态
5. **状态保留** - 所有步骤的数据和状态都保留

---

## 数据流

### 完整数据流示例：保存配置

```
┌─────────────────────────────────────────────────────┐
│  1. 用户操作                                         │
│     ConfigEditor 点击"保存"按钮                      │
└────────────────┬────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────┐
│  2. 组件事件处理                                     │
│     handleSave() {                                   │
│       setSaving(true)                                │
│       await updateConfig(currentConfig, configData)  │
│       setSaving(false)                               │
│     }                                                 │
└────────────────┬────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────┐
│  3. API 层调用                                       │
│     updateConfig(name, config) {                     │
│       return await fetchJSON(                        │
│         `/api/configs/${name}`,                      │
│         {                                            │
│           method: 'PUT',                             │
│           headers: { 'Content-Type': 'application/json' },│
│           body: JSON.stringify(config)               │
│         }                                            │
│       )                                              │
│     }                                                 │
└────────────────┬────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────┐
│  4. HTTP 请求                                        │
│     PUT /api/configs/config.yaml                     │
│     Content-Type: application/json                   │
│     Body: { start_port: 20000, ... }                 │
└────────────────┬────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────┐
│  5. 后端处理                                         │
│     server/config_service.go                         │
│     ├─ 接收请求                                      │
│     ├─ 解析 JSON                                     │
│     ├─ 验证配置                                      │
│     ├─ 保存到文件                                    │
│     └─ 返回响应                                      │
└────────────────┬────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────┐
│  6. HTTP 响应                                        │
│     {                                                │
│       "code": 0,                                     │
│       "message": "success",                          │
│       "data": null                                   │
│     }                                                │
└────────────────┬────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────┐
│  7. API 层解包                                       │
│     fetchJSON() 检查 code === 0                      │
│     返回 result.data (null)                          │
└────────────────┬────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────┐
│  8. 组件状态更新                                     │
│     toast({ title: '保存成功！' })                   │
│     onSave()                                         │
│       ├─ loadConfigs() (刷新列表)                   │
│       └─ selectConfig(currentConfig) (重新加载)      │
└────────────────┬────────────────────────────────────┘
                 ↓
┌─────────────────────────────────────────────────────┐
│  9. UI 更新                                          │
│     ├─ 配置列表刷新                                  │
│     ├─ 配置数据重新加载                              │
│     └─ Toast 提示显示                                │
└─────────────────────────────────────────────────────┘
```

---

## 最佳实践

### 1. API 调用封装

✅ **好的做法**：
```javascript
// api.js 统一封装
export async function updateConfig(name, config) {
  return await fetchJSON(`${API_BASE}/configs/${name}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  })
}

// 组件中使用
const handleSave = async () => {
  await updateConfig(currentConfig, configData)
}
```

❌ **不好的做法**：
```javascript
// 组件中直接调用 fetch
const handleSave = async () => {
  const response = await fetch(`/api/configs/${currentConfig}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(configData),
  })
  const result = await response.json()
  // 重复的错误处理代码...
}
```

---

### 2. 错误处理

✅ **好的做法**：
```javascript
try {
  await updateConfig(name, config)
  toast({ title: '保存成功！', status: 'success' })
} catch (error) {
  toast({
    title: '保存失败',
    description: error.message,  // fetchJSON 已处理错误
    status: 'error',
  })
}
```

---

### 3. 状态管理

✅ **好的做法**：
```javascript
// 原始数据备份，支持取消
const [originalData, setOriginalData] = useState(null)
const [configData, setConfigData] = useState(null)

// 选择配置时同时保存原始数据
const selectConfig = async (name) => {
  const data = await fetchConfig(name)
  setConfigData(data)
  setOriginalData(JSON.parse(JSON.stringify(data)))  // 深拷贝
}

// 取消修改
const handleCancel = () => {
  setConfigData(JSON.parse(JSON.stringify(originalData)))
}
```

---

### 4. 实时更新

✅ **好的做法**（Probe 步骤）：
```javascript
while (!allCompleted && probeCount < maxProbes) {
  const data = await probeTest(selectedConfig)
  setProbeData(data)  // 实时更新 UI
  
  if (!data.all_completed) {
    await new Promise(resolve => setTimeout(resolve, 2000))  // 等待 2 秒
    probeCount++
  }
}
```

---

### 5. 组件职责分离

✅ **好的做法**：
```javascript
// ProbeResults.jsx - 纯展示组件
function ProbeResults({ data }) {
  // 只负责渲染，不处理业务逻辑
  return <Table>...</Table>
}

// TrafficTestPage.jsx - 容器组件
function TrafficTestPage() {
  const [probeData, setProbeData] = useState(null)
  
  const executeProbeStep = async () => {
    const data = await probeTest(selectedConfig)
    setProbeData(data)  // 数据处理在这里
  }
  
  return <ProbeResults data={probeData} />
}
```

---

### 6. Toast 通知使用

✅ **统一的 Toast 模式**：
```javascript
// 成功
toast({
  title: '操作成功！',
  status: 'success',
  duration: 2000,
})

// 错误
toast({
  title: '操作失败',
  description: error.message,
  status: 'error',
  duration: 3000,
})

// 警告
toast({
  title: 'PreCheck 发现问题',
  description: '请检查结果',
  status: 'warning',
  duration: 3000,
})
```

---

### 7. 加载状态处理

✅ **好的做法**：
```javascript
const [loading, setLoading] = useState(false)

const handleSave = async () => {
  try {
    setLoading(true)
    await updateConfig(name, config)
    toast({ title: '保存成功！' })
  } catch (error) {
    toast({ title: '保存失败', description: error.message })
  } finally {
    setLoading(false)  // 确保 loading 状态恢复
  }
}

return (
  <Button onClick={handleSave} isLoading={loading}>
    保存
  </Button>
)
```

---

## 总结

### 设计亮点

1. **清晰的分层架构** - API 层、页面层、组件层职责分明
2. **统一的 API 封装** - `api.js` 统一处理所有 HTTP 通信
3. **智能的工作流引擎** - TrafficTestPage 实现完整的自动化测试流程
4. **实时的状态反馈** - Probe 步骤实时刷新，用户体验流畅
5. **灵活的状态管理** - Props Drilling 适合中小型应用，易于理解
6. **优雅的错误处理** - 统一的错误处理机制，用户友好的提示
7. **响应式的 UI 设计** - Chakra UI 提供统一的设计系统

### 技术选型优势

- **React 18** - 现代化、生态丰富
- **Chakra UI** - 开箱即用的组件、无需编写 CSS
- **Vite** - 极快的开发体验和构建速度
- **Fetch API** - 原生支持、无需额外依赖

### 扩展性考虑

如果未来需要扩展，可以考虑：
- **React Router** - 多页面路由（如果页面数量增加）
- **Redux / Zustand** - 全局状态管理（如果状态复杂度增加）
- **React Query** - 服务端状态管理和缓存（如果 API 调用频繁）
- **WebSocket** - 实时通信（如果需要服务端主动推送）

---

## 相关文档

- [API 参考文档](api-reference.md) - 完整的 HTTP API 文档
- [Web UI 快速开始](web-ui-quickstart.md) - 用户使用指南
- [流量测试指南](traffic-test-guide.md) - 测试流程说明

---

**最后更新**: 2025-11-05  
**维护者**: xnetperf Team
