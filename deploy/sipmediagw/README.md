# SIPMediaGW 部署

本目录是从 [Renater/SIPMediaGW](https://github.com/Renater/SIPMediaGW) 的 `deploy/` 目录复制的完整部署单元，不是一个可以独立运行的简化 Compose 文件。

官方部署包含以下服务和依赖：

- `sip_server`: 本地构建的 Kamailio 镜像，使用 host network
- `sip_db`: Kamailio MySQL 数据库
- `turn_server`: 本地构建的 Coturn 镜像，使用 host network
- `hep_db`、`heplify_server`、`homer_webapp`: SIP 抓包和 Homer 监控
- `scaler`: 官方的扩缩容组件，默认使用 host network
- `services/sipmediagw.service`: 通过官方脚本启动媒体网关实例

注意：官方 `docker-compose.yml` 没有 `sipmediagw` service。`docker compose up -d` 只启动基础设施；媒体网关镜像由官方 systemd service 和启动脚本单独启动。

## 重要限制

- 官方 Compose 的入口文件是 `docker-compose.yml`，执行命令必须在本目录运行。
- 官方部署的 Kamailio、Coturn 和 scaler 使用 `network_mode: host`，数据库和 Homer 使用 Compose 默认 bridge 网络。它不使用项目 `deploy/livekit` 中的 `lk-net` 网络。不要把 `livekit-server` 容器名直接写成 SIPMediaGW 的 LiveKit 地址；SIPMediaGW 的默认配置通过 Web 会议地址加入会议，具体 LiveKit 集成参数需要按上游项目配置模型设置。
- `sip_server`、`turn_server`、`scaler` 使用本地 `build:`，所以不能只复制 `docker-compose.yml`，必须保留整个目录结构。
- 官方配置会占用宿主机的 SIP、TURN、数据库、Homer 和 HEP 端口。正式部署前确认没有 FreeSWITCH、LiveKit SIP Server 或其他 SIP 服务占用这些端口。
- `provision.sh` 是 Ubuntu/Vagrant 风格的初始化脚本，会安装软件包、修改内核模块和 systemd。生产服务器应先审阅再执行，不建议无审计直接执行。

## 测试环境

### 1. 准备目录

```bash
cd deploy/sipmediagw
cp .env.example .env
```

修改 `.env`：

- `HOST_IP`: 测试机可被 SIP 终端访问的 IPv4 地址
- `ID`: 网关编号，必须是数字
- `ROOM`: 测试会议名
- `USER`: 测试网关用户名
- `SIP_REGISTRAR`、`SIP_PROXY`、`TURN_SRV`: 测试环境中的实际地址

同时检查 `.env_kamailio`、`.env_turn` 和 `kamailio/config/.env_cred` 中的凭据。测试环境可以使用上游默认值，但不要把它们用于生产。

### 2. 校验并启动基础设施

```bash
docker compose -f docker-compose.yml config
docker compose -f docker-compose.yml build
docker compose -f docker-compose.yml up -d
docker compose -f docker-compose.yml ps
```

Compose 完成后，网关本身还没有启动。测试机上可以使用官方启动脚本：

```bash
export HOST_IP=127.0.0.1
export HOST_TZ=UTC
export MAIN_APP=baresip
export BROWSING=false
export ID=1
export ROOM=test-room
export USER=sip-bridge
export DOCKER_IMAGE=renater/sipmediagw:1.8.9
bash services/start_all_gateways.bash
```

脚本会读取上游项目约定的环境变量并启动网关容器。生产环境应使用 systemd 部署方式，见下文。

查看日志：

```bash
docker compose -f docker-compose.yml logs -f sip_server turn_server
```

停止但保留数据：

```bash
docker compose -f docker-compose.yml down
```

### 4. 可选监控

Homer 和 HEP 服务包含在官方 Compose 中：

- Homer: `http://<HOST_IP>:8080`
- HEP TCP: `9060`
- HEP UDP: `9060`

## 生产环境

### 1. 复制项目部署目录

如果目标服务器上的项目目录是 `/opt/limian1`，推荐将本目录整体复制，不要只复制 Compose 文件：

```bash
mkdir -p /opt/limian1/deploy
rsync -a --delete deploy/sipmediagw/ user@production-host:/opt/limian1/deploy/sipmediagw/
ssh user@production-host
cd /opt/limian1/deploy/sipmediagw
cp .env.example .env
```

也可以直接从上游仓库复制官方目录：

```bash
git clone --depth 1 https://github.com/Renater/SIPMediaGW.git /tmp/SIPMediaGW
cp -R /tmp/SIPMediaGW/deploy/. /opt/limian1/deploy/sipmediagw/
```

### 2. 必须修改的生产配置

- `HOST_IP`: 公网 SIP/RTP/TURN 地址或正确的 NAT 映射地址
- `DOCKER_IMAGE`: 固定经过验证的版本，不使用 `latest`
- `ROOM`、`USER`、`GW_NAME`、`ID`: 使用生产唯一值
- `SIP_DOMAIN`: 配置正式 SIP 域名
- `.env_mysql`、`kamailio/config/.env_cred`、`coturn/.env_cred`: 使用随机高强度密码
- `.env_postgres`: 修改 Homer 数据库密码
- `.env_turn`: 配置生产 TURN 公网地址和凭据
- `.env_kamailio`: 关闭不需要的调试能力并设置 `PUBLIC_IP`、`LOCAL_IP`、`HEPLIFY_SRV`
- 防火墙：只开放实际需要的 SIP、RTP、TURN、HEP 和 Homer 管理端口

上游根目录还有一个 `.env` 文件，里面包含 `DOCKER_IMAGE`、`ROOM`、`USER`、`SIP_REGISTRAR` 等网关变量。本项目提供的 `.env.example` 是部署模板；正式部署时执行 `cp .env.example .env`，再按实际网络和会议接入方式补全。不要把生产 `.env` 提交到 Git。

### 3. 生产启动前检查

```bash
docker compose -f docker-compose.yml config
docker compose -f docker-compose.yml build --pull
docker compose -f docker-compose.yml up -d
docker compose -f docker-compose.yml ps
```

首次生产部署建议先只启动数据库并确认健康状态：

```bash
docker compose -f docker-compose.yml up -d sip_db hep_db
docker compose -f docker-compose.yml ps
```

确认数据库健康后再启动其余服务：

```bash
docker compose -f docker-compose.yml up -d
```

安装并启用官方 systemd 服务前，确保项目路径与 service 文件中的路径一致。上游 service 默认使用 `/sipmediagw`，如果项目部署在 `/opt/limian1/deploy/sipmediagw`，需要把 service 文件中的路径改为实际绝对路径，或将项目放到 `/sipmediagw`：

```bash
sed -i 's#/sipmediagw#/opt/limian1/deploy/sipmediagw#g' services/*.service
sudo cp services/*.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now coturn.service kamailio.service homer.service sipmediagw.service
```

上游 `sipmediagw.service` 负责网关实例，`kamailio.service`、`coturn.service` 和 `homer.service` 负责对应基础服务。若只使用 Compose 管理基础设施，也可以不安装这些 systemd 单元，但必须自行执行等价的启动脚本。

### 4. 备份和升级

```bash
docker compose -f docker-compose.yml down
docker compose -f docker-compose.yml pull
docker compose -f docker-compose.yml build --pull
docker compose -f docker-compose.yml up -d
```

升级前备份 Docker volumes `kamailio_db` 和 `hep_db`。不要使用 `down -v`，否则会删除数据库卷。

## 和本项目 LiveKit 的关系

本目录只负责 SIPMediaGW 官方部署。项目中的 `deploy/livekit/docker-compose.yaml` 负责 LiveKit Server。两者是否能直接组成当前项目的 LiveKit 视频桥接，取决于 SIPMediaGW 上游版本支持的 Web 会议接入方式和网络地址，不能仅凭添加一个 `lk-net` 网络保证可用。部署视频 SIP 前必须使用实际的 LiveKit 房间 URL、认证参数和 SIPMediaGW 版本做端到端验证。

## 来源和同步

当前目录同步自上游 `main` 分支的 `deploy/` 目录。后续同步时建议重新复制整个目录，并重新检查 `.env`、数据库密码、端口和本地改动：

```bash
cp -R /path/to/SIPMediaGW/deploy/. deploy/sipmediagw/
```
