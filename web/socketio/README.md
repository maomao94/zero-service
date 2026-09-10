# SocketIO 网关测试工具

SocketIO 消息网关测试前端工具，连接身份（用户域/设备域）由 Token claims 决定。

## 功能特性

- **连接管理**：支持配置服务器 URL、Token 认证
- **双域支持**：连接身份由 Token claims 自动识别（用户域 `user-id` / 设备域 `device-id`）
- **Claims 解析**：实时解析并展示 Token claims，便于验证 user-name / device-name / dept-code 等声明
- **消息收发**：支持上行消息、房间广播、全局广播
- **房间管理**：支持加入/离开房间、查询房间列表
- **事件监听**：支持动态添加/移除自定义事件监听器
- **日志系统**：实时显示连接状态、消息收发日志，支持过滤和自动滚动

## 快速开始

### 安装依赖

```bash
cd web/socketio
npm install
```

### 启动开发服务器

```bash
npm run dev
```

将在 http://localhost:5179 启动

### 构建生产版本

```bash
npm run build
```

## 使用说明

### 1. 连接配置

1. **服务器 URL**：输入 SocketIO 服务器地址，默认 `http://127.0.0.1:11003`
2. **Authorization Token**：粘贴认证 Token（由 live 子系统 `POST /live/v1/generateToken` 签发），用户域/设备域身份由 Token claims 决定

粘贴 Token 后，可在「Token Claims 解析」中查看 `user-id` / `user-name` / `dept-code`（用户域）或 `device-id` / `device-name`（设备域）等声明。

### 2. 消息发送

- **上行消息**：发送 `__up__` 事件到服务器
- **房间广播**：向指定房间广播消息
- **全局广播**：向所有连接广播消息

### 3. 房间管理

- **加入房间**：加入指定房间
- **离开房间**：离开指定房间
- **查询房间**：查询当前会话已加入的房间列表

### 4. 事件监听

- 自动监听系统事件：`connect`、`disconnect`、`__down__`、`__stat_down__`
- 支持动态添加自定义事件监听器
- 支持移除已注册的事件监听器

## 用户域 vs 设备域

### 用户域（User Domain）

- Token claims 包含 `user-id`
- 适用于用户登录后的会话
- 可通过 `GetSessionByUserId()` 查询会话
- 典型场景：Web 应用、移动端应用

### 设备域（Device Domain）

- Token claims 包含 `device-id`
- 适用于 IoT 设备、嵌入式设备
- 可通过 `GetSessionByDeviceId()` 查询会话
- 典型场景：传感器、监控设备、智能硬件

## 技术栈

- React 18
- TypeScript
- Vite
- Socket.IO Client 4.x
- Lucide React (图标库)

## 项目结构

```
web/socketio/
├── src/
│   ├── App.tsx          # 主应用组件
│   ├── main.tsx         # 入口文件
│   ├── styles.css       # 样式文件
│   ├── lib/
│   │   └── token.ts     # JWT claims 解析
│   └── vite-env.d.ts    # 类型定义
├── index.html           # HTML 入口
├── package.json         # 项目配置
├── tsconfig.json        # TypeScript 配置
└── vite.config.ts       # Vite 配置
```

## 环境变量

- `VITE_SOCKET_URL`：SocketIO 服务器地址（开发环境默认 `http://127.0.0.1:11003`）

## 相关文档

- [SocketIO Server 文档](../../docs/agent/standards/networking/socketio.md)
- [SocketIO 源码](../../common/socketiox/)
