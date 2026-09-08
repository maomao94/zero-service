# LiveKit SIP 部署

Docker Compose 部署 SIP 环境：
- LiveKit Server：WebRTC 媒体服务器（**不原生支持 TLS**，Config 结构体无 `tls` 字段，生产需反向代理终止 TLS）
- LiveKit SIP Server：SIP↔WebRTC 桥接
- FreeSWITCH：本地测试用 SIP 服务器（生产环境不需要）

#### Docker 网络与端口

Docker 网络 `lk-net`，容器 IP 在重建后可能变动。端口映射：

| 容器 | 端口映射 | 说明 |
|------|---------|------|
| livekit-server | 7880:7880, 60000-60100:60000-60100/udp | HTTP API + WebRTC 媒体 |
| freeswitch | 5060:5060, 5080:5080, 8021:8021, 16384-16484:16384-16484/udp | SIP + RTP |
| livekit-sip | 5070:5060, 11000-11100:11000-11100/udp | SIP 桥接 |

#### FreeSWITCH 配置挂载（`:ro` 单文件方案）

FreeSWITCH 镜像（safarov/freeswitch）启动时会检查 `/etc/freeswitch/freeswitch.xml` 是否存在，不存在则复制 vanilla 配置。利用 `:ro` 只读挂载覆盖关键文件，其余 vanilla 配置正常复制：

```yaml
# docker-compose.yaml
freeswitch:
  volumes:
    - ../freeswitch-vars.xml:/etc/freeswitch/vars.xml:ro          # 覆盖 domain/ext-rtp_ip
    - ../freeswitch-switch.conf.xml:/etc/freeswitch/autoload_configs/switch.conf.xml:ro  # 覆盖 RTP 端口范围
```

镜像启动的 vanilla 复制因只读挂载而跳过这两个文件，无需 entrypoint hack。

#### FreeSWITCH 关键配置

```xml
<!-- vars.xml -->
<X-PRE-PROCESS cmd="set" data="domain=192.0.2.10"/>        <!-- 示例地址；替换为软电话注册 IP -->
<X-PRE-PROCESS cmd="set" data="default_password=example-password"/>
<X-PRE-PROCESS cmd="set" data="external_rtp_ip=192.0.2.10"/> <!-- 示例地址；替换为宿主机局域网 IP -->
<X-PRE-PROCESS cmd="set" data="external_sip_ip=192.0.2.10"/>

<!-- switch.conf.xml -->
<param name="rtp-start-port" value="16384"/>
<param name="rtp-end-port" value="16484"/>  <!-- 范围 ≤100，Docker Desktop 端口映射上限 16k -->
```

> **Warning**: `external_rtp_ip` / `external_sip_ip` 必须设为宿主机局域网 IP，不能用 `host.docker.internal`。Docker Desktop 会把它解析为内部网关地址，宿主机软电话无法到达时会导致 RTP 媒体流不通（`packets: 0`，30s media-timeout 挂断）。

> **Warning**: Docker Desktop 端口映射上限约 16k 个 UDP 端口。FreeSWITCH 默认 RTP 范围 16384-32768（16k 端口）刚好达到上限，会导致 Docker Desktop 卡死。必须将 RTP 范围缩小到 ≤100（如 16384-16484）。

> **Warning**: `domain` 必须与软电话注册地址一致。例如软电话注册到示例地址 `192.0.2.10:5060` 时，domain 应为 `192.0.2.10`，不能使用 loopback 地址。

#### 软电话注册

软电话注册到宿主机局域网 IP（非 127.0.0.1），使 SDP 中 RTP IP 为宿主机 LAN IP，FreeSWITCH 容器可通过 Docker 网关到达：

```
sip:1001@192.0.2.10:5060  密码: example-password
```

#### 浏览器自动播放策略

现代 Chrome 浏览器要求用户交互后才能播放音频。LiveKit 的 `RoomAudioRenderer` 组件渲染远端音频，但首次播放需要用户点击页面。这是预期行为，不是 bug。

#### SIPMediaGW 视频 SIP 部署

