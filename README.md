# zero-service

🚀 **基于 [go-zero](https://github.com/zeromicro/go-zero) 的高性能微服务脚手架**  
`zero-service` 旨在帮助开发者快速搭建可扩展、可维护的工业协议处理系统，已适配 **IEC 60870-5-104** 协议在电力场站的接入需求, 已适配 **Modbus** 协议在工业自动化系统的接入需求。

> 支持 Kafka、gRPC、异步调度、文件上传等场景，适配多语言微服务对接。

---

## 📚 目录

- [项目简介](#zero-service)
- [IEC 104 协议对接结构图](#iec-104-协议对接结构图)
- [核心服务模块](#核心服务模块)
    - [`iec104`：IEC 协议处理模块](#iec104-协议处理模块)
    - [`iec-stash`：Kafka 数据消费与转发](#iec-stash-数据消费与转发)
    - [`file`：文件上传服务](#file-文件上传服务)
    - [`trigger`：异步任务调度服务](#trigger-异步任务调度服务)
    - [`lalhook`：流媒体钩子服务](#lalhook-流媒体钩子服务)
    - [`bridgegtw`：HTTP 代理转发网关](#bridgegtw-http-代理转发网关)
    - [`bridgedump`：南瑞反向隔离装置文件生成服务](#bridgedump-南瑞反向隔离装置文件生成服务)
    - [`bridgemodbus`：modbus协议处理服务](#bridgemodbus-modbus协议处理服务)
    - [`bridgemqtt`：mqtt协议处理服务](#bridgemqtt-mqtt协议处理服务)
    - [`gis`：gis处理服务](#gis-gis处理服务)
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

#### IEC 60870-5-104 协议简介
IEC 60870-5-104 是一种用于电力系统远程控制和监测的国际标准协议，基于 TCP/IP 网络传输，采用问答式和循环式混合通信方式，适用于电力调度自动化系统中主站与子站之间的数据交换。

#### 核心功能
- 💡 提供完整的 IEC 60870-5-104 主站功能，支持与多个子站通信
- 🔍 支持多种类型 ASDU 的解析与生成（遥信、遥测等）
- 🔗 双协议消息推送：同时支持 Kafka 和 MQTT 消息推送，可灵活配置
- 🔀 动态 Topic 生成：基于配置规则和消息元数据动态生成推送 Topic
- 🧪 提供协议模拟功能，便于调试与联调
- 📄 支持完整的 ASDU 类型映射表，涵盖电力系统常见数据类型

#### 应用场景
- 智能电网监控系统
- 工业自动化远程控制
- 新能源电站数据采集
- 电力调度自动化
- 城市配电网监测

#### 技术特点
- 高性能设计，支持高并发消息处理
- 配置驱动，支持灵活部署和多环境适配
- 模块化架构，易于扩展和维护
- 完善的错误处理和异常恢复机制
- 详细的日志记录和监控功能

📄 对接文档：[IEC-104 消息对接文档](common/iec104/IEC-104 消息对接文档.md)

---

### `iec-stash` Kafka 数据消费与转发

- ✅ 消费来自 `iec104` 服务发送的 Kafka 消息
- 🧩 支持 chunk 批处理机制，提升处理效率
- 🚀 基于 [go-queue](https://github.com/zeromicro/go-queue)，支持高并发处理（理论峰值 15W/s）
- 📡 将处理结果通过 gRPC 下发至后端业务模块
- 📄 转发协议定义：[`streamevent.proto`](facade/streamevent/streamevent.proto)

---

### `file` 文件上传服务

- 💾 支持基于 gRPC 的分片流式上传
- ☁️ 集成对象存储（OSS）能力，支持大文件grpc stream 上传
- 📁 可用于存储协议数据、任务报告等业务文件

---

### `trigger` 异步任务调度服务

- ⏱️ 基于 [asynq](https://github.com/hibiken/asynq)，实现定时/延时任务调度
- 📦 使用 Redis 存储任务队列，支持多节点部署与高可用
- 🔁 支持 HTTP/gRPC 回调，适配多种业务场景
- 🔧 支持任务归档、删除与自动重试等管理能力
- 📄 协议定义：[`trigger.proto`](app/trigger/trigger.proto)

<div align="center">
  <img src="doc/trigger-flow.png" alt="Trigger 服务流程图" style="max-width: 80%; height: auto;" />
</div>

---

### `lalhook` 流媒体钩子服务

- 🔧 集成 LAL 回调接口
- 📦 集成 ts 录制记录回调，提供分片播放能力

---

### `bridgegtw` HTTP 代理转发网关

- 🌉 提供高性能的 HTTP 请求代理转发功能
- 🔀 支持多后端服务负载均衡与请求路由
- 🔒 内置访问控制与安全防护机制
- 📊 提供请求监控与统计功能

---

### `bridgedump` 南瑞反向隔离装置文件生成服务

- 📄 生成符合南瑞反向隔离装置要求的文本文件，格式为 `<Bridge:=Free...>JSON数据</Bridge:=Free>`
- 📑 支持多种数据类型的文件生成：
  - 电缆工作列表数据（输出到 `/opt/bridgedump/cable_work_list/*_json.txt`）
  - 电缆故障数据（输出到 `/opt/bridgedump/cable_fault/*_json.txt`）
  - 电缆故障波形数据（输出到 `/opt/bridgedump/cable_fault_wave/*_json.txt`）
- 📤 与 filebeat 无缝集成，自动采集生成的 txt 文件
- 📥 通过 filebeat 将数据分类发送至不同的 Kafka topic：
  - 电缆工作列表数据：`cable_work_list`
  - 电缆故障数据：`cable_fault`
  - 电缆故障波形数据：`cable_fault_wave`

---

### `bridgemodbus` modbus协议处理服务
- 📦 提供 Modbus TCP/RTU 协议处理能力
- 🔗 集成 GRPC 服务 
- 📄 协议定义：[`bridgemodbus.proto`](app/bridgemodbus/bridgemodbus.proto)

---

### `bridgemqtt` mqtt协议处理服务
- 📦 提供 mqtt 协议处理能力
- 🔗 集成 GRPC 服务
- 📄 协议定义：[`bridgemqtt.proto`](app/bridgemqtt/bridgemqtt.proto)
- 📄 转发协议定义：[`streamevent.proto`](facade/streamevent/streamevent.proto)

---

### `gis` gis处理服务
- 📦 提供 GIS 处理能力, 常见地理算法,围栏计算等
- 🔗 集成 GRPC 服务
- 📄 协议定义：[`gis.proto`](app/gis/gis.proto)

---

## ⚙️ 使用注意事项

1. **依赖管理**：请确认 `go.mod` 中的依赖已正确安装，执行 `go mod tidy`
2. **日志配置**：确认各服务配置文件中的日志路径可用
3. **Kafka 配置**：确保 Kafka 地址与 topic 配置正确
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
- [Modbus 协议库](https://github.com/grid-x/modbus)

---

