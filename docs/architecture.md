# 架构概览

## 系统架构

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

## 分层结构

```
┌─────────────────────────────────────┐
│           前端 / 客户端              │
└─────────────────┬───────────────────┘
                  │
┌─────────────────v───────────────────┐
│         gtw (BFF 网关)              │
│    HTTP + gRPC-Gateway 入口         │
└─────────────────┬───────────────────┘
                  │
┌─────────────────v───────────────────┐
│         gRPC Service Mesh           │
│  ieccaller / iecstash / trigger /   │
│  djicloud / file / gis / podengine / │
│  bridgemodbus / bridgemqtt / ...     │
└─────────────────┬───────────────────┘
                  │
┌─────────────────v───────────────────┐
│         common/ 公共组件库           │
│  iec104 / djisdk / socketiox /      │
│  asynqx / mqttx / ossx / ...       │
└─────────────────┬───────────────────┘
                  │
┌─────────────────v───────────────────┐
│           基础设施                   │
│  Kafka / Redis / PostgreSQL /       │
│  TDengine / MQTT Broker / Nacos     │
└─────────────────────────────────────┘
```

## 核心数据流

### 数采平台

```
IEC 104 从站 --> ieccaller --> Kafka --> iecstash --> streamevent --> TDengine
                          |-> MQTT --> 外部系统
                          |-> gRPC (流事件协议) --> streamevent / 消费端
```

`ieccaller` 采集后经 Kafka、MQTT、gRPC 三通道并行推送。`iecstash` 消费 Kafka 数据并批量转发给 `streamevent`，由 `streamevent` 完成点位过滤、分发和 TDengine 落库。流事件协议采用跨语言的 gRPC 定义，也可供其他语言实现端消费。

### DJI 云平台

```
业务系统 --> djicloud gRPC --> common/djisdk --> MQTT Broker --> DJI Dock 3/飞行器
                         ^              |
                         |              v
                  services_reply / events / osd / state / drc/up
```

两条下行通道：`services` 主题（请求-应答）用于航线、直播、调试等标准指令；`drc/down` 主题（即发即忘）用于杆量控制等高频指令。

### 实时通信

```
前端客户端 ──WebSocket──> socketgtw ──gRPC──> 业务服务
                                  <──gRPC──
后端服务 ──gRPC──> socketpush ──gRPC──> socketgtw ──WebSocket──> 前端客户端
```

`socketgtw` 管理 SocketIO 长连接和房间，`socketpush` 提供后端推送接口。两者通过 gRPC 协作完成双向消息路由。

## 技术选型

| 类别 | 技术 | 选型理由 |
|------|------|----------|
| 微服务框架 | go-zero | Go 生态成熟、内置服务治理、代码生成完善 |
| RPC | gRPC + grpc-gateway | 高性能、跨语言、HTTP 兼容 |
| 消息队列 | Kafka | 高吞吐、持久化、多消费组 |
| 任务队列 | asynq + Redis | 分布式、可靠、延时/定时任务 |
| 实时通信 | SocketIO | 浏览器原生支持、双向通信 |
| 工业协议 | IEC 104 / Modbus / MQTT | 覆盖电力、工业自动化、物联网 |
| 时序数据库 | TDengine | 高性能时序数据存储写入 |
| 对象存储 | MinIO / 阿里 OSS / 腾讯 COS | 多云兼容 |
| 服务发现 | Nacos | 配置中心 + 服务注册一体化 |
| 容器管理 | Docker SDK | 原生容器操作能力 |

> 服务端口分配见[服务端口清单](./service-ports.md)。
