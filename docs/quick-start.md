# 快速开始

Zero-Service 包含多个可独立运行的服务。第一次使用时建议只启动一个目标服务，并按该服务的配置准备依赖，不要直接尝试全量启动整个仓库。

## 环境要求

| 依赖 | 用途 | 备注 |
| --- | --- | --- |
| Go 1.26+ | 编译和运行 | 版本以仓库根目录 `go.mod` 为准 |
| Git | 获取代码 | - |
| Docker Compose v2 | 可选，用于启动部署模板 | 仅适用于已完成环境变量和路径调整的场景 |

各服务的外部依赖不同，常见依赖包括 Redis、Kafka、关系数据库、MQTT Broker、Nacos、TDengine、对象存储和 Docker Engine。

## 安装

```bash
git clone https://github.com/maomao94/zero-service.git
cd zero-service
go mod download
```

`go mod download` 只下载依赖，不会像 `go mod tidy` 一样改写 `go.mod` 或 `go.sum`。

## 启动一个服务

各服务的配置文件通常位于对应目录的 `etc/` 下。以 Trigger 为例：

1. 编辑 `app/trigger/etc/trigger.yaml`，配置 Redis 和关系数据库；如需计划任务回调，再配置 `StreamEventConf`。
2. 启动服务：

```bash
cd app/trigger
go run . -f etc/trigger.yaml
```

其他常用服务：

| 服务 | 目录 | 启动命令 |
| --- | --- | --- |
| `ieccaller` | `app/ieccaller/` | `go run . -f etc/ieccaller.yaml` |
| `iecstash` | `app/iecstash/` | `go run . -f etc/iecstash.yaml` |
| `djicloud` | `app/djicloud/` | `go run . -f etc/djicloud.yaml` |
| `gtw` | `gtw/` | `go run . -f etc/gtw.yaml` |
| `socketgtw` | `socketapp/socketgtw/` | `go run . -f etc/socketgtw.yaml` |

> 上表命令需要在对应目录执行。默认配置中的地址、账号和端口仅用于本地示例，请在启动前替换为实际环境值。

## 服务依赖速查

| 服务 | 常见最小依赖 |
| --- | --- |
| `trigger` | Redis、MySQL/PostgreSQL；计划任务回调还需要 StreamEvent gRPC 服务 |
| `ieccaller` | IEC 104 从站；至少配置 Kafka、MQTT 或 gRPC 推送通道之一 |
| `iecstash` | Kafka、StreamEvent gRPC 服务；Nacos 为可选项 |
| `djicloud` | PostgreSQL、MQTT Broker；使用飞行区能力时还需要对象存储 |
| `gtw` | 配置中声明的上游 gRPC 服务（默认包括 `zerorpc` 和 `file`） |
| `file` | 关系数据库和对象存储（MinIO、阿里云 OSS 或腾讯 COS） |
| `bridgemodbus` | 关系数据库和 Modbus 设备 |
| `bridgemqtt` | MQTT Broker |
| `gis` | GEOS 运行库；数据库为可选项 |
| `podengine` | Docker Engine 或可访问的 Docker 主机 |

## Docker Compose

仓库提供的 `deploy/docker-compose.yml` 是部署模板，不是开箱即用的本地开发环境。使用前至少检查以下内容：

- `${REGISTER}` 和 `${MAIN_TAG}` 镜像变量；
- Kafka 对外广播地址；
- Filebeat、日志和隔离装置的宿主机路径；
- `network_mode: host`、特权容器和外部依赖是否符合部署环境。

```bash
cd deploy
export REGISTER=your-registry
export MAIN_TAG=latest
docker compose config
docker compose up -d
```

模板默认包含 Kafka、Filebeat、`ieccaller`、`iecstash`、`bridgegtw`、`bridgedump` 和 Kafdrop。修改模板后再启动，避免直接沿用示例中的主机地址和挂载路径。

## 验证

服务启动后，可按实际端口执行基础检查：

```bash
# Trigger 开发模式默认开启 gRPC reflection
grpcurl -plaintext localhost:21006 list

# gtw 已配置对应路由时检查 HTTP 入口
curl http://localhost:11001/health
```

完整端口和协议见[服务端口清单](./service-ports.md)。

## 常见问题

### 如何查看完整端口分配？

参考[服务端口清单](./service-ports.md)。约定为 HTTP `1xxxx`、gRPC `2xxxx`，部分协议服务还会额外监听设备 TCP 端口。

### 如何对接 IEC 104 数据？

先阅读[IEC 104 数采平台](./iec104.md)，再根据消费通道查看[IEC 104 消息对接](./iec104-message.md)。需要下发控制命令时，继续阅读[IEC 104 控制命令](./iec104-command.md)。

### 如何配置数据库和敏感信息？

参考目标服务 `etc/` 下的示例配置。生产环境请使用环境变量或配置中心注入账号、密码和密钥，不要把真实凭据提交到仓库。
