# 文档索引

## 用户/对接方

| 文档 | 说明 |
|------|------|
| [快速开始](quick-start.md) | 环境要求、安装、启动示例、常见问题 |
| [架构概览](architecture.md) | 系统架构、模块依赖、数据流、技术选型 |
| [服务端口清单](service-ports.md) | 各微服务默认端口、协议和用途 |
| [错误码规范](error-codes.md) | HTTP/gRPC 状态码映射、自定义 detail.reason 编码规则 |

## 核心服务

| 文档 | 说明 |
|------|------|
| [IEC 104 数采平台](iec104.md) | 数采平台架构、服务组件、数据流、配置管理 |
| [IEC 104 消息对接](iec104-message.md) | 消息格式、ASDU 类型映射、信息体结构、数据消费指南 |
| [IEC 104 控制命令](iec104-command.md) | 控制命令接口、响应机制、错误码、TypeId 对照表 |
| [Trigger 服务](trigger.md) | 异步任务调度、计划任务管理、API、状态流转 |
| [SocketIO 实时通信](socketio.md) | SocketIO 网关对接、事件体系、MQTT 桥接、鉴权 |
| [DJI 云平台](djicloud.md) | DJI Dock3 Cloud API 封装、RPC 接口、配置说明 |
| [ISP 巡检协议](isp.md) | 变电站远程智能巡视系统 ISP 协议服务端/代理、帧格式、任务和模型同步 |
| [LAL 流媒体回调](../app/lalhook/README.md) | LAL HTTP 回调事件、鉴权、配置和接口说明 |

## 开发者

| 文档 | 说明 |
|------|------|
| [开发指南](development.md) | 环境搭建、代码生成、模块扩展、调试技巧 |
| [部署指南](deployment.md) | Docker 部署、集群部署、配置管理 |
| [KML/KMZ 指南](kml-kmz-guide.md) | 无人机航点任务 KML 文件结构和使用 |
| [antsx 与响应式模式](antsx-vs-reactive.md) | antsx Promise/Invoke 与响应式编排模式对比 |
