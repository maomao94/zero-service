# 部署指南

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

# 启动
docker-compose up -d
```

默认包含的服务：
- Kafka（消息队列）
- Filebeat（日志收集）
- ieccaller（IEC 104 主站）
- iecstash（数据合并）
- bridgegtw（HTTP 代理）
- bridgedump（南瑞隔离装置）
- Kafdrop（Kafka 管理 UI）

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

通过 Nacos 实现服务注册与发现：

```yaml
# 在服务配置中添加
Nacos:
  Host: nacos-server:8848
  Namespace: production
  Group: DEFAULT_GROUP
  ClusterName: default
```

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
- `aiapp/{service}/etc/{service}.yaml`
- `gtw/etc/gtw.yaml`

### 典型配置项

```yaml
Name: service-name
Host: 0.0.0.0
Port: 21006

# Redis
Redis:
  Host: redis-server:6379
  Type: node

# Kafka
Kafka:
  Brokers:
    - kafka-server:9092

# Nacos
Nacos:
  Host: nacos-server:8848
  Namespace: production
```

### 配置安全

- 不要提交真实密钥到代码仓库
- 使用环境变量或配置中心管理敏感信息
- 示例配置仅保留占位值

## 端口规划

参考 [服务端口清单](service-ports.md)。

端口规则：
- HTTP 服务：1xxxx
- gRPC 服务：2xxxx
