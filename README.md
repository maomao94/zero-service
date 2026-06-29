# Zero-Service

基于 [go-zero](https://github.com/zeromicro/go-zero) 的工业级微服务集合，覆盖物联网数采、任务调度、实时通信、无人机机场等场景，提供多协议接入和流数据处理能力。

## 场景能力

- **IEC 104 数采平台** — 完整主站实现，Kafka/MQTT/gRPC 三协议并行推送，SQLite 轻量化配置；提供跨语言流数据事件协议（gRPC）
- **DJI 云平台** — Dock3 Cloud API MQTT 封装，航线任务、直播推流、DRC 指令飞行
- **异步任务调度** — asynq 分布式队列 + 自研计划任务引擎，HTTP/gRPC 回调
- **实时通信** — SocketIO 消息网关，房间管理、广播推送、MQTT 桥接
- **协议桥接** — Modbus TCP/RTU、MQTT 协议转换
- **地理信息** — H3 网格、GeoHash 编解码、电子围栏、坐标系转换
- **容器管理** — Docker 容器生命周期管理

## 快速开始

```bash
git clone https://github.com/maomao94/zero-service.git
cd zero-service && go mod tidy
cd app/trigger && go run trigger.go -f etc/trigger.yaml
```

> 环境要求：Go 1.25+、Redis。详细步骤见 [快速开始](docs/quick-start.md)。

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
   +-----+ +--+--+ +------+ +----------+
   |       |      |        | |
+--v--+ +--v--+ +-v-------+ +-v-----+
|trig | |file | |bridgeXxx| |djicloud|
|ger  | |     | |modbus/mq| |DJI平台|
+-----+ +-----+ +----+----+ +---+----+
                         |         |
        +--------+-------+         v
        |        |           DJI Cloud API
   +----v---+ +--v----+        MQTT
   |ieccaller| |iecstash|
   |IEC 104 | |Kafka消费|
   +----+---+ +--+----+
        |        |
   +----v--------v------------+
   |       Kafka / Redis / DB  |
   |  TDengine / OSS / SQLite  |
   +----------------------------+
```

[架构概览](docs/architecture.md) · [服务端口清单](docs/service-ports.md)

## 服务

| 服务 | 说明 | 文档 |
|------|------|------|
| **ieccaller** | IEC 104 主站 — 多从站通信、三协议推送 | [IEC 104 平台](docs/iec104.md) |
| **iecstash** | 数据合并 — Kafka 消费、ASDU 压缩 | [IEC 104 平台](docs/iec104.md) |
| **trigger** | 异步任务调度 + 计划任务管理 | [Trigger](docs/trigger.md) |
| **djicloud** | DJI 云平台 — Dock3 Cloud API | [DJI 云平台](docs/djicloud.md) |
| **socketgtw/push** | SocketIO 实时通信网关 | [SocketIO](docs/socketio.md) |
| **gtw** | BFF 网关 — HTTP/gRPC-Gateway 聚合入口 | - |
| **file** | 文件服务 — 分片流上传、OSS 集成 | - |
| **bridgemodbus** | Modbus TCP/RTU 协议桥接 | - |
| **bridgemqtt** | MQTT 协议桥接 | - |
| **gis** | 地理信息 — H3/GeoHash/围栏/坐标转换 | - |
| **podengine** | 容器管理 — Docker 容器生命周期 | - |

## 技术栈

go-zero · gRPC · Kafka · asynq + Redis · SocketIO · IEC 104 · Modbus · MQTT · DJI Cloud API · MySQL / PostgreSQL / SQLite · TDengine · MinIO / OSS · Nacos · Docker

## 文档

[文档索引](docs/README.md) · [快速开始](docs/quick-start.md) · [架构概览](docs/architecture.md) · [错误码规范](docs/error-codes.md) · [开发指南](docs/development.md) · [部署指南](docs/deployment.md)

## License

[MIT](LICENSE)
