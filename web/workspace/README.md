# Zero 工作台

统一管理和访问 Zero Service 各个子系统的开发工作台。

## 功能特性

- 统一入口访问所有子系统
- 实时健康检测系统状态（每 15 秒自动检测，支持手动刷新）
- 快速打开和切换子系统
- 响应式设计，支持多种屏幕尺寸

## 包含的子系统

### 1. Live 视频会议
- 端口：5178
- 功能：实时视频会议、屏幕共享、会议管理、SIP 电话
- 路径：`/web/live`

### 2. SocketIO 网关测试
- 端口：5179
- 功能：SocketIO 消息网关测试、用户域/设备域连接、房间管理
- 路径：`/web/socketio`

## 快速开始

### 1. 启动工作台

```bash
cd web/workspace
npm install
npm run dev
```

工作台将在 http://localhost:5180 启动

### 2. 启动子系统

分别在不同的终端中启动各个子系统：

```bash
# 启动 Live 视频会议
cd web/live
npm install
npm run dev

# 启动 SocketIO 网关测试
cd web/socketio
npm install
npm run dev
```

### 3. 访问系统

- 工作台：http://localhost:5180
- Live 视频会议：http://localhost:5178
- SocketIO 网关测试：http://localhost:5179

## 开发说明

### 项目结构

```
web/workspace/
├── src/
│   ├── App.tsx          # 主应用组件
│   ├── main.tsx         # 入口文件
│   ├── styles.css       # 样式文件
│   └── vite-env.d.ts    # 类型定义
├── index.html           # HTML 入口
├── package.json         # 项目配置
├── tsconfig.json        # TypeScript 配置
└── vite.config.ts       # Vite 配置
```

### 添加新子系统

1. 在 `src/App.tsx` 的 `subSystems` 数组中添加新系统配置
2. 提供系统名称、描述、图标、端口等信息
3. 确保新系统已在对应端口运行

## 技术栈

- React 18
- TypeScript
- Vite
- Lucide React (图标库)
