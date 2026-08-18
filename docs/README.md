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
| [错误码规范](./error-codes.md) | HTTP/gRPC 状态码映射与 `detail.reason` 编码 |

## 核心服务

| 服务 | 文档 | 内容 |
| --- | --- | --- |
| IEC 104 | [数采平台](./iec104/README.md) | 采集、消息分发、数据合并与控制命令 |
| Trigger | [Trigger 服务](./trigger/README.md) | 异步任务、计划任务、RRULE CronJob |
| SocketIO | [实时通信](./socketio/README.md) | 网关对接、事件体系、房间广播和鉴权 |
| DJI | [云平台](./djicloud/README.md) | Dock 3 Cloud API 与航点任务文件 |
| ISP | [巡检协议](./isp/README.md) | ISP 服务端/代理、帧格式、任务和模型同步 |
| LAL | [流媒体回调](../app/lalhook/README.md) | LAL HTTP 回调事件、鉴权、配置和接口说明 |
| File | [文件与对象存储](./file/README.md) | OSS 配置管理、文件上传/中继、签名 URL 与视频截帧 |
| GIS | [地理信息服务](./gis/README.md) | H3、GeoHash、电子围栏与坐标转换 |
| PodEngine | [容器编排](./podengine/README.md) | Docker 容器生命周期管理 |
| Bridge | [协议桥接](./bridge/README.md) | HTTP/Kafka/Modbus/MQTT 桥接与报文落地 |

## 开发者

| 文档 | 内容 |
| --- | --- |
| [开发指南](./development.md) | 环境搭建、代码生成、模块扩展和调试技巧 |
| [部署指南](./deployment.md) | Docker、单服务和集群部署、配置管理 |
| [antsx 与响应式模式](./antsx-vs-reactive.md) | antsx Promise/Invoke 与响应式编排模式对比 |
| [Superpowers 技能全量介绍](./superpowers.md) | AI 编程技能集：14 个技能的设计哲学、铁律与使用流程 |

## 架构决策

| 决策 | 内容 |
| --- | --- |
| [ADR-0001 使用 antsx 表达并发与异步编排](./adr/0001-use-antsx-for-concurrency.md) | 用 Go goroutine/channel/泛型实现并发原语，不引入响应式框架 |
| [ADR-0002 IEC 104 采集数据采用三通道并行分发](./adr/0002-iec104-three-channel-distribution.md) | Kafka/MQTT/gRPC 三通道并行推送 |
