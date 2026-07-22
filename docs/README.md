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

| 文档 | 内容 |
| --- | --- |
| [IEC 104 数采平台](./iec104.md) | 数采平台架构、服务组件、数据流和配置管理 |
| [IEC 104 消息对接](./iec104-message.md) | 消息格式、ASDU 类型、信息体结构和消费指南 |
| [IEC 104 控制命令](./iec104-command.md) | 控制命令接口、响应机制、错误码和 TypeId 对照表 |
| [Trigger 服务](./trigger.md) | 异步任务、计划任务、API 和状态流转 |
| [SocketIO 实时通信](./socketio.md) | 网关对接、事件体系、房间广播和鉴权 |
| [DJI 云平台](./djicloud.md) | DJI Dock 3 Cloud API、RPC 接口和配置说明 |
| [ISP 巡检协议](./isp.md) | ISP 服务端/代理、帧格式、任务和模型同步 |
| [LAL 流媒体回调](../app/lalhook/README.md) | LAL HTTP 回调事件、鉴权、配置和接口说明 |

## 开发者

| 文档 | 内容 |
| --- | --- |
| [开发指南](./development.md) | 环境搭建、代码生成、模块扩展和调试技巧 |
| [部署指南](./deployment.md) | Docker、单服务和集群部署、配置管理 |
| [KML/KMZ 指南](./kml-kmz-guide.md) | 无人机航点任务 KML/KMZ 文件结构 |
| [antsx 与响应式模式](./antsx-vs-reactive.md) | antsx Promise/Invoke 与响应式编排模式对比 |
