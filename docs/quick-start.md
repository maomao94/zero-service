# 快速开始

## 环境要求

| 依赖 | 版本 | 说明 |
|------|------|------|
| Go | 1.25+ | 编译运行 |
| Redis | 6.0+ | 任务队列、缓存（trigger 必需，其他可选） |
| Kafka | 3.0+ | 消息队列（iecstash 必需，其他可选） |
| MySQL/PostgreSQL | - | 关系数据库（djicloud 必需，其他可选） |
| TDengine | 3.0+ | 时序数据库（数采场景可选） |
| Nacos | 2.0+ | 服务发现（集群部署可选） |

## 安装

```bash
git clone https://github.com/maomao94/zero-service.git
cd zero-service
go mod tidy
```

## 启动

各服务可独立运行，按需启动。以 trigger 为例：

```bash
cd app/trigger
go run trigger.go -f etc/trigger.yaml
```

其他服务：

| 服务 | 目录 | 启动命令 |
|------|------|----------|
| ieccaller | `app/ieccaller/` | `go run ieccaller.go -f etc/ieccaller.yaml` |
| iecstash | `app/iecstash/` | `go run iecstash.go -f etc/iecstash.yaml` |
| djicloud | `app/djicloud/` | `go run djicloud.go -f etc/djicloud.yaml` |
| gtw | `gtw/` | `go run gtw.go -f etc/gtw.yaml` |
| socketgtw | `socketapp/socketgtw/` | `go run socketgtw.go -f etc/socketgtw.yaml` |

> 配置文件位于各服务 `etc/` 目录。端口分配见 [服务端口清单](service-ports.md)。

## Docker Compose

```bash
cd deploy
docker-compose up -d
```

默认启动：Kafka、Filebeat、ieccaller、iecstash、Kafdrop。按需修改 `docker-compose.yml`。

## 验证

```bash
# gRPC 服务健康检查
grpcurl -plaintext localhost:21006 list

# HTTP 网关
curl http://localhost:11001/health
```

## 常见问题

### 服务运行需要哪些外部依赖？

| 服务 | 最少依赖 |
|------|----------|
| trigger | Redis |
| ieccaller | Kafka（或 MQTT/gRPC 下游） |
| iecstash | Kafka |
| djicloud | PostgreSQL + MQTT Broker |
| gtw | 无（纯代理转发） |
| file | OSS（MinIO/阿里/腾讯） |
| bridgemodbus | Modbus 设备 |
| bridgemqtt | MQTT Broker |
| gis / podengine | 无 |

### 如何查看完整端口分配？

参考 [服务端口清单](service-ports.md)。规则：HTTP `1xxxx`，gRPC `2xxxx`。

### 如何对接 IEC 104 数据？

参考 [IEC 104 消息对接](iec104-message.md)。

### 如何配置数据库？

参考各服务 `etc/` 目录下的示例配置。DJI 云平台配置见 [DJI 云平台文档](djicloud.md)。
