# Zero Service Web 网关（Nginx）

静态托管前端产物（`deploy/nginx/dist`），并反向代理 livegtw / socketgtw / LiveKit / 录制文件，同时提供公司 GitLab 的本地代理入口。

## 文件说明

| 文件 | 作用 |
| --- | --- |
| `dist/` | 前端产物目录（`dist/live`、`dist/socketio`、`dist/workspace`、`dist/openapi`），gitignore，部署时可自行覆盖/上传 |
| `build-web.sh` | 前端打包脚本：构建各 web 子项目并汇总产物到 `dist/<app>/` |
| `deploy.sh` | 部署与运维脚本（不构建、不校验产物，仅部署） |
| `docker-compose.yaml` | 容器编排（只读挂载配置目录、`dist/<app>`、录制目录与证书） |
| `nginx.conf` | 主配置：`events` / `http` / `stream` 与全局 include |
| `conf.d/default.conf` | HTTP 站点定义（Live 默认站点、SocketIO 工具、工作台、OpenAPI 控制台） |
| `stream.d/gitlab-ssh.conf` | GitLab SSH（容器 22 → `10.10.1.100:2224`）TCP 转发 |
| `snippets/live-locations.conf` | Live 网关共享 location（前端 SPA、录制、反代） |
| `snippets/openapi-locations.conf` | app 开放 API location（`/openapi/` 控制台、`/v1/` 反代；`/open-admin/` 不对外） |
| `snippets/gitlab.conf` | 公司 GitLab 反向代理（子域名 `gitlab.localhost` HTTP/HTTPS） |
| `snippets/gitlab-upstream.conf` | GitLab 上游代理参数（`proxy_pass`、Header、流式与超时） |
| `snippets/gitlab-api-proxy.conf` | GitLab API 代理公共参数（Header、超时、关闭上游压缩） |
| `snippets/gitlab-api-rewrite.conf` | API JSON 地址回写（兼容 OpenChamber 项目校验） |
| `snippets/tls.conf` | HTTPS 共享证书与协议参数 |

目录约定：

```
deploy/nginx/
├── nginx.conf            # 主配置（容器 /etc/nginx/nginx.conf）
├── conf.d/               # HTTP server 块（容器 /etc/nginx/conf.d）
├── stream.d/             # TCP/SSH 转发（容器 /etc/nginx/stream.d）
├── snippets/             # 可复用 location / 参数（容器 /etc/nginx/snippets）
├── dist/                 # 前端产物
└── docker-compose.yaml
```

## 端口规划

| 宿主机端口（可覆盖） | 容器端口 | 用途 |
| --- | --- | --- |
| `80`（`NGINX_HOST_PORT`） | 80 | 统一 HTTP 入口：Live 默认站点 + GitLab 子域名 |
| `443`（`NGINX_SSL_HOST_PORT`） | 443 | 统一 HTTPS 入口：同上 |
| `8090`（`SOCKETIO_HOST_PORT`） | 8090 | SocketIO 测试工具（仅 HTTP） |
| `8091`（`WORKSPACE_HOST_PORT`） | 8091 | Zero 工作台（仅 HTTP） |
| `22`（`GITLAB_SSH_HOST_PORT`） | 22 | GitLab SSH 转发（`git@gitlab.localhost`，客户端免 ssh 端口配置） |

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
| GitLab SSH | `git@gitlab.localhost:<group>/<repo>.git` | - |
| SocketIO 测试工具 | http://127.0.0.1:8090/ | - |
| Zero 工作台 | http://127.0.0.1:8091/ | - |
| OpenAPI 控制台 | http://127.0.0.1/openapi/ | https://127.0.0.1/openapi/ |

端口可用环境变量覆盖：`NGINX_HOST_PORT`、`NGINX_SSL_HOST_PORT`、`SOCKETIO_HOST_PORT`、`WORKSPACE_HOST_PORT`、`GITLAB_SSH_HOST_PORT`。

## 反向代理规则（Live 网关）

| 路径 | 目标 | 说明 |
| --- | --- | --- |
| `/` | `dist/live` | 会议前端（SPA 回落 index.html） |
| `/recordings/` | `deploy/livekit/recordings` | 录制文件（只读、Range 播放，无目录浏览） |
| `/live/` | `livegtw:11002` | 会议 HTTP API（`/live/v1/...`） |
| `/socket.io/` | `socketgtw:11003` | 会议邀请推送（Socket.IO / WS） |
| `/livekit/` | `livekit-server:7880` | 信令（WS，去掉 `/livekit` 前缀） |
| `/openapi/` | `dist/openapi` | OpenAPI 测试控制台（SPA 回落 index.html） |
| `/v1/` | `opengtw:11005` | app 开放 API（`/v1/gateway`、`/v1/health`） |

反代目标通过 `host.docker.internal` 访问宿主机服务（Linux 由 compose `extra_hosts` 注入）。

## 公司 GitLab 反向代理

`gitlab.conf` 以子域名方式代理公司 GitLab（`10.10.1.100:8929`），路径原样透传，网页端 CSS/JS/跳转全部正常：

