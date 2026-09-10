# Zero Service Web 前端项目

Zero Service 的 Web 前端项目集合，包含多个子系统和一个统一的工作台。

## 项目结构

```
web/
├── live/                    # Live 视频会议系统
├── socketio/                # SocketIO 网关测试工具
├── workspace/               # 统一工作台
└── start.sh                 # 一键启动脚本
```

## 子系统说明

### 1. Live 视频会议 (`live/`)

实时视频会议系统，基于 LiveKit 构建。

**功能特性：**
- 多人视频通话
- 屏幕共享
- 会议管理（创建、加入、结束）
- SIP 电话集成
- 会议邀请和通知
- 聊天和成员管理

**技术栈：**
- React 18 + TypeScript
- LiveKit Client SDK
- Socket.IO Client
- Vite

**端口：** 5178

**文档：** [Live 视频会议文档](../docs/agent/domains/livekit/web.md)

---

### 2. SocketIO 网关测试 (`socketio/`)

SocketIO 消息网关测试工具，支持用户域和设备域连接。

**功能特性：**
- 连接管理（服务器 URL、Token 认证）
- Token Claims 解析展示
- 用户域和设备域双模式支持
- 消息收发（上行、房间广播、全局广播）
- 房间管理（加入、离开、查询）
- 事件监听（动态添加/移除）
- 日志系统（过滤、自动滚动）

**技术栈：**
- React 18 + TypeScript
- Socket.IO Client 4.x
- Vite

**端口：** 5179

**文档：** [SocketIO 网关测试文档](./socketio/README.md)

---

### 3. 统一工作台 (`workspace/`)

统一管理和访问各个子系统的开发工作台。

**功能特性：**
- 统一入口访问所有子系统
- 实时健康检测系统状态（自动定时检测 + 手动刷新）
- 快速打开和切换子系统
- 响应式设计

**技术栈：**
- React 18 + TypeScript
- Lucide React
- Vite

**端口：** 5180

**文档：** [工作台文档](./workspace/README.md)

---

## 快速开始

### 方式一：一键启动（推荐）

```bash
cd web
./start.sh
```

这将自动安装依赖并启动所有项目。

### 方式二：手动启动

分别在不同的终端中启动各个项目：

```bash
# 1. 启动 Live 视频会议
cd web/live
npm install
npm run dev

# 2. 启动 SocketIO 网关测试
cd web/socketio
npm install
npm run dev

# 3. 启动工作台
cd web/workspace
npm install
npm run dev
```

### 访问地址

- **工作台：** http://localhost:5180
- **Live 视频会议：** http://localhost:5178
- **SocketIO 网关测试：** http://localhost:5179

---

## 开发指南

### 添加新子系统

1. 在 `web/` 目录下创建新的项目目录
2. 使用 Vite + React + TypeScript 初始化项目
3. 在 `workspace/src/App.tsx` 的 `subSystems` 数组中添加新系统配置
4. 更新 `web/start.sh` 启动脚本

### 项目配置

每个子系统都是独立的 Vite 项目，可以独立开发和部署。

**共同配置：**
- TypeScript 严格模式
- React 18
- Vite 6.x

### 环境变量

各子系统支持环境变量配置：

**Live 视频会议：**
- `VITE_LIVEKIT_URL`：LiveKit 服务器 URL
- `VITE_SOCKET_URL`：SocketIO 服务器 URL

**SocketIO 网关测试：**
- `VITE_SOCKET_URL`：SocketIO 服务器 URL

---

## 部署

### 开发环境

使用 `npm run dev` 启动开发服务器，支持热更新。

### 生产环境

```bash
# 构建所有项目
cd web/live && npm run build
cd web/socketio && npm run build
cd web/workspace && npm run build

# 部署 dist 目录到 Web 服务器
```

### Docker 部署（可选）

可以为每个子系统创建 Dockerfile，使用 nginx 提供静态文件服务。

---

## 相关文档

- [Agent 知识库](../docs/agent/README.md)
- [SocketIO Server 文档](../docs/agent/standards/networking/socketio.md)
- [LiveKit Web 前端文档](../docs/agent/domains/livekit/web.md)
- [领域上下文地图](../CONTEXT-MAP.md)

---

## 常见问题

### Q: 如何修改端口？

A: 编辑对应项目的 `vite.config.ts` 文件，修改 `server.port` 配置。

### Q: 如何添加新的事件监听？

A: 在 SocketIO 测试工具中，使用"事件监听"区域的"添加监听"功能。

### Q: 用户域和设备域有什么区别？

A: 用户域使用 userId 标识，适用于用户会话；设备域使用 deviceId 标识，适用于 IoT 设备。

### Q: 如何调试 SocketIO 连接？

A: 使用 SocketIO 测试工具的日志功能，可以实时查看连接状态和消息收发。

---

## 贡献指南

1. 遵循现有代码风格
2. 使用 TypeScript 严格模式
3. 添加必要的类型定义
4. 编写清晰的注释和文档
5. 测试新功能后再提交

---

## 许可证

内部项目，仅限内部使用。
