# zero-service

🚀 **基于 [go-zero](https://github.com/zeromicro/go-zero) 的高性能微服务脚手架**  
`zero-service` 旨在帮助开发者快速搭建可扩展、可维护的工业协议处理系统，已适配 **IEC 60870-5-104** 协议在电力场站的接入需求。

> 支持 Kafka、gRPC、异步调度、文件上传等场景，适配多语言微服务对接。

---

## 📚 目录

- [项目简介](#zero-service)
- [IEC 104 协议对接结构图](#iec-104-协议对接结构图)
- [核心服务模块](#核心服务模块)
    - [`iec104`：IEC 协议处理模块](#iec104-iec-协议处理模块)
    - [`iec-stash`：Kafka 数据消费与转发](#iec-stash-kafka-数据消费与转发)
    - [`file`：文件上传服务](#file-文件上传服务)
    - [`trigger`：异步任务调度服务](#trigger-异步任务调度服务)
    - [`lalhook`：LAL 流媒体服务钩子](#lalhook-流媒体服务钩子模块)
- [使用注意事项](#使用注意事项)
- [相关链接](#相关链接)

---

## 🛰 IEC 104 协议对接结构图

该图展示了 **本系统作为 IEC 60870-5-104 主站，主动对接多个下级从站设备的整体架构**：

<div align="center">
  <img src="doc/iec-architecture.png" alt="IEC104 主站对接结构图" style="max-width: 100%; height: auto;" />
</div>

---

## 🧩 核心服务模块

### `iec104` IEC 协议处理模块

- 💡 提供 IEC 60870-5-104 协议 ASDU 报文的接收、解析与编码能力；
- 🔍 支持多种类型 ASDU 的解析与生成（遥信、遥测、遥控等）；
- 🧪 提供协议模拟功能，便于调试与联调；
- 🔗 集成 Kafka，实现异步、高吞吐的数据流处理；
- 📄 对接文档：[Kafka 消息格式说明](common/iec104/kafka.md)

---

### `iec-stash` Kafka 数据消费与转发

- ✅ 消费来自 `iec104` 服务发送的 Kafka 消息；
- 🧩 支持 chunk 批处理机制，提升处理效率；
- 🚀 基于 [go-queue](https://github.com/zeromicro/go-queue)，支持高并发处理（理论峰值 15W/s）；
- 📡 将处理结果通过 gRPC 下发至后端业务模块；
- 📄 协议定义：[`iecstream.proto`](facade/iecstream/iecstream.proto)

---

### `file` 文件上传服务

- 💾 支持基于 gRPC 的分片流式上传；
- ☁️ 集成对象存储（OSS）能力，支持大文件断点续传；
- 📁 可用于存储协议数据、任务报告等业务文件。

---

### `trigger` 异步任务调度服务

- ⏱️ 基于 [asynq](https://github.com/hibiken/asynq)，实现定时/延时任务调度；
- 📦 使用 Redis 存储任务队列，支持多节点部署与高可用；
- 🔁 支持 HTTP/gRPC 回调，适配多种业务场景；
- 🔧 支持任务归档、删除与自动重试等管理能力；
- 📄 协议定义：[`trigger.proto`](app/trigger/trigger.proto)

<div align="center">
  <img src="doc/trigger-flow.png" alt="Trigger 服务流程图" style="max-width: 80%; height: auto;" />
</div>

---

## `lalhook` LAL 直播服务钩子模块

- 🔧 集成 LAL 回调接口
- 📦 集成 ts 录制记录回调，提供分片播放能力

---

## ⚙️ 使用注意事项

1. **依赖管理**：请确认 `go.mod` 中的依赖已正确安装，执行 `go mod tidy`；
2. **日志配置**：确认各服务配置文件中的日志路径可用；
3. **Kafka 配置**：确保 Kafka 地址与 topic 配置正确；
4. **Java 接入**：如需与 Java
   应用集成，可参考 [grpc-spring-boot-starter](https://yidongnan.github.io/grpc-spring-boot-starter/zh-CN/)。

---

## 🔗 相关链接

- [go-zero 微服务框架](https://github.com/zeromicro/go-zero)
- [go-queue 高性能队列](https://github.com/zeromicro/go-queue)
- [asynq 异步任务队列](https://github.com/hibiken/asynq/)
- [lancet 工具包](https://github.com/duke-git/lancet)
- [squirrel SQL 构建器](https://github.com/Masterminds/squirrel)
- [IEC104 协议库](https://github.com/wendy512/iec104)

---

