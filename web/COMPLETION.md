# Web 前端交付说明

Zero Service 的 Web 前端集合，包含统一工作台与三个子系统。四个项目独立开发、统一由 nginx 托管，开发态使用 Vite 端口。

## 组成

| 项目 | 目录 | 开发端口 | 生产入口 | 说明 |
| --- | --- | --- | --- | --- |
| Zero 工作台 | `workspace/` | 5180 | `:8091/` | 统一入口、健康检测、端点速查、GitHub、命令面板 |
| Live 会议 | `live/` | 5178 | `/`（80/443） | 会议大厅、会议室、录制、邀请、本地 Token |
| SocketIO 测试 | `socketio/` | 5179 | `:8090/` | 连接、消息、房间、事件、控制台、本地 Token |
| OpenAPI 控制台 | `openapi/` | 5181 | `/openapi/` | 开放 API 调试、应用管理、错误码速查 |

生产入口与反向代理见 [`deploy/nginx/README.md`](../deploy/nginx/README.md)。

## 工作台

- **状态总览**：在线模块、平均响应、检测倒计时与最近 40 次检测的聚合趋势。
- **服务模块**：Live、SocketIO、OpenAPI、GitLab 四个探活入口，展示延迟、最近检测与逐次检测条；离线时给出失败原因。
- **GitHub 入口**：横向卡片直达 `maomao94/zero-service`，支持打开与复制地址；外部站点不参与探活。
- **端点速查**：`/v1/`、`/live/v1/`、`/socket.io/`、`/recordings/`、`/openapi/` 与上游说明，一键复制完整地址。
- **检测记录**：启动事件与上下线变化，可清空；侧栏同步最近 3 条。
- **交互**：`⌘K` 命令面板、深浅主题、移动端抽屉、可访问焦点与 reduced-motion。
- **健康检测**：15 秒同源探测，页面隐藏时暂停，回前台补检。

## 子系统

### Live 会议

- 登录、会议大厅、会议室、录制、邀请票据、SIP 电话与主持人管理。
- 登录页支持 **本地生成 JWT**（WebCrypto HS256），不请求 `POST /live/v1/generateToken`。
- 后端不可达时按本地 Token claims 展示用户信息，可只启动前端预览大厅。
- 生产态同源使用 `/live/` 与 `/socket.io/`，由 nginx 反代。

### SocketIO 测试

- 三区布局：深色顶栏 + 固定侧栏 + 独立工作区 + 可拖拽底部控制台。
- 连接配置、Token Claims、连接检查、本地签发 Token。
- 消息发送、房间管理（解析房间列表并回填）、事件监听与接收统计。
- 控制台支持类型筛选、自动滚动、导出、清空、详情展开与复制。

### OpenAPI 控制台

- API 调试、应用管理、错误码速查。
- 顶栏网关健康状态（可达耗时 / 不可达），点击重新检测。
- 生产态挂在 Live 站点的 `/openapi/`，`/v1/` 由 nginx 反代到 opengtw。

## 本地签发 Token

Live 与 SocketIO 均可在浏览器内生成测试 JWT：

- 算法 HS256，claims 与 `common/tool/jwtutil.go` 一致：`user-id`、`auth-type`、可选 `user-name` / `dept-code`。
- 默认密钥与 `app/livegtw/etc/livegtw.yaml` 的 `JwtAuth.AccessSecret` 一致，可在界面修改，也可用 `VITE_JWT_SECRET` 覆盖。
- 不请求后端，因此不需要为了拿 Token 先启动 livegtw；实际会议/网关连接仍需要对应服务。
- WebCrypto 只在 localhost / https 等安全上下文可用；密钥仅用于内部开发联调，不要在生产构建中暴露。

## 构建与部署

```bash
# 构建四个项目并汇总到 deploy/nginx/dist
bash deploy/nginx/build-web.sh

# 启动 / 更新 nginx 容器
bash deploy/nginx/deploy.sh
```

也可以单独构建某个项目：

```bash
bash deploy/nginx/build-web.sh workspace
```

## 验证状态

- 四个项目 `npm run build` 均通过。
- 本地 nginx 部署后：`/`、`/openapi/`、`:8090/`、`:8091/` 均返回 200；`/health/live`、`/health/socketio`、`/health/gitlab` 为 200。
- `/health/openapi` 在 opengtw 未启动时返回 502，工作台显示该模块未启动，属于预期。
- 桌面、移动端、深浅主题与本地 Token 流程已截图验证。

## 已知限制

- `livegtw`、`socketgtw`、`LiveKit` 未启动时，真实入会、Socket 连接、ACK 与房间查询无法端到端联调；前端静态页面与本地 Token 流程可用。
- Live 会议室页仅完成配色与顶栏统一，尚未在真实会议中复核。
- 详细说明见各项目 README 与 [`README.md`](./README.md)。
