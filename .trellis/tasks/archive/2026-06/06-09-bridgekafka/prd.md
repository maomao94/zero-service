# bridgekafka 模块 PRD

## 背景

当前项目已有 `bridgemqtt` 模块，负责 MQTT 消息的桥接：接收外部 MQTT 消息，转发到内部 gRPC（streamevent）和 SocketIO。现在需要一个对称的 `bridgekafka` 模块，用于 Kafka 消息桥接。

## 需求概述

参照 `bridgemqtt` 的架构模式，新建 `bridgekafka` 模块，使用 `go-queue`（kq）作为 Kafka 客户端，实现以下能力：

1. **多集群连接**：支持配置多个 Kafka 集群地址（brokers）
2. **消息推送**：提供 gRPC 接口，将消息推送到指定集群的指定 topic
3. **消息消费转发**：消费 Kafka 消息，通过 gRPC 调用 `streamevent.ReceiveKafkaMessage` 转发到下游

## 功能需求

### FR1: Kafka 消息推送（Publish）

- 提供 gRPC 服务 `BridgeKafka`
- 接口 `Publish(PublishReq) returns (PublishRes)`：将消息推送到指定 topic
- 接口 `PublishWithTrace(PublishWithTraceReq) returns (PublishWithTraceRes)`：带 traceId 的推送，用于链路追踪
- 底层使用 `kq.NewPusher(brokers, topic)` 创建生产者

### FR2: Kafka 消息消费转发

- 启动时根据配置订阅指定 Kafka topic + group
- 消费到消息后，封装为 `streamevent.KafkaMessage`，调用 `streamevent.ReceiveKafkaMessage` gRPC 接口
- 消费接口实现 `Consume(ctx context.Context, key, value string) error`（go-queue 标准接口）
- 支持可选的 SocketIO 广播转发（与 bridgemqtt 一致）

### FR3: 配置管理

- 使用 `kq.KqConf` 作为 Kafka 消费配置（brokers, topic, group, consumers 等）
- 使用自定义结构体管理推送用的 brokers/topic 配置
- 支持 Nacos 服务注册
- 支持日志配置

## 非功能需求

- 遵循项目现有 go-zero 微服务分层：config / server / logic / handler / svc
- 使用 goctl 生成的 proto 代码骨架
- 消息处理使用 `threading.TaskRunner` 异步调度，避免阻塞消费

## 验收标准

- [ ] `bridgekafka.proto` 定义 `BridgeKafka` 服务，包含 `Publish` 和 `PublishWithTrace` 两个 RPC
- [ ] 启动后能连接 Kafka 集群并消费消息
- [ ] 消费到的消息能正确转发到 `streamevent.ReceiveKafkaMessage`
- [ ] 通过 gRPC 调用 `Publish` 能成功将消息推送到 Kafka topic
- [ ] 配置结构清晰，支持多集群地址
- [ ] 代码结构与 `bridgemqtt` 保持一致

## 参考模块

- `app/bridgemqtt` — 架构模板
- `common/mqttx` — 客户端封装模式（本模块直接使用 go-queue，不需要额外封装）
- `facade/streamevent/streamevent.proto` — 下游 gRPC 接口定义
- `app/iecstash` — go-queue 消费者模式参考
- `app/ieccaller` — go-queue 生产者模式参考
