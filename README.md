# Zero-Service

基于 [go-zero](https://github.com/zeromicro/go-zero) 的工业级微服务脚手架，面向物联网数采、异步任务调度、实时通信、无人机机场接入等场景，提供开箱即用的多协议接入和高性能数据处理能力。

## 特性

- **多协议接入** -- IEC 60870-5-104 / Modbus TCP/RTU / MQTT / gRPC / HTTP，覆盖电力、工业自动化、物联网场景
- **数采平台** -- 完整的 IEC 104 主站实现，支持 Kafka/MQTT/gRPC 三协议并行推送，内嵌 SQLite 轻量化配置管理
- **大疆云平台** -- DJI Dock3 Cloud API MQTT 封装，支持直播、航线、远程调试、DRC 指令飞行
- **异步任务调度** -- 基于 asynq 的分布式任务队列 + 自研计划任务管理引擎，支持 HTTP/gRPC 回调
- **实时通信** -- SocketIO 消息网关，支持房间管理、广播推送、MQTT 桥接和 Token 鉴权
- **容器管理** -- Docker 容器生命周期管理，提供 Kubernetes-like 的 Pod 抽象接口
- **地理信息** -- H3 网格、GeoHash 编解码、电子围栏、坐标系转换
- **响应式工具包** -- [antsx](common/antsx/README.md)：Stream/Promise/Invoke/EventEmitter，Go 原生的流式与异步编排

## 快速开始

### 环境要求

- Go 1.25+
- Redis（任务队列、缓存）
- 可选：Kafka / MySQL / PostgreSQL / TDengine / Docker / MQTT Broker / Nacos

### 安装

```bash
git clone https://github.com/maomao94/zero-service.git
cd zero-service
go mod tidy
```

### 启动服务

```bash
# 启动单个服务（以 trigger 为例）
cd app/trigger
go run trigger.go -f etc/trigger.yaml

# 或使用 Docker Compose 启动
cd deploy
docker-compose up -d
```

详细指南：[快速开始](docs/quick-start.md)

## 架构

```
                    +-----------------+
                    |   Frontend/App  |
                    +--------+--------+
                             |
              +--------------+--------------+
              |                             |
     +--------v--------+         +---------v---------+
     |   gtw (BFF)     |         | socketgtw/push    |
     | HTTP + gRPC-GW  |         | SocketIO 实时通信  |
     +--------+--------+         +---------+---------+
              |                             |
    +---------+---------+         +---------+---------+
    |  gRPC Service Mesh |
    +----+----+----+----+
         |    |    |
   +-----+ +--+--+ +------+ +----------+ +----------+
   |       |      |        |          | |          |
+--v--+ +--v--+ +-v----+ +-v-------+ +-v-----+
|trig | |file | |alarm | |bridgeXxx| |djicloud|
|ger  | |     | |      | |modbus/mq| |DJI平台|
+-----+ +-----+ +------+ +----+----+ +---+----+
                                |         |
        +--------+---------+----+         v
        |        |         |        DJI Cloud API
   +----v---+ +--v----+ +-v--------+    MQTT
   |ieccaller| |iecstash| |streamevent|
   |IEC 104 | |Kafka消费| |数据落库   |
   +----+---+ +--+----+ +----+-----+
        |        |            |
   +----v--------v------------v----+
   |       Kafka / Redis / DB      |
   |  TDengine / OSS / SQLite      |
   +-------------------------------+
```

详细架构：[架构概览](docs/architecture.md)

## 核心服务

| 服务 | 说明 | 文档 |
|------|------|------|
| **ieccaller** | IEC 104 主站 - 多从站通信、三协议推送 | [IEC 104 数采平台](docs/iec104.md) |
| **iecstash** | IEC 104 数据合并 - Kafka 消费、ASDU 压缩 | [IEC 104 数采平台](docs/iec104.md) |
| **streamevent** | 统一流数据事件协议 - 跨语言 gRPC | [IEC 104 数采平台](docs/iec104.md) |
| **trigger** | 异步任务调度 + 计划任务管理 | [Trigger 服务](docs/trigger.md) |
| **djicloud** | DJI 云平台服务 - Dock3 Cloud API | [DJI 云平台](docs/djicloud.md) |
| **socketgtw/push** | SocketIO 实时通信网关 | [SocketIO 文档](docs/socketio.md) |
| **gtw** | BFF 网关 - HTTP/gRPC 聚合入口 | - |
| **file** | 文件服务 - 分片流上传、OSS 集成 | - |
| **gis** | 地理信息 - H3/GeoHash/围栏/坐标转换 | - |
| **alarm** | 告警服务 - 多级告警、钉钉/飞书集成 | - |
| **podengine** | 容器管理 - Docker 容器生命周期 | - |
| **bridgemodbus** | Modbus TCP/RTU 协议桥接 | - |
| **bridgemqtt** | MQTT 协议桥接 | - |

## 技术栈

| 类别 | 技术 |
|------|------|
| 微服务框架 | go-zero |
| RPC | gRPC + grpc-gateway + Protocol Buffers |
| 消息队列 | Kafka (go-queue) |
| 任务队列 | asynq + Redis |
| 实时通信 | SocketIO / SSE |
| 响应式工具 | antsx（Stream/Promise/Invoke/EventEmitter） |
| 工业协议 | IEC 60870-5-104 / Modbus / MQTT / DJI Cloud API |
| 关系数据库 | MySQL / PostgreSQL / SQLite |
| 时序数据库 | TDengine |
| 对象存储 | MinIO / 阿里 OSS / 腾讯 COS |
| 服务发现 | Nacos |
| 容器管理 | Docker SDK |

## 文档

| 文档 | 说明 |
|------|------|
| [文档索引](docs/README.md) | 全部文档导航 |
| [快速开始](docs/quick-start.md) | 环境要求、安装、启动示例 |
| [架构概览](docs/architecture.md) | 系统架构、模块依赖、数据流 |
| [服务端口清单](docs/service-ports.md) | 各服务默认端口 |
| [错误码规范](docs/error-codes.md) | HTTP/gRPC 错误码映射 |
| [antsx 工具包](common/antsx/README.md) | Stream/Promise/Invoke/EventEmitter 响应式工具 |
| [为什么选 antsx](docs/antsx-vs-reactive.md) | 与 Java WebFlux/RxJava 的对比分析 |
| [开发指南](docs/development.md) | 环境搭建、代码生成、调试技巧 |
| [部署指南](docs/deployment.md) | Docker/集群部署、配置管理 |

## 参与贡献

欢迎贡献代码！请先阅读 [贡献指南](CONTRIBUTING.md)。

## 许可证

[MIT License](LICENSE)

## 链接

- GitHub: [https://github.com/maomao94/zero-service](https://github.com/maomao94/zero-service)
- Issues: [https://github.com/maomao94/zero-service/issues](https://github.com/maomao94/zero-service/issues)
- go-zero: [https://github.com/zeromicro/go-zero](https://github.com/zeromicro/go-zero)
