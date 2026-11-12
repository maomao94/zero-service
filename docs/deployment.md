# 部署指南

本文档覆盖单服务、Docker Compose 和集群部署。仓库中的 Compose 文件包含环境相关的镜像、主机路径和网络配置，部署前必须先完成替换和校验。

> 根目录 `go.mod` 当前要求 Go 1.27。构建镜像前请确认目标服务的 `Dockerfile` 使用了兼容的 Go builder 镜像。

## Docker 部署

### 单服务构建

```bash
cd app/{service}
docker build -t zero-service/{service}:latest .
```

### Docker Compose 部署

```bash
cd deploy

# 按需修改配置
vim docker-compose.yml

# 设置环境变量
export REGISTER=your-registry
export MAIN_TAG=latest

# 校验并启动（Docker Compose v2）
docker compose config
docker compose up -d
```

默认包含的服务：

- Kafka（消息队列）
- Filebeat（日志收集）
- ieccaller（IEC 104 主站）
- iecstash（数据合并）
- bridgegtw（HTTP 代理）
- bridgedump（南瑞隔离装置）
- Kafdrop（Kafka 管理 UI）

> `deploy/docker-compose.yml` 使用 `${REGISTER}`、`${MAIN_TAG}`，并挂载 `/home/root/...` 等宿主机路径。请先修改镜像仓库、Kafka 广播地址、日志/Filebeat 路径和 `network_mode: host` 相关设置。

### 环境变量

| 变量 | 说明 | 示例 |
|------|------|------|
| `REGISTER` | 镜像仓库地址 | `registry.cn-hangzhou.aliyuncs.com/your-ns` |
| `MAIN_TAG` | 镜像标签 | `latest` / `v1.0.0` |

## 单服务部署

### 编译

```bash
cd app/{service}
go build -o {service} .
```

### 运行

```bash
./{service} -f etc/{service}.yaml
```

### systemd 服务

```ini
[Unit]
Description=Zero Service - {service}
After=network.target

[Service]
Type=simple
User=app
WorkingDirectory=/opt/zero-service
ExecStart=/opt/zero-service/{service} -f etc/{service}.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## 集群部署

### 服务发现

通过 Nacos 实现服务注册与发现。服务侧配置键为 `NacosConfig`（见各服务 `internal/config/config.go`），`IsRegister` 为 `false` 时不注册：

```yaml
NacosConfig:
  IsRegister: true
  Host: nacos-server
  Port: 8848
  Username: nacos
  PassWord: <nacos 密码>
  NamespaceId: production
  ServiceName: trigger
```

调用侧则用 `nacos://` 目标串代替直连地址，例如 `Target: nacos://nacos:<密码>@nacos-server:8848/stream-event?namespaceid=production`；resolver 实现见 [`common/nacosx`](../common/nacosx/README.md)。

### 负载均衡

- **HTTP**：Nginx 反向代理
- **gRPC**：go-zero 内置负载均衡 + Nacos 服务发现

### 高可用

- **Redis**：Redis Cluster 或 Sentinel
- **Kafka**：多 Broker 集群
- **数据库**：主从复制

### 监控

- **OpenTelemetry**：分布式追踪
- **Prometheus**：指标采集
- **Grafana**：可视化

## 配置管理

### 配置文件位置

各服务配置文件通常位于：
- `app/{service}/etc/{service}.yaml`
- `gtw/etc/gtw.yaml`

### 典型配置项

gRPC 服务监听 `ListenOn`，HTTP 网关监听 `Host` + `Port`，两者不要混用：

```yaml
Name: trigger.rpc
ListenOn: 0.0.0.0:21006

# Redis
Redis:
  Host: redis-server:6379
  Type: node

# Nacos 注册（可选，IsRegister 为 false 时跳过）
NacosConfig:
  IsRegister: false
  Host: nacos-server
  Port: 8848
  NamespaceId: production
  ServiceName: trigger

# 下游 gRPC 依赖；不同服务的中介件配置键不同（如 iecstash 用 KafkaASDUConfig），以各自 etc/*.yaml 为准
StreamEventConf:
  Endpoints:
    - streamevent:21009
```

> 上例只示意键名与层次；Kafka、MQTT、对象存储等配置键在各服务间并不统一，落地时以该服务 `etc/*.yaml` 与配置结构体为准。

### 配置安全

- 不要提交真实密钥到代码仓库
- 使用环境变量或配置中心管理敏感信息
- 示例配置仅保留占位值

## 端口规划

参考[服务端口清单](./service-ports.md)。

端口规则：
- HTTP 服务：1xxxx
- gRPC 服务：2xxxx
