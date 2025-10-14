# xnetperf Web UI

基于 React + Chakra UI 的配置管理界面。

## 开发

### 前提条件
- Node.js 18+ 
- npm 或 yarn

### 安装依赖
```bash
cd web
npm install
```

### 开发模式
```bash
npm run dev
```
这会启动 Vite 开发服务器（默认端口 5173），并自动代理 API 请求到后端服务器（localhost:8080）。

### 构建生产版本
```bash
npm run build
```
构建后的文件会输出到 `web/static/` 目录，可以被 Go 的 embed 功能嵌入到二进制文件中。

## 项目结构

```
web/
├── src/                    # React 源代码
│   ├── components/         # React 组件
│   │   ├── ConfigList.jsx  # 配置文件列表
│   │   └── ConfigEditor.jsx # 配置编辑器
│   ├── App.jsx             # 主应用组件
│   ├── api.js              # API 调用封装
│   └── main.jsx            # 入口文件
├── static/                 # 构建输出目录（由 npm run build 生成）
├── index.html              # HTML 模板
├── vite.config.js          # Vite 配置
└── package.json            # 项目依赖

```

## 技术栈

- **React 18**: UI 框架
- **Chakra UI**: 组件库，提供现代化的 UI 组件
- **Vite**: 构建工具，快速的开发服务器和优化的生产构建
- **Emotion**: CSS-in-JS，Chakra UI 的样式引擎

## 功能特性

- ✅ 配置文件列表展示
- ✅ 新建/删除配置文件
- ✅ 配置文件编辑（表单方式）
- ✅ 配置验证
- ✅ 响应式设计
- ✅ 友好的用户交互和提示