LiveKit SIP Server 不支持视频（官方文档明确 "Video over SIP: Not Supported"）。视频 SIP 需要使用 [Renater/SIPMediaGW](https://github.com/Renater/SIPMediaGW) 桥接。

**关键约束**：SIPMediaGW 的 `deploy/` 目录是一个完整部署单元，不能只复制 `docker-compose.yml`。官方目录结构：

```
deploy/sipmediagw/
├── docker-compose.yml          # 官方 Compose 入口（非 docker-compose.yaml）
├── .env                        # 网关变量（DOCKER_IMAGE, ROOM, USER, SIP_REGISTRAR 等）
├── .env_kamailio               # Kamailio 变量（PUBLIC_IP, SIP_DOMAIN, TLS 等）
├── .env_turn                   # Coturn 变量（PUBLIC_IP, HEPLIFY_SRV）
├── .env_mysql                  # MySQL root 密码
├── .env_postgres               # Homer Postgres 凭据
├── .env_hep / .env_homer       # Homer/HEP 配置
├── kamailio/                   # 本地构建上下文（Dockerfile + 配置）
│   ├── Dockerfile
│   ├── config/                 # kamailio.cfg, kamctlrc, TLS 证书
│   └── db_healthcheck.sh
├── coturn/                     # 本地构建上下文
│   ├── Dockerfile
│   └── .env_cred               # TURN 凭据（TURN_USER/TURN_PASS）
├── scaler/                     # 扩缩容组件
├── services/                   # systemd 服务文件
│   ├── sipmediagw.service      # 网关实例启动（调用 start_all_gateways.bash）
│   ├── kamailio.service
│   ├── coturn.service
│   └── homer.service
└── proxyAPI/                   # 管理 API（可选）
```

**架构要点**：

| 服务 | 网络模式 | 说明 |
|------|---------|------|
| `sip_server` (Kamailio) | `network_mode: host` | SIP 信令，占用宿主机 5060/5061 |
| `turn_server` (Coturn) | `network_mode: host` | TURN/STUN，占用宿主机 3478/5349 |
| `scaler` | `network_mode: host` | 扩缩容 API |
| `sip_db` (MySQL) | bridge | Kamailio 数据库 |
| `hep_db` + `heplify_server` + `homer_webapp` | bridge | SIP 抓包/监控（可选） |

**关键事实**：`docker-compose.yml` 中**没有 `sipmediagw` service**。`docker compose up -d` 只启动基础设施（Kamailio、Coturn、DB、Homer）。媒体网关实例由 `services/sipmediagw.service` 通过 `start_all_gateways.bash` 脚本单独启动。这是上游仓库的真实部署模型。

**与本项目 LiveKit 的关系**：SIPMediaGW 的 Kamailio/Coturn 使用 `network_mode: host`，不使用项目 `deploy/livekit` 中的 `lk-net` bridge 网络。不能简单地把 `livekit-server` 容器名写成 SIPMediaGW 的 LiveKit 地址。SIPMediaGW 默认通过 Web 会议地址（如 `meet.livekit.io/rooms`）加入会议，具体 LiveKit 集成方式需按上游配置模型设置。

**同步方式**：从上游仓库整体复制 `deploy/` 目录，保留完整目录结构。同步后需重新检查 `.env`、数据库密码、端口和本地改动。

> **Warning**: 不要尝试只复制 `docker-compose.yml` 而省略构建上下文（kamailio/、coturn/）。`sip_server` 和 `turn_server` 使用 `build: context:`，缺少上下文会导致 `docker compose build` 失败。

> **Warning**: 官方 `provision.sh` 是 Ubuntu/Vagrant 初始化脚本，会安装内核模块（snd-aloop, v4l2loopback）、修改 systemd 和安装 Docker。生产服务器应先审阅再执行，不建议无审计直接执行。

#### SIP 静音同步

| 方向 | 机制 | 说明 |
|------|------|------|
| 主持人静音电话参与者 | LiveKit `MuteRoomTrack` API | ManagePane「静音语音」按钮调用，SIP 参与者音频轨道被服务端静音 |
| 电话侧静音 → 会议 UI | LiveKit SIP Server 检测 re-INVITE | 依赖 SIP Server 实现，可能不自动同步 |