| 入口 | 目标 |
| --- | --- |
| `http://gitlab.localhost/` | `http://10.10.1.100:8929` |
| `https://gitlab.localhost/` | 同上（复用网关自签证书） |

- 浏览器访问 `*.localhost` 会自动解析到 `127.0.0.1`；但 **ssh 与 Node 不会**，命令行使用前需要一条 hosts 记录：`127.0.0.1 gitlab.localhost`（或内网 DNS）。
- SSH clone/fetch（推荐）：`git clone git@gitlab.localhost:<group>/<repo>.git`。网关把容器 22 端口透传到 `10.10.1.100:2224`，**客户端无需 `~/.ssh/config` 端口映射**。
- HTTPS clone 需信任网关自签证书，例如：`git -c http.sslCAInfo=deploy/tls/server.crt clone https://gitlab.localhost/<group>/<repo>.git`。
- 代理参数：`client_max_body_size 0`（大仓库 push）、`proxy_buffering off`（流式 clone）、600s 超时。
- 只监听 `gitlab.localhost`，不影响默认站点（`127.0.0.1` 仍是 Live 前端）。
- upstream 不可达时返回 502；如需改地址，修改 `snippets/gitlab-upstream.conf` 中的 `proxy_pass`。

### OpenChamber 项目地址回写（临时兼容）

GitLab 的项目接口固定按 `external_url`（当前为 `http://10.10.1.100:8929`）返回
`web_url`、`http_url_to_repo`、`ssh_url_to_repo`。OpenChamber 会把这些地址的 host
与实例地址（`https://gitlab.localhost`）比对，不一致会报
`GitLab returned an invalid project`。

网关因此对 `https://gitlab.localhost/api/` 的 JSON 响应做 host 回写
（`snippets/gitlab-api-rewrite.conf`），只影响本机 vhost，不改 GitLab 配置。
GitLab 的 `external_url` 改为正式域名后，删除该 location 与片段即可。
分页使用 `x-next-page`，不依赖 `Link` 头，无需额外处理。

### 接入 OpenChamber

推荐使用子域名 HTTPS 入口，使 OpenChamber 的实例地址与 git 远端主机名一致（自动识别）：

1. GitLab 生成个人访问令牌：头像 → **Preferences → Access Tokens**，勾选 `api`（需要读写仓库再勾 `read_repository`/`write_repository`）。
2. 让 OpenChamber 信任网关证书（自签时必须）：`launchctl setenv NODE_EXTRA_CA_CERTS <repo>/deploy/tls/server.crt`，然后完全退出并重启 OpenChamber。
3. 设置 → **GitLab → 添加实例**：`https://gitlab.localhost`，粘贴 token。
4. git 远端用 SSH：`git@gitlab.localhost:<group>/<repo>.git`（走网关 22 端口转发，不受 HTTPS 传输校验影响）。


## 新增一个平台代理（扩展约定）

统一入口只有容器内的 80/443（宿主机标准端口 80/443），**新平台不要再占新端口**，按子域名分流：

1. 新建 `deploy/nginx/conf.d/<app>.conf`，内容是一个完整 server 块：

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

2. 无需修改 `nginx.conf`：主配置已 `include /etc/nginx/conf.d/*.conf`。
3. 可复用的 location/参数片段放 `deploy/nginx/snippets/`，用 `include /etc/nginx/snippets/<name>.conf;` 引用。
4. compose 已按目录挂载 `conf.d/`、`stream.d/`、`snippets/`，新增文件无需改挂载。
5. `docker exec web-gateway nginx -t && docker exec web-gateway nginx -s reload`。

需要独立端口的场景（如本仓库的 GitLab SSH 22）放 `stream.d/`，并在 `nginx.conf` 的 stream include 范围内生效。

规则：

| 场景 | 做法 |
| --- | --- |
| 新平台网页（有自己的绝对路径资源） | 子域名 server + `proxy_pass` 原样透传 |
| 后端 API / 网关（路径无冲突） | 在对应站点内加 `location /xxx/`，可改写前缀 |
| 必须独立端口（如专用 WS、探测） | compose 增加映射 + 新 `listen`，并在 README 端口表登记 |

只有必须独立的场景才新增端口；能用 `Host` 区分的都走 80/443。

## 工作台子系统健康检查

工作台 server（8091）提供同源健康检查，供工作台探测子系统在线状态：

| 路径 | 目标 | 说明 |
| --- | --- | --- |
| `/health/live` | `127.0.0.1:80/healthz` | Live 站点健康检查 |
| `/health/socketio` | `127.0.0.1:8090/` | SocketIO 测试工具首页 |
| `/health/openapi` | `host.docker.internal:11005/v1/health` | 开放 API 网关健康检查 |
| `/health/gitlab` | `gitlab.localhost/users/sign_in` | GitLab 登录页可达性 |

- OpenAPI 网关未启动时 `/health/openapi` 返回 502，工作台会显示该模块未启动，其余站点不受影响。
- GitHub 入口是外部链接，不经过网关；Live / SocketIO 的本地 JWT 签发在浏览器内完成，不需要后端。
- SocketIO 测试工具在 8090 站点内同源请求 `/socket.io/`，由该 server 反代到 socketgtw；前端生产构建不需要填写 `11003`。

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
