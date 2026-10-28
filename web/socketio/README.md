# SocketIO 网关测试工具

SocketIO 消息网关测试前端工具，连接身份（用户域/设备域）由 Token claims 决定。

## 功能特性

- **连接管理**：支持配置服务器 URL、Token 认证
- **双域支持**：连接身份由 Token claims 自动识别（`auth-type` 为 `user` / `device`，身份统一使用 `user-id` / `user-name`）
- **Claims 解析**：实时解析并展示 Token claims，便于验证 `user-id` / `user-name` / `dept-code` / `auth-type` 等声明
- **本地签发 Token**：在配置页直接生成用户/设备 JWT（HS256），不依赖 live 子系统或后端服务
- **消息收发**：支持上行消息、房间广播、全局广播
- **房间管理**：支持加入/离开房间、查询房间列表
- **事件监听**：支持动态添加/移除自定义事件监听器
- **房间列表**：查询结果解析为房间标签，点击即可填入房间名
- **接收统计**：按事件名累计接收次数与最近接收时间
- **日志系统**：底部控制台实时显示连接、消息、ACK 与错误，支持类型过滤、自动滚动、拖拽调高、导出、清空与详情复制

## 界面结构

```
┌ 顶栏：品牌 / 网关地址 / 连接状态 / 连接与断开 ─────────────┐
├ 侧栏：连接配置 · 消息发送 · 房间管理 · 事件监听 ┬ 工作区 ─────┤
│  当前会话：Socket ID / Token claims / 事件数     │ 卡片式表单  │
├ 底部控制台（可拖拽、可收起）────────────────────────────────┤
└──────────────────────────────────────────────────────────────┘
```

- 桌面端为固定侧栏 + 独立滚动工作区 + 底部控制台；窄屏侧栏收为横向标签栏。
- `Ctrl / ⌘ + Enter` 发送上行消息；控制台日志可展开查看 JSON 详情并复制。

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

将在 http://127.0.0.1:5179 启动

### 构建生产版本

```bash
npm run build
```

生产环境由 nginx 的 8090 站点托管静态页面，并把 `/socket.io/` 反代到 socketgtw。前端保持 `VITE_SOCKET_URL` 留空即可走同源，不需要在页面填写 `11003`。

## 使用说明

### 1. 连接配置

1. **服务器 URL**：开发态默认 `http://127.0.0.1:11003`；生产构建留空时使用当前站点（nginx 8090）的 `/socket.io/`
2. **Authorization Token**：可直接粘贴，或在「本地签发 Token」中生成。本地生成使用 HS256，默认密钥与 `app/livegtw/etc/livegtw.yaml` 的 `JwtAuth.AccessSecret` 一致，可用 `VITE_JWT_SECRET` 覆盖。

粘贴 Token 后，可在「Token Claims 解析」中查看 `user-id` / `user-name` / `dept-code` / `auth-type`；设备域同样复用 `user-id` / `user-name`，以 `auth-type: device` 区分。

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

- Token claims 包含 `user-id`（设备 ID）且 `auth-type` 为 `device`
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

- [SocketIO 源码](../../common/socketiox/server.go)
