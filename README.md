# Zero-Service

基于 [go-zero](https://github.com/zeromicro/go-zero) 的工业级微服务脚手架，面向物联网数采、异步任务调度、实时通信等场景，提供开箱即用的多协议接入和高性能数据处理能力。

## 特性

- **多协议接入** -- IEC 60870-5-104 / Modbus TCP/RTU / MQTT / gRPC / HTTP，覆盖电力、工业自动化、物联网等场景
- **数采平台** -- 完整的 IEC 104 主站实现，支持 Kafka/MQTT/gRPC 三协议并行推送，内嵌 SQLite 轻量化配置管理
- **异步任务调度** -- 基于 asynq 的分布式任务队列 + 自研计划任务管理引擎，支持 HTTP/gRPC 回调
- **实时通信** -- SocketIO 消息网关，支持房间管理、广播推送、MQTT 桥接和 Token 鉴权
- **容器管理** -- Docker 容器生命周期管理，提供 Kubernetes-like 的 Pod 抽象接口
- **地理信息** -- H3 网格、GeoHash 编解码、电子围栏、坐标系转换
- **BFF 网关** -- 统一的 API 入口，聚合 gRPC 后端服务并提供 grpc-gateway HTTP 访问

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
    |  gRPC Service Mesh |        |   MQTT Broker     |
    +----+----+----+----+        +-------------------+
         |    |    |
   +-----+ +--+--+ +------+
   |       |      |        |
+--v--+ +--v--+ +-v----+ +-v---------+
|trig | |file | |alarm | |bridgeXxx  |
|ger  | |     | |      | |modbus/mqtt|
+-----+ +-----+ +------+ +-----------+
                                |
        +--------+---------+----+
        |        |         |
   +----v---+ +--v----+ +-v--------+
   |ieccaller| |iecstash| |streamevent|
   |IEC 104 | |Kafka消费| |数据落库   |
   +----+---+ +--+----+ +----+-----+
        |        |            |
   +----v--------v------------v----+
   |       Kafka / Redis / DB      |
   |  TDengine / OSS / SQLite      |
   +-------------------------------+
```

### 数采平台架构

<div align="center">
  <img src="docs/images/iec-architecture.png" alt="IEC 104 数采平台架构图" style="max-width: 80%; height: auto;" />
</div>

## 项目结构

```
zero-service/
├── app/                          # 核心微服务
│   ├── ieccaller/                # IEC 104 主站 - 多从站通信、三协议推送
│   ├── iecstash/                 # IEC 104 数据合并 - Kafka 消费、ASDU 压缩
│   ├── iecagent/                 # IEC 104 代理管理
│   ├── trigger/                  # 异步任务调度 + 计划任务管理
│   ├── file/                     # 文件服务 - 分片流上传、OSS 集成
│   ├── gis/                      # 地理信息 - H3/GeoHash/围栏/坐标转换
│   ├── alarm/                    # 告警服务 - 多级告警、钉钉/飞书集成
│   ├── podengine/                # 容器管理 - Docker 容器生命周期管理
│   ├── bridgemodbus/             # Modbus TCP/RTU 协议桥接
│   ├── bridgemqtt/               # MQTT 协议桥接
│   ├── bridgegtw/                # HTTP 代理转发网关
│   ├── bridgedump/               # 南瑞反向隔离装置文件生成
│   ├── lalhook/                  # LAL 流媒体回调服务
│   ├── lalproxy/                 # LAL 代理服务
│   ├── logdump/                  # 日志导出服务
│   ├── xfusionmock/              # 融合模拟服务
│   └── mcpserver/                # MCP 服务器
├── socketapp/                    # 实时通信模块
│   ├── socketgtw/                # SocketIO 网关 - 连接管理、房间、消息路由
│   └── socketpush/               # SocketIO 推送 - Token 生成、gRPC 推送接口
├── gtw/                          # BFF 网关 - HTTP/gRPC 聚合入口
├── facade/                       # 对外接口层
│   └── streamevent/              # 统一流数据事件协议（跨语言 gRPC）
├── common/                       # 公共组件库
│   ├── iec104/                   # IEC 104 协议完整实现
│   ├── socketiox/                # SocketIO 服务器封装
│   ├── asynqx/                   # asynq 任务队列扩展
│   ├── nacosx/                   # Nacos 服务注册/发现
│   ├── modbusx/                  # Modbus 协议扩展
│   ├── mqttx/                    # MQTT 协议扩展
│   ├── ossx/                     # 对象存储（MinIO/阿里OSS/腾讯COS）
│   ├── dbx/                      # 数据库扩展（多库支持）
│   ├── gisx/                     # GIS 地理信息处理
│   ├── dockerx/                  # Docker 操作封装
│   ├── imagex/                   # 图像处理和 EXIF 提取
│   ├── tool/                     # 通用工具函数
│   ├── Interceptor/              # gRPC 拦截器
│   └── ...                       # 更多组件
├── model/                        # 数据库模型和 SQL 脚本
├── deploy/                       # Docker Compose 编排配置
├── docs/                         # 详细文档
├── swagger/                      # Swagger API 文档
├── third_party/                  # 第三方 Proto 定义
└── util/                         # 工具集
```

## 核心服务

### IEC 104 数采平台

完整的 IEC 60870-5-104 数据采集解决方案，由三个服务组件协同工作：

| 服务 | 职责 | 关键能力 |
|------|------|----------|
| **ieccaller** | IEC 104 主站 | 多从站并行通信、Kafka/MQTT/gRPC 三协议推送、内嵌 SQLite 动态配置、弱校验模式 |
| **iecstash** | 数据合并 | Kafka 消费、ASDU 压缩合并、Chunk 批量处理、下游 RPC 转发 |
| **streamevent** | 数据落库 | gRPC 接收、点位配置管理、TDengine 时序存储、多协议消息聚合 |

**数据流**：
```
IEC 104 从站 --> ieccaller --> Kafka --> iecstash --> streamevent --> TDengine
                          |-> MQTT --> 自定义系统
                          |-> gRPC --> streamevent --> TDengine
```

支持 12 种 ASDU 信息体类型：单点遥信、双点遥信、标度化遥测值、短浮点数遥测值、累计量等。

详细文档：[IEC 104 数采平台](./docs/iec104.md) | [IEC 104 消息对接文档](./docs/iec104-protocol.md)

### Trigger 异步任务调度

提供两种核心业务模式：

**1. 异步任务调度（基于 asynq）**
- 分布式任务队列，Redis 存储
- 支持定时/延时任务
- HTTP POST JSON 和 gRPC 两种回调方式
- 自动重试、归档、删除等生命周期管理
- 任务历史统计和仪表板

<div align="center">
  <img src="docs/images/trigger-flow.png" alt="Trigger 异步任务回调流程" style="max-width: 80%; height: auto;" />
</div>

**2. 计划任务管理（自研引擎）**
- 基于数据库扫描的定时巡检调度
- Plan -> Batch -> ExecItem 三级模型
- 完整的状态机：WAITING -> RUNNING -> COMPLETED/FAILED/DELAYED/ONGOING/TERMINATED
- 分布式锁防重、执行日志追踪、批次/计划自动状态聚合

详细文档：[Trigger 服务架构](./docs/trigger.md)

### SocketIO 实时通信

由 socketgtw + socketpush 两个服务组成：

| 服务 | 职责 |
|------|------|
| **socketgtw** | 网关服务 -- 客户端连接管理、房间管理、消息路由、Token 认证 |
| **socketpush** | 推送服务 -- Token 生成/验证、gRPC 推送接口、后端服务调用入口 |

核心能力：
- 房间加入/离开/广播、全局广播
- 单播/批量推送（按 Session 或 Metadata 寻址）
- 会话剔除和元数据管理
- MQTT 桥接 -- 将 MQTT Topic 映射到 SocketIO Room，支持事件映射配置
- 统计信息推送和房间加载错误检测

前端对接文档：[SocketIO 消息网关客户端对接文档](./docs/socketiox-documentation.md)

### 其他服务

| 服务 | 描述 |
|------|------|
| **file** | 文件服务 -- gRPC 分片流上传、OSS 集成（MinIO/阿里OSS/腾讯COS）、视频流捕获 |
| **gis** | 地理信息服务 -- H3 网格编解码、GeoHash、电子围栏生成和检测、坐标系转换（WGS84/GCJ02/BD09） |
| **alarm** | 告警服务 -- 多级告警（P0-P3）、钉钉/飞书通知集成 |
| **podengine** | 容器管理 -- Docker 容器 CRUD、Pod 抽象模型、资源统计（CPU/内存/网络/存储）、镜像管理 |
| **bridgemodbus** | Modbus TCP/RTU 协议桥接 -- 线圈和寄存器读写、设备配置管理、gRPC 集成 |
| **bridgemqtt** | MQTT 协议桥接 -- 消息发布/订阅、带追踪的推送、gRPC 集成 |
| **bridgegtw** | HTTP 代理转发网关 -- 多后端负载均衡、请求路由 |
| **bridgedump** | 南瑞反向隔离装置文件生成 -- 文本文件生成、Filebeat 集成、Kafka 分类发送 |
| **lalhook** | LAL 流媒体回调 -- TS 录制回调、推流/拉流事件处理、分片播放 |
| **logdump** | 日志导出服务 |

### BFF 网关 (gtw)

项目的统一 API 入口，功能包括：
- gRPC 服务聚合，同时支持 grpc-gateway 提供 HTTP 访问
- 用户认证（JWT）、微信支付回调、短信验证码
- 文件上传（单文件/分片/流式）、文件下载
- CORS 跨域支持

### 对外接口层 (facade/streamevent)

统一的跨语言流数据事件协议，基于 gRPC 定义，支持：
- MQTT / WebSocket / Kafka 消息接收
- IEC 104 ASDU 消息推送（PushChunkAsdu）
- Socket 上行消息处理
- 计划任务事件处理和通知

任何语言实现 `streamevent.proto` 即可与数采平台交互。

## 技术栈

| 类别 | 技术 |
|------|------|
| **微服务框架** | go-zero |
| **RPC** | gRPC + grpc-gateway + Protocol Buffers |
| **消息队列** | Kafka (go-queue) |
| **任务队列** | asynq + Redis |
| **实时通信** | SocketIO (fork of socket.io-golang) |
| **工业协议** | IEC 60870-5-104 (go-iecp5) / Modbus (grid-x/modbus) / MQTT (paho.mqtt) |
| **关系数据库** | MySQL / PostgreSQL / SQLite |
| **时序数据库** | TDengine |
| **对象存储** | MinIO / 阿里 OSS / 腾讯 COS |
| **服务发现** | Nacos |
| **地理计算** | H3 (uber/h3-go) / GeoHash / orb / go-geom |
| **容器管理** | Docker SDK |
| **监控追踪** | OpenTelemetry / Prometheus |
| **容器编排** | Docker Compose / Kubernetes (可选) |

## 快速开始

### 环境要求

- Go 1.25+
- Redis（任务队列和缓存）
- 可选：Kafka / MySQL / PostgreSQL / TDengine / Docker

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

### 配置

各服务配置文件位于 `app/{service}/etc/` 目录下。典型配置项包括：
- 服务监听地址和端口
- Redis / Kafka / 数据库连接
- Nacos 服务注册配置
- 协议特定配置（IEC 104 从站列表、MQTT Broker 等）

## 开发指南

### 新增服务流程

1. 在 `app/` 下创建服务目录
2. 编写 `.proto` 文件定义服务接口
3. 运行 `gen.sh` 生成代码框架
4. 在 `internal/logic/` 实现业务逻辑
5. 在 `etc/` 下创建配置文件
6. 编写入口 `main` 文件启动服务

### 代码生成

```bash
# 进入服务目录
cd app/{service}

# 执行代码生成
./gen.sh
```

项目提供数据库模型生成脚本：
- `model/genModel.sh` -- 通用模型生成
- `model/genPgModel.sh` -- PostgreSQL 专用
- `model/genModelSql.sh` -- SQL 脚本生成

### gRPC API

各服务的 Proto 文件即 API 定义，位于 `app/{service}/{service}.proto`，Swagger 文档位于 `swagger/` 目录：
- `trigger.swagger.json`
- `podengine.swagger.json`
- `streamevent.swagger.json`
- 更多见 swagger 目录

### 错误码规范

项目遵循 `google.rpc.Code` 错误码标准，HTTP 和 gRPC 错误码映射关系参见 [code.md](./code.md)。

## 部署

### Docker 部署

```bash
cd deploy
# 按需修改 docker-compose.yml
docker-compose up -d
```

Docker Compose 默认包含：Kafka、Filebeat、ieccaller、bridgegtw、bridgedump 等核心服务。

各服务目录下提供独立的 `Dockerfile`，支持单独构建：

```bash
cd app/{service}
docker build -t zero-service/{service}:latest .
```

### 集群部署

- **服务发现**：通过 Nacos 实现服务注册与发现
- **负载均衡**：Nginx / gRPC 内置负载均衡
- **高可用**：Redis Cluster + Kafka 集群 + 数据库主从
- **监控**：OpenTelemetry -> Prometheus -> Grafana

## 文档

| 文档 | 描述 |
|------|------|
| [IEC 104 数采平台](./docs/iec104.md) | 数采平台架构设计、服务组件、数据流、TDengine 表结构、配置管理 |
| [IEC 104 消息对接文档](./docs/iec104-protocol.md) | IEC 104 协议详细对接规范、ASDU 类型、消息格式、多协议推送配置 |
| [Trigger 服务架构](./docs/trigger.md) | 异步任务调度和计划任务管理的架构设计、API、状态流转、部署 |
| [SocketIO 客户端对接文档](./docs/socketiox-documentation.md) | 前端 SocketIO 对接指南、事件体系、数据结构、MQTT 桥接、鉴权 |
| [KML/KMZ 文件指南](./docs/kml-kmz-guide.md) | 无人机航点任务 KML 文件结构和使用指南 |
| [错误码规范](./code.md) | google.rpc.Code 错误码映射和使用规范 |

## 许可证

[MIT License](LICENSE)

Copyright (c) 2026 zero-service

## 链接

- GitHub: [https://github.com/maomao94/zero-service](https://github.com/maomao94/zero-service)
- Issues: [https://github.com/maomao94/zero-service/issues](https://github.com/maomao94/zero-service/issues)
- go-zero: [https://github.com/zeromicro/go-zero](https://github.com/zeromicro/go-zero)
- asynq: [https://github.com/hibiken/asynq](https://github.com/hibiken/asynq)
- go-iecp5: [https://github.com/wendy512/iec104](https://github.com/wendy512/iec104)
