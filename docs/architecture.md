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

## 模块依赖

### 分层结构

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
│  ieccaller / trigger / djicloud /   │
│  file / gis / alarm / podengine /   │
│  bridgemodbus / bridgemqtt / ...    │
└─────────────────┬───────────────────┘
                  │
┌─────────────────v───────────────────┐
│         common/ 公共组件库           │
│  iec104 / djisdk / socketiox /      │
│  asynqx / mqttx / ossx / dbx / ... │
└─────────────────┬───────────────────┘
                  │
┌─────────────────v───────────────────┐
│         基础设施                     │
│  Kafka / Redis / MySQL / TDengine / │
│  MQTT Broker / Nacos / Docker       │
└─────────────────────────────────────┘
```

### 核心数据流

**数采平台**：
```
IEC 104 从站 --> ieccaller --> Kafka --> iecstash --> streamevent --> TDengine
                          |-> MQTT --> 自定义系统
                          |-> gRPC --> streamevent --> TDengine
```

**DJI 云平台**：
```
业务系统 --> djicloud gRPC --> common/djisdk --> MQTT Broker --> DJI Dock3/飞行器
                         ^              |
                         |              v
                  services_reply / events / osd / state / drc/up
```

**实时通信**：
```
前端客户端 ──WebSocket──> socketgtw ──gRPC──> StreamEvent（业务处理）
                                  <──gRPC──
后端服务 ──gRPC──> socketpush ──gRPC──> socketgtw ──WebSocket──> 前端客户端
```

## 技术选型

| 类别 | 技术 | 选型理由 |
|------|------|----------|
| 微服务框架 | go-zero | Go 生态成熟、内置服务治理、代码生成完善 |
| RPC | gRPC + grpc-gateway | 高性能、跨语言、HTTP 兼容 |
| 消息队列 | Kafka | 高吞吐、持久化、多消费组 |
| 任务队列 | asynq + Redis | 分布式、可靠、支持延时/定时任务 |
| 实时通信 | SocketIO | 浏览器原生支持、双向通信 |
| 工业协议 | IEC 104 / Modbus / MQTT | 覆盖电力、工业自动化、物联网场景 |
| 时序数据库 | TDengine | 高性能时序数据存储 |
| 对象存储 | MinIO / 阿里 OSS / 腾讯 COS | 多云兼容 |
| 服务发现 | Nacos | 配置中心 + 服务注册一体化 |
| 容器管理 | Docker SDK | 原生容器操作能力 |
