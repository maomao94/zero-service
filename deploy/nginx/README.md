# Zero Service Web 网关（Nginx）

静态托管前端产物（`deploy/nginx/dist`），并反向代理 livegtw / socketgtw / LiveKit / 录制文件，同时提供公司 GitLab 的本地代理入口。

## 文件说明

| 文件 | 作用 |
| --- | --- |
| `dist/` | 前端产物目录（`dist/live`、`dist/socketio`、`dist/workspace`），gitignore，部署时可自行覆盖/上传 |
| `build-web.sh` | 前端打包脚本：构建各 web 子项目并汇总产物到 `dist/<app>/` |
| `deploy.sh` | 部署与运维脚本（不构建、不校验产物，仅部署） |
| `docker-compose.yaml` | 容器编排（只读挂载 `dist/<app>` 与录制目录、证书） |
| `nginx.conf` | server 定义与端口规划（Live 默认站点 HTTP/HTTPS、SocketIO 工具、工作台） |
| `live-locations.conf` | Live 网关共享 location（前端 SPA、录制、反代） |
| `gitlab.conf` | 公司 GitLab 反向代理（子域名 `gitlab.localhost` HTTP/HTTPS + 本机 `127.0.0.1:8929`） |
| `gitlab-upstream.conf` | GitLab 上游代理参数（`proxy_pass`、Header、流式与超时） |
| `tls.conf` | HTTPS 共享证书与协议参数 |

## 端口规划

| 宿主机端口（可覆盖） | 容器端口 | 用途 |
| --- | --- | --- |
| `80`（`NGINX_HOST_PORT`） | 80 | 统一 HTTP 入口：Live 默认站点 + GitLab 子域名 |
| `443`（`NGINX_SSL_HOST_PORT`） | 443 | 统一 HTTPS 入口：同上 |
| `8090`（`SOCKETIO_HOST_PORT`） | 8090 | SocketIO 测试工具（仅 HTTP） |
| `8091`（`WORKSPACE_HOST_PORT`） | 8091 | Zero 工作台（仅 HTTP） |
| `8929`（`GITLAB_DIRECT_PORT`） | 8929 | GitLab 本机直连，仅绑定 `127.0.0.1`（OpenChamber 集成用） |

容器内 80/443 按 `Host` 头分发：默认站点是 Live，`gitlab.localhost` 走 GitLab 反代。

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
| Live 会议前端 | http://127.0.0.1/ | https://127.0.0.1/ |
| 录制文件 | http://127.0.0.1/recordings/ | https://127.0.0.1/recordings/ |
| 公司 GitLab | http://gitlab.localhost/ | https://gitlab.localhost/ |
| GitLab（OpenChamber） | http://127.0.0.1:8929/ | - |
| SocketIO 测试工具 | http://127.0.0.1:8090/ | - |
| Zero 工作台 | http://127.0.0.1:8091/ | - |

端口可用环境变量覆盖：`NGINX_HOST_PORT`、`NGINX_SSL_HOST_PORT`、`SOCKETIO_HOST_PORT`、`WORKSPACE_HOST_PORT`、`GITLAB_DIRECT_PORT`。

## 反向代理规则（Live 网关）

| 路径 | 目标 | 说明 |
| --- | --- | --- |
| `/` | `dist/live` | 会议前端（SPA 回落 index.html） |
| `/recordings/` | `deploy/livekit/recordings` | 录制文件（只读、Range 播放，无目录浏览） |
| `/live/` | `livegtw:11002` | 会议 HTTP API（`/live/v1/...`） |
| `/socket.io/` | `socketgtw:11003` | 会议邀请推送（Socket.IO / WS） |
| `/livekit/` | `livekit-server:7880` | 信令（WS，去掉 `/livekit` 前缀） |

反代目标通过 `host.docker.internal` 访问宿主机服务（Linux 由 compose `extra_hosts` 注入）。

## 公司 GitLab 反向代理

`gitlab.conf` 以子域名方式代理公司 GitLab（`10.10.1.100:8929`），路径原样透传，网页端 CSS/JS/跳转全部正常：

| 入口 | 目标 |
| --- | --- |
| `http://gitlab.localhost/` | `http://10.10.1.100:8929` |
| `https://gitlab.localhost/` | 同上（复用网关自签证书） |

