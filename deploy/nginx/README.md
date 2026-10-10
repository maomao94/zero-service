# Zero Service Web 网关（Nginx）

静态托管前端产物（`deploy/nginx/dist`），并反向代理 livegtw / socketgtw / LiveKit / 录制文件，作为统一 https 入口。

## 文件说明

| 文件 | 作用 |
| --- | --- |
| `dist/` | 前端产物目录（`dist/live`、`dist/socketio`、`dist/workspace`），gitignore，部署时可自行覆盖/上传 |
| `build-web.sh` | 前端打包脚本：构建各 web 子项目并汇总产物到 `dist/<app>/` |
| `deploy.sh` | 部署与运维脚本（不构建、不校验产物，仅部署） |
| `docker-compose.yaml` | 容器编排（只读挂载 `dist/<app>` 与录制目录、证书） |
| `nginx.conf` | server 定义（Live 网关 HTTP/HTTPS、SocketIO 工具、工作台） |
| `live-locations.conf` | Live 网关共享 location（前端 SPA、录制、反代） |

## 快速开始

```bash
# 1. 生成 TLS 证书（Nginx 不自签，证书缺失会直接报错）
bash deploy/tls/gen-tls.sh

# 2. 构建前端产物到 deploy/nginx/dist（测试/生产可跳过，自行上传覆盖 dist）
bash deploy/nginx/build-web.sh

# 3. 部署
bash deploy/nginx/deploy.sh
```

> 也可以只用 `deploy.sh`：它只做部署，不构建前端、不校验产物；`dist` 缺失时对应站点为空（自行准备）。

## 访问地址

| 站点 | HTTP（推荐） | HTTPS |
| --- | --- | --- |
| Live 会议前端 | http://127.0.0.1:8088/ | https://127.0.0.1:8443/ |
| 录制文件 | http://127.0.0.1:8088/recordings/ | https://127.0.0.1:8443/recordings/ |
| SocketIO 测试工具 | http://127.0.0.1:8090/ | - |
| Zero 工作台 | http://127.0.0.1:8091/ | - |

端口可用环境变量覆盖：`NGINX_HOST_PORT`、`NGINX_SSL_HOST_PORT`、`SOCKETIO_HOST_PORT`、`WORKSPACE_HOST_PORT`。

## 反向代理规则（Live 网关）

| 路径 | 目标 | 说明 |
| --- | --- | --- |
| `/` | `dist/live` | 会议前端（SPA 回落 index.html） |
| `/recordings/` | `deploy/livekit/recordings` | 录制文件（只读、Range 播放，无目录浏览） |
| `/live/` | `livegtw:11002` | 会议 HTTP API（`/live/v1/...`） |
| `/socket.io/` | `socketgtw:11003` | 会议邀请推送（Socket.IO / WS） |
| `/livekit/` | `livekit-server:7880` | 信令（WS，去掉 `/livekit` 前缀） |

反代目标通过 `host.docker.internal` 访问宿主机服务（Linux 由 compose `extra_hosts` 注入）。

## 工作台子系统健康检查

工作台 server（8091）提供同源健康检查，供工作台探测子系统在线状态：

| 路径 | 目标（同容器） | 说明 |
| --- | --- | --- |
| `/health/live` | `127.0.0.1:80/healthz` | Live 站点健康检查 |
| `/health/socketio` | `127.0.0.1:8090/` | SocketIO 测试工具首页 |

## HTTPS / 证书

- 证书来自 `deploy/tls/server.crt|key`（用 `bash deploy/tls/gen-tls.sh` 生成，或用正式证书）。
- 也可用 `NGINX_TLS_CERT` / `NGINX_TLS_KEY` 指向其他证书路径。
- Nginx **不生成自签证书**：证书不存在时部署直接报错。
- 自签证书的信任由使用方自行处理（浏览器手动信任等）。

## 与录制功能对接

`app/live/etc/live.yaml` 的 `LiveKit.Record.PlayURLBase` 用**同源相对路径**：

```yaml
LiveKit:
  Record:
    PlayURLBase: "/recordings"
```

这样无论页面是 http(8088) 还是 https(8443)，播放地址都解析到当前站点 `/recordings/...`，不会出现混合内容。Vite 开发模式（5178）已在 `web/live/vite.config.ts` 增加 `/recordings` 代理到本网关。

## 注意

- 反代目标（livegtw/socketgtw/LiveKit）需先启动，否则对应接口返回 502。
- 当前为本地开发配置；生产环境请补充鉴权、限流与访问控制。
