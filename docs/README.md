# 文档索引

这里集中放置 Zero-Service 的用户、对接方和开发者文档。项目由多个可以独立运行的服务组成，请先根据使用场景选择服务，再阅读对应的配置和部署说明。

## 推荐阅读路径

1. 从[快速开始](./quick-start.md)准备环境并启动一个服务。
2. 通过[架构概览](./architecture.md)了解服务分层和主要数据流。
3. 查看[服务端口清单](./service-ports.md)确认服务入口。
4. 根据协议或业务场景进入下方的专项文档。

## 用户与对接方

| 文档 | 内容 |
| --- | --- |
| [快速开始](./quick-start.md) | 环境要求、安装、启动示例、常见问题 |
| [架构概览](./architecture.md) | 系统分层、模块依赖、数据流和技术选型 |
| [服务端口清单](./service-ports.md) | 各服务默认端口、协议和用途 |
| [错误码规范](./error-codes.md) | 错误码维度、模块分配、HTTP/gRPC 映射与网关返回契约 |
| [数据库时区配置](./database-timezone.md) | Kingbase/openGauss/PostgreSQL/MySQL 四库时区设置、会话行为与重启持久性 |

## 核心服务

| 服务 | 文档 | 内容 |
| --- | --- | --- |
| IEC 104 | [数采平台](./iec104/README.md) | 采集、消息分发、数据合并与控制命令 |
| Trigger | [Trigger 服务](./trigger/README.md) | 异步任务、计划任务、RRULE CronJob |
| SocketIO | [实时通信](./socketio/README.md) | 网关对接、事件体系、房间广播和鉴权 |
| DJI | [云平台](./djicloud/README.md) | 第一代上云 API 参考实现（Dock 3 Cloud API 与航点任务文件）；新项目建议[上云 API 2](https://developer.dji.com/cn/cloud-api-2) |
| ISP | [巡检协议](./isp/README.md) | ISP 服务端/代理、帧格式、任务和模型同步 |
| LiveKit | [视频会议](./live/README.md) | 会议管理、入会 token、票据邀请、Webhook、聊天消息；[LiveKit](https://github.com/livekit/livekit) · [Server SDK](https://github.com/livekit/server-sdk-go) · [对接指南](./live/integration-guide.md) |
| Oryx | [流媒体](./oryx/README.md) | HTTP 回调网关、gRPC API 代理、录制管理与中继拉流；[Oryx](https://github.com/ossrs/oryx) · [SRS](https://github.com/ossrs/srs) |
| LAL | [流媒体回调](../app/lalhook/README.md) | LAL HTTP 回调事件与 gRPC 代理（已不推荐，迁移至 Oryx）；[LAL](https://github.com/q191201771/lal) |
| File | [文件与对象存储](./file/README.md) | OSS 配置管理、文件上传/中继、签名 URL 与视频截帧 |
| GIS | [地理信息服务](./gis/README.md) | H3、GeoHash、电子围栏与坐标转换 |
| PodEngine | [容器编排](./podengine/README.md) | Docker 容器生命周期管理 |
| OpenAPI | [开放 API 网关](./open-api/README.md) | Basic 鉴权、统一信封、method 路由、Podman + Modbus |
| Bridge | [协议桥接](./bridge/README.md) | HTTP/Kafka/Modbus/MQTT 桥接与报文落地 |

## Web 前端

| 站点 | 文档 | 内容 |
| --- | --- | --- |
| Zero 工作台 | [前端工作台](../web/workspace/README.md) | 统一入口、健康检测、端点速查、GitHub 与命令面板 |
| Live 会议 | [Live 前端](../web/live/README.md) | 会议大厅、会议室、录制、邀请与本地 Token |
| SocketIO 测试 | [SocketIO 前端](../web/socketio/README.md) | 连接、消息、房间、事件与底部控制台 |
| OpenAPI 控制台 | [OpenAPI 前端](../web/openapi/README.md) | 开放 API 调试、应用管理与错误码速查 |

四个项目的构建、部署与验证状态见 [`web/COMPLETION.md`](../web/COMPLETION.md)。

## 开发者

| 文档 | 内容 |
| --- | --- |
| [开发指南](./development.md) | 环境搭建、代码生成、模块扩展和调试技巧 |
| [网关标准开发流程](./gateway-standard.md) | 网关分层、路由鉴权、统一信封、错误码出口与新增接口验收 |
| [部署指南](./deployment.md) | Docker、单服务和集群部署、配置管理 |
| [antsx 并发原语指南](./antsx-vs-reactive.md) | antsx 核心 API、使用场景与最佳实践 |
| [Superpowers 技能全量介绍](./superpowers.md) | AI 编程技能集：14 个技能的设计哲学、铁律与使用流程 |
| [Matt Pocock 技能全量介绍](./matt-pocock-skills.md) | AI 编程技能集：37 个技能的工程实践、调试诊断与需求拆分 |


