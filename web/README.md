# Zero Service Web 前端项目

Zero Service 的 Web 前端项目集合，包含多个子系统和一个统一的工作台。

## 项目结构

```
web/
├── live/                    # Live 视频会议系统
├── socketio/                # SocketIO 网关测试工具
├── workspace/               # 统一工作台
├── openapi/                 # OpenAPI 测试控制台
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

统一入口、运行状态与端点速查面板。

**功能特性：**
- 状态总览：在线模块、平均响应、检测倒计时与响应趋势
- 五个入口：Live 会议、SocketIO 网关、OpenAPI 控制台、GitLab、GitHub
- 实时健康检测（15 秒自动 + 手动刷新，页面隐藏时暂停）
- 端点速查、检测记录、运行环境与构建信息
- `⌘K` 命令面板、深浅主题、移动端抽屉导航
- 响应式设计

**技术栈：**
- React 18 + TypeScript
- Lucide React
- Vite

**端口：** 5180

**文档：** [工作台文档](./workspace/README.md)

---

### 4. OpenAPI 测试控制台 (`openapi/`)

`app/opengtw` 的配套调试控制台，仅限开发/测试环境。

**功能特性：**
- API 调试（19 个 method：Podman 4 + Modbus 15、method 分组搜索、写操作提示、JSON 编辑器、响应高亮）
- Basic 凭据与 Base URL 管理
- 应用创建、启停、密钥重置（JWT 管理面）
- 最近 50 条请求历史与回填重放
- 错误码速查与处理建议

**技术栈：**
- React 18 + TypeScript
- Vite
- Lucide React

**端口：** 5181

**生产路径：** `/openapi/`

**文档：** [OpenAPI 控制台文档](./openapi/README.md)

---

> Live 与 SocketIO 支持在浏览器内本地签发测试 JWT（HS256），无需先启动 livegtw 即可生成 Token；实际会议与网关连接仍需对应后端服务。

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

# 4. 启动 OpenAPI 测试控制台
cd web/openapi
npm install
npm run dev
```

### 访问地址

- **工作台：** http://127.0.0.1:5180
- **Live 视频会议：** http://127.0.0.1:5178
- **SocketIO 网关测试：** http://127.0.0.1:5179
- **OpenAPI 测试控制台：** http://127.0.0.1:5181

生产环境由 nginx 统一托管：Live `/`、SocketIO `:8090/`、工作台 `:8091/`、OpenAPI `/openapi/`。

> 开发服务器统一监听 `127.0.0.1`，避免 `localhost` 在某些环境解析到 IPv6 后连接失败。

---

## 开发指南

### 添加新子系统

1. 在 `web/` 目录下创建新的项目目录
2. 使用 Vite + React + TypeScript 初始化项目
3. 在 `workspace/src/data/modules.ts` 的 `modules` 数组中添加入口配置，并在 nginx / vite 中补上 `/health/<id>` 代理
4. 更新 `web/start.sh` 启动脚本

### 项目配置

每个子系统都是独立的 Vite 项目，可以独立开发和部署。

**共同配置：**
- TypeScript 严格模式
- React 18
- Vite 6.x

### 环境变量

各子系统支持环境变量配置：

**工作台：**
- `VITE_LIVE_URL`：Live 子系统访问地址
- `VITE_SOCKETIO_URL`：SocketIO 子系统访问地址
- `VITE_OPENAPI_URL`：OpenAPI 控制台访问地址（生产留空时按 Live 站点 + `/openapi/` 推导）
- `VITE_GITLAB_URL`：GitLab 访问地址（留空使用 `https://gitlab.localhost/`）
- `VITE_GITHUB_URL`：GitHub 仓库地址（留空使用 `https://github.com/maomao94/zero-service`）

**Live 视频会议：**
- `VITE_LIVEKIT_URL`：LiveKit 服务器 URL
- `VITE_SOCKET_URL`：SocketIO 服务器 URL
- `VITE_JWT_SECRET`：本地生成测试 Token 的 HS256 密钥（默认取 livegtw 开发配置）

**SocketIO 网关测试：**
- `VITE_SOCKET_URL`：SocketIO 服务器 URL
- `VITE_JWT_SECRET`：本地生成测试 Token 的 HS256 密钥（默认取 livegtw 开发配置）

**OpenAPI 测试控制台：**
- `VITE_OPENAPI_BASE_URL`：默认 Base URL，留空表示同源；页面内仍可覆盖

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
cd web/openapi && npm run build

# 或统一构建到 deploy/nginx/dist
bash deploy/nginx/build-web.sh

# 构建产物位于 deploy/nginx/dist
```

### nginx 部署（推荐）

四个前端统一由 `deploy/nginx` 的 `web-gateway` 容器托管：

```bash
bash deploy/nginx/build-web.sh   # 构建四个项目并汇总到 deploy/nginx/dist
bash deploy/nginx/deploy.sh      # 启动 / 更新 nginx 容器
```

访问入口、端口与反向代理说明见 [`deploy/nginx/README.md`](../deploy/nginx/README.md)。

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

### Q: 生产环境 SocketIO 连接失败？

A: 生产构建留空 `VITE_SOCKET_URL`，页面会使用当前站点 `:8090` 的 `/socket.io/`，由 nginx 反代到 socketgtw；不要在页面填写 `11003`。

### Q: 本地生成 Token 提示 WebCrypto 不可用？

A: WebCrypto 只在 localhost / https 等安全上下文可用；通过内网 IP 访问时请使用 HTTPS，或直接粘贴后端签发的 Token。

---

## 交付记录

四个项目的当前状态、验证结果与已知限制见 [`COMPLETION.md`](./COMPLETION.md)。

## 贡献指南

1. 遵循现有代码风格
2. 使用 TypeScript 严格模式
3. 添加必要的类型定义
4. 编写清晰的注释和文档
5. 测试新功能后再提交

---

## 许可证

内部项目，仅限内部使用。
