# Zero-Service

[![Go Version](https://img.shields.io/github/go-mod/go-version/maomao94/zero-service)](https://go.dev/)
[![Go Zero](https://img.shields.io/badge/go--zero-1.10-00ADD8)](https://github.com/zeromicro/go-zero)
[![License](https://img.shields.io/github/license/maomao94/zero-service)](./LICENSE)

面向工业物联网与边缘集成场景的 Go 微服务集合。项目基于 [go-zero](https://github.com/zeromicro/go-zero) 构建，覆盖工业协议接入、流数据处理、任务调度、实时通信和无人机机场等能力。

Zero-Service 不是一个必须整体部署的单体应用。各服务可以按需独立运行，也可以通过 gRPC、Kafka、MQTT 和 SocketIO 组合成完整业务链路。

[快速开始](#快速开始) · [架构概览](./docs/architecture.md) · [完整文档](./docs/README.md) · [参与贡献](./CONTRIBUTING.md)

## 核心能力

- **IEC 104 数采**：多从站通信，通过 Kafka、MQTT 和 gRPC 并行分发采集数据，并支持 ASDU 合并与时序存储。
- **DJI 云平台接入**：封装 Dock 3 Cloud API，支持航线任务、直播推流和 DRC 指令飞行。
- **异步任务调度**：基于 asynq 与 Redis 提供分布式任务、计划任务以及 HTTP/gRPC 回调。
- **实时通信**：通过 SocketIO 网关完成连接管理、房间广播、服务端推送和 MQTT 桥接。
- **工业协议桥接**：支持 Modbus TCP/RTU、MQTT、Kafka、gRPC-Gateway 及反向隔离装置接入。
- **变电站巡检**：提供 ISP 协议服务端与代理，连接上级平台和下级巡检设备。
- **通用基础能力**：包含地理信息计算、对象存储、文件传输、容器管理、服务发现和可观测性组件。

## 快速开始

### 环境要求

- Go 1.26 或更高版本
- Git
- 与目标服务匹配的外部依赖，例如 Redis、Kafka、数据库或 MQTT Broker

### 获取代码

```bash
git clone https://github.com/maomao94/zero-service.git
cd zero-service
go mod download
```

### 启动服务

各服务均可独立运行。以 Trigger 任务调度服务为例，先根据本地环境调整 Redis 和数据库配置，再启动服务：

```bash
cd app/trigger
go run . -f etc/trigger.yaml
```

配置文件通常位于服务的 `etc/` 目录。更多启动方式、依赖说明和常见问题见[快速开始指南](./docs/quick-start.md)，端口分配见[服务端口清单](./docs/service-ports.md)。

## 架构概览

```text
设备 / 第三方系统 / 前端应用
              |
   +----------+-----------+
   |                      |
协议接入与 BFF       SocketIO 实时网关
   |                      |
   +----------+-----------+
              |
    领域服务与协议桥接服务
              |
   +----------+-----------+
   |          |           |
 Kafka      Redis      数据库 / 对象存储
```

系统以 gRPC 作为主要服务间通信协议，并通过 Kafka、MQTT、SocketIO 等通道适配不同实时性和吞吐需求。详细的模块依赖与数据流见[架构文档](./docs/architecture.md)。

## 核心服务

| 服务 | 主要职责 | 相关文档 |
| --- | --- | --- |
| `ieccaller` / `iecstash` / `streamevent` | IEC 104 采集、消息分发、数据合并与落库 | [IEC 104 数采平台](./docs/iec104.md) |
| `trigger` | 异步任务与计划任务调度 | [Trigger 服务](./docs/trigger.md) |
| `djicloud` | DJI Dock 3 云平台接入 | [DJI 云平台](./docs/djicloud.md) |
| `socketgtw` / `socketpush` | SocketIO 连接管理与服务端推送 | [SocketIO 实时通信](./docs/socketio.md) |
| `bridge*` | Modbus、MQTT、Kafka 和网关协议桥接 | [服务端口清单](./docs/service-ports.md) |
| `ispagent` / `ispserver` | 变电站 ISP 巡检协议代理与服务端 | [ISP 巡检协议](./docs/isp.md) |
| `file` | 分片文件传输与对象存储集成 | [服务端口清单](./docs/service-ports.md) |
| `gis` | H3、GeoHash、电子围栏和坐标转换 | [服务端口清单](./docs/service-ports.md) |
| `podengine` | Docker 容器生命周期管理 | [服务端口清单](./docs/service-ports.md) |

## 仓库结构

```text
app/         核心业务与协议服务
aiapp/       AI 应用与模型接入服务
socketapp/   SocketIO 网关与推送服务
common/      跨服务复用的公共组件
facade/      跨服务协议与数据契约
model/       数据模型及生成脚本
deploy/      Docker Compose 与部署配置
docs/        架构、对接和开发文档
```

## 技术栈

| 类别 | 技术 |
| --- | --- |
| 服务框架 | Go、go-zero、gRPC、gRPC-Gateway |
| 消息与任务 | Kafka、MQTT、asynq、Redis、SocketIO |
| 工业与设备协议 | IEC 60870-5-104、Modbus、ISP、DJI Cloud API |
| 数据与存储 | MySQL、PostgreSQL、SQLite、TDengine、MinIO、OSS |
| 服务治理 | Nacos、OpenTelemetry、Prometheus、Docker |

## 文档

| 文档 | 内容 |
| --- | --- |
| [文档索引](./docs/README.md) | 按用户、对接方和开发者分类的完整导航 |
| [快速开始](./docs/quick-start.md) | 环境准备、服务启动和常见问题 |
| [架构概览](./docs/architecture.md) | 系统分层、核心数据流和技术选型 |
| [开发指南](./docs/development.md) | 代码生成、模块扩展和调试方式 |
| [部署指南](./docs/deployment.md) | Docker、单服务与集群部署 |
| [错误码规范](./docs/error-codes.md) | HTTP/gRPC 状态映射和业务错误编码 |

## 参与贡献

欢迎通过 [Issues](https://github.com/maomao94/zero-service/issues) 反馈问题或提出建议。提交代码前，请先阅读[贡献指南](./CONTRIBUTING.md)。

## 开源许可

本项目基于 [MIT License](./LICENSE) 开源。
