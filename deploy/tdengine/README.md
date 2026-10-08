# TDengine TSDB 3.4 Docker 部署

TDengine 时序数据库本机 Docker 单机部署（社区版），用于时序数据存储与查询。

## 文件说明

| 文件 | 作用 |
| --- | --- |
| `deploy.sh` | 部署与运维脚本（单脚本子命令模式） |
| `docker-compose.yaml` | 容器编排配置（含逐行注释） |
| `init.d/` | 业务库初始化 SQL 目录（`deploy.sh init` 按文件名顺序执行，每个业务一个文件） |
| `data/` | 数据库持久化目录（gitignore，不入库） |
| `log/` | 日志持久化目录（gitignore，不入库） |

## 快速开始

```bash
# 1. 部署（自动拉取最新镜像 tdengine/tsdb:latest）
bash deploy/tdengine/deploy.sh
# 端口冲突时只修改宿主机映射端口（容器内端口不变）
TDENGINE_HOST_PORT=16030 TDENGINE_REST_PORT=16041 bash deploy/tdengine/deploy.sh

# 2. 验证
docker exec -it tdengine taos
# taos> show databases;

# 3. REST 接口验证
curl -u root:taosdata -d "show databases" localhost:6041/rest/sql

# 4. Web 管理界面（taosExplorer）
#    浏览器打开 http://localhost:6060

# 5. 业务库初始化（init.d/ 建库 + 追加业务超表 DDL，见下节）
bash deploy/tdengine/deploy.sh init model/sql/tdengine.sql
```

## deploy.sh 命令

```bash
bash deploy/tdengine/deploy.sh            # 部署（默认，幂等可重跑）
bash deploy/tdengine/deploy.sh init       # 业务库初始化（init.d/ 建库，可追加 DDL 参数，幂等）
bash deploy/tdengine/deploy.sh stop       # 移除容器，保留数据
bash deploy/tdengine/deploy.sh restart    # 重启容器
bash deploy/tdengine/deploy.sh status     # 容器状态
bash deploy/tdengine/deploy.sh logs       # 最近 100 行容器日志
bash deploy/tdengine/deploy.sh taos       # 进入 taos CLI 交互
```

**重新初始化**（危险操作，脚本不提供一键清空命令）：`stop` 后手动 `rm -rf data log`，再执行 `deploy.sh`。

## 业务库初始化

`deploy.sh init` 通用初始化入口（幂等，可重跑）：

1. **按文件名顺序执行 `init.d/*.sql`**——每个业务一个建库脚本（`01-create_iec104.sql` 建采集库，
   新业务按 `02-xxx.sql`、`03-xxx.sql` 递增命名加入）
2. **执行命令行追加的 SQL 文件**——业务超表 DDL 等不落本目录的脚本，直接传路径

```bash
# 仅执行 init.d/ 下的建库脚本
bash deploy/tdengine/deploy.sh init

# 建库 + 追加执行业务超表 DDL（可传多个文件）
bash deploy/tdengine/deploy.sh init model/sql/tdengine.sql

# 验证
docker exec tdengine taos -s "use iec104; show stables;"
```

### IEC 104 采集（当前已接入业务）

streamevent 服务的 IEC 104 协议采集数据写入 TDengine 超表（DDL 单源维护在 `model/sql/tdengine.sql`）：

| 超表 | 用途 |
| --- | --- |
| `iec104.raw_point_data` | 原始数据总表（ASDU 原始报文，子表按 `raw_<stationId>_<coa>_<ioa>` 自动创建） |
| `iec104.tele_signal_data` | 遥信表（开关状态，bool） |
| `iec104.telemetry_data` | 遥测表（模拟量，double） |

初始化命令（等价手动方式）：

```bash
# 方式一: deploy.sh init（推荐）
bash deploy/tdengine/deploy.sh init model/sql/tdengine.sql

# 方式二: 手动逐个执行（taos -s 整文件传参；注意 taos < file 管道方式会报
#         Incomplete SQL statement，taos CLI 把管道 stdin 按交互终端逐字符解析）
docker exec tdengine taos -s "$(cat deploy/tdengine/init.d/01-create_iec104.sql)"   # 建库 iec104
docker exec tdengine taos -s "$(cat model/sql/tdengine.sql)"                         # 建三张超表
```

**方式三: Web 管理界面（taosExplorer）**——图形化操作，不依赖命令行：

1. 浏览器打开 `http://localhost:6060/login`，使用 `root` / 部署密码登录
2. 进入左侧导航 **Data Browser（数据浏览器）**
3. 在 SQL 编辑器中依次粘贴执行（支持一条或多条 SQL）：
   - `deploy/tdengine/init.d/01-create_iec104.sql` 的建库语句（`CREATE DATABASE IF NOT EXISTS iec104 ...`）
   - `model/sql/tdengine.sql` 的三张超表 DDL（`CREATE STABLE IF NOT EXISTS ...`）
4. 左侧库表树中确认 `iec104` 库下出现三张超表

