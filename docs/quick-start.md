# 快速开始

## 环境要求

| 依赖 | 版本 | 说明 |
|------|------|------|
| Go | 1.25+ | 主要开发语言 |
| Redis | 6.0+ | 任务队列、缓存 |
| Kafka | 3.0+ | 消息队列（数采场景） |
| MySQL/PostgreSQL | - | 关系数据库（按需） |
| TDengine | 3.0+ | 时序数据库（数采场景） |
| Docker | 20.10+ | 容器部署（可选） |
| Nacos | 2.0+ | 服务发现（集群部署） |

## 安装

```bash
git clone https://github.com/maomao94/zero-service.git
cd zero-service
go mod tidy
```

## 启动单个服务

### Trigger 任务调度服务

```bash
cd app/trigger
go run trigger.go -f etc/trigger.yaml
```

### IEC 104 主站

```bash
cd app/ieccaller
go run ieccaller.go -f etc/ieccaller.yaml
```

### DJI 云平台服务

```bash
cd app/djicloud
go run djicloud.go -f etc/djicloud.yaml
```

### BFF 网关

```bash
cd gtw
go run gtw.go -f etc/gtw.yaml
```

## Docker Compose 启动

```bash
cd deploy
# 按需修改 docker-compose.yml 和环境变量
docker-compose up -d
```

默认包含：Kafka、Filebeat、ieccaller、bridgegtw、bridgedump、iecstash、Kafdrop。

## 配置说明

各服务配置文件位于 `app/{service}/etc/` 或网关自身 `etc/` 目录。

典型配置项：
- 服务监听地址和端口
- Redis / Kafka / 数据库连接
- Nacos 服务注册配置
- 协议特定配置（IEC 104 从站列表、MQTT Broker 等）

## 常见问题

### Q: 如何只启动一个服务？

进入服务目录，`go run` 指定配置文件即可。各服务可独立运行。

### Q: 没有 Kafka/TDengine 能运行吗？

部分服务可以：
- **trigger**：只需要 Redis
- **file**：只需要 OSS 配置
- **gis/alarm/podengine**：独立运行
- **ieccaller**：需要 Kafka 或 MQTT 或 gRPC 下游

### Q: 如何查看服务端口？

参考 [服务端口清单](service-ports.md)。

### Q: 如何调试 gRPC 服务？

```bash
# 列出服务
grpcurl -plaintext localhost:21006 list

# 调用方法
grpcurl -plaintext -d '{"name":"test"}' localhost:21006 trigger.Trigger/Ping
```

### Q: 如何对接 IEC 104 数据？

参考 [IEC 104 消息对接文档](iec104-message.md)。