- `*.localhost` 由浏览器自动解析到 `127.0.0.1`，无需修改 hosts。
- git 命令行同样可用：`git clone http://gitlab.localhost/<group>/<repo>.git`。
- 代理参数：`client_max_body_size 0`（大仓库 push）、`proxy_buffering off`（流式 clone）、600s 超时。
- 只监听 `gitlab.localhost`，不影响默认站点（`127.0.0.1` 仍是 Live 前端）。
- upstream 不可达时返回 502；如需改地址，修改 `gitlab-upstream.conf` 中的 `proxy_pass`。

### 接入 OpenChamber

OpenChamber 的 GitLab 集成只接受 HTTPS，或 `http://` + `localhost/127.0.0.1`（`gitlab.localhost` 不在白名单内），因此使用本机直连端口：

1. GitLab 生成个人访问令牌：头像 → **Preferences → Access Tokens**，勾选 `api`（需要读写仓库再勾 `read_repository`/`write_repository`）。
2. OpenChamber → 设置 → **GitLab → 添加实例**：
   - 实例：`http://127.0.0.1:8929`
   - 个人访问令牌：粘贴上一步的 token
3. 连接成功后即可在 OpenChamber 内浏览 MR / Issue / CI。

该端口只在宿主机 `127.0.0.1` 监听，不对局域网暴露。

## 新增一个平台代理（扩展约定）

统一入口只有容器内的 80/443（宿主机标准端口 80/443），**新平台不要再占新端口**，按子域名分流：

1. 新建 `deploy/nginx/<app>.conf`，内容是一个完整 server 块：

   ```nginx
   server {
       listen 80;
       server_name <app>.localhost;
       client_max_body_size 0;                 # 按需

       location / {
           proxy_pass http://<upstream-host>:<port>;   # 路径原样透传，不要加前缀
           proxy_http_version 1.1;
           proxy_set_header Host              $http_host;   # 保留原始 Host:Port
           proxy_set_header X-Real-IP         $remote_addr;
           proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
           proxy_read_timeout 600s;
           proxy_send_timeout 600s;
       }
   }
   ```

2. `nginx.conf` 里加一行：`include /etc/nginx/snippets/<app>.conf;`
3. `docker-compose.yaml` 里加一行挂载：`- ./<app>.conf:/etc/nginx/snippets/<app>.conf:ro`
4. `docker exec web-gateway nginx -t && docker exec web-gateway nginx -s reload`

规则：

| 场景 | 做法 |
| --- | --- |
| 新平台网页（有自己的绝对路径资源） | 子域名 server + `proxy_pass` 原样透传 |
| 后端 API / 网关（路径无冲突） | 在对应站点内加 `location /xxx/`，可改写前缀 |
| 必须独立端口（如专用 WS、探测） | compose 增加映射 + 新 `listen`，并在 README 端口表登记 |

只有必须独立的场景才新增端口；能用 `Host` 区分的都走 80/443。

## 工作台子系统健康检查

工作台 server（8091）提供同源健康检查，供工作台探测子系统在线状态：

| 路径 | 目标（同容器） | 说明 |
| --- | --- | --- |
| `/health/live` | `127.0.0.1:80/healthz` | Live 站点健康检查 |
| `/health/socketio` | `127.0.0.1:8090/` | SocketIO 测试工具首页 |
| `/health/gitlab` | `gitlab.localhost/users/sign_in` | GitLab 登录页可达性 |

## HTTPS / 证书

- 证书来自 `deploy/tls/server.crt|key`（用 `bash deploy/tls/gen-tls.sh` 生成，或用正式证书）。
- HTTPS 公共参数（证书、协议、会话缓存）统一在 `tls.conf`，Live 与 GitLab 的 443 server 共享。
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

这样无论页面是 http(80) 还是 https(443)，播放地址都解析到当前站点 `/recordings/...`，不会出现混合内容。Vite 开发模式（5178）已在 `web/live/vite.config.ts` 增加 `/recordings` 代理到本网关。

## 注意

- 反代目标（livegtw/socketgtw/LiveKit）需先启动，否则对应接口返回 502。
- 当前为本地开发配置；生产环境请补充鉴权、限流与访问控制。