> Web 端与命令行方式等价，均幂等可重跑；SQL 编辑器执行的历史可收藏复用。

**服务配置对齐（重要）**：`facade/streamevent/etc/streamevent.yaml` 的 `TaosDB.DBName` 必须与库名
`iec104` 一致——示例配置默认值 `default` 不会命中本库，`INSERT ... USING` 要求超表已存在，
不一致时写入直接失败：

```yaml
TaosDB:
  DataSource: root:taosdata@http(localhost:6041)/?timezone=Asia%2FShanghai
  DBName: iec104
```

说明：
- DSN 使用 driver-go v3 `taosWS` 驱动（WebSocket 6041），`timezone` 参数 URL 编码为 Asia/Shanghai
- 修改 root 密码后 DSN 中的密码需同步更新
- 建库参数（`PRECISION 'ms'` / `KEEP 3650`）见 `init.d/01-create_iec104.sql`，按业务留存要求调整

## 连接信息

| 项 | 值 |
| --- | --- |
| Host / Port | `127.0.0.1:6030`（native）/ `127.0.0.1:6041`（REST） |
| 用户 / 密码 | `root` / `taosdata`（默认，生产环境必须修改） |
| Web 管理 | `http://localhost:6060`（taosExplorer） |
| 数据卷 | `./data`（容器 `/var/lib/taos`）、`./log`（容器 `/var/log/taos`） |
| CLI | `docker exec -it tdengine taos` |
| REST | `curl -u root:taosdata -d "show databases" localhost:6041/rest/sql` |

### 端口说明（对照官方 Network Port Requirements）

| 端口 | 组件 | 用途 |
| --- | --- | --- |
| 6030 | taosd | native 协议（taosc，taos CLI / JDBC-JNI / C/C++ 客户端） |
| 6041 | taosAdapter | WebSocket + RESTful 接口（宿主机外部访问推荐入口） |
| 6043 | taosKeeper | 监控服务 |
| 6044 | taosAdapter | StatsD 格式写入（TCP/UDP） |
| 6045 | taosAdapter | collectd 格式写入（TCP/UDP） |
| 6046-6049 | taosAdapter | OpenTSDB TELNET 及其变体格式写入 |
| 6050 | taosX | REST API |
| 6055 | taosX | gRPC（Agent / Xnode 数据接入） |
| 6060 | taosExplorer | Web 管理界面 |

### 连接方式建议

官方推荐宿主机外部访问优先使用 **RESTful/WebSocket（6041）**，不依赖 taosc，服务端升级不影响客户端；
native 协议（6030）需要客户端可解析容器 FQDN（`tdengine`），宿主机直连需在 `/etc/hosts` 添加：

```bash
echo "127.0.0.1 tdengine" | sudo tee -a /etc/hosts
```

### 修改 root 密码

```bash
# 方式一：环境变量（推荐，首次部署前设置）
TDENGINE_ROOT_PASSWORD=YourStrongPassword bash deploy/tdengine/deploy.sh

# 方式二：初始化后通过 SQL 修改
docker exec -it tdengine taos
# taos> alter user root password 'YourStrongPassword';
```

**注意（官方 3.3.8.8+ 行为）**：通过 SQL 修改 root 密码后，必须同步更新部署配置中的
`TDENGINE_ROOT_PASSWORD` 环境变量再重启容器，否则重启/升级镜像会因密码不一致失败。

## 环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `TDENGINE_HOST_PORT` | `6030` | 宿主机 native 协议端口 |
| `TDENGINE_REST_PORT` | `6041` | 宿主机 REST 接口端口 |
| `TDENGINE_ROOT_PASSWORD` | `taosdata` | root 密码，生产环境必须修改 |

## 集群部署（可选）

TDengine 支持多节点集群部署。官方 docker-compose 示例：

```yaml
services:
  td1:
    image: tdengine/tsdb:latest
    environment:
      - TAOS_FQDN=td1
  td2:
    image: tdengine/tsdb:latest
    environment:
      - TAOS_FQDN=td2
      - TAOS_FIRST_EP=td1:6030
  td3:
    image: tdengine/tsdb:latest
    environment:
      - TAOS_FQDN=td3
      - TAOS_FIRST_EP=td1:6030
```

详见[官方文档](https://docs.tdengine.com/operations-and-tooling/operations/deployment/docker/)。

## 镜像说明

- 社区版镜像：`tdengine/tsdb`（3.3.7.0 起由 `tdengine/tdengine` 更名）
- 企业版镜像：`tdengine/tsdb-ee`（3.3.7.0 起由 `tdengine/tdengine-ee` 更名）
- 当前 latest 版本：3.4.2.8
- 支持架构：linux/amd64、linux/arm64

## 参考

- 官方 Docker 部署文档: https://docs.tdengine.com/operations-and-tooling/operations/deployment/docker/
- 官方快速开始: https://docs.tdengine.com/quick-start/download-and-install/docker/
- Docker Hub: https://hub.docker.com/r/tdengine/tsdb
