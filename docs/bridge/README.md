# Bridge 协议桥接服务

Bridge 系列是一组将外部协议接入内部系统的桥接服务，包含 HTTP 网关、Kafka、Modbus、MQTT 和文件落地 5 个服务。各服务独立部署，通过 gRPC 与内部服务交互。

## 服务总览

| 服务 | 目录 | 端口 | 职责 | 协议方向 |
| --- | --- | --- | --- | --- |
| bridgegtw | `app/bridgegtw/` | 15001 (HTTP) | HTTP → gRPC 代理网关 | HTTP → gRPC |
| bridgekafka | `app/bridgekafka/` | 21013 (gRPC) | Kafka 发布与消费转发 | gRPC + Kafka 双工 |
| bridgemodbus | `app/bridgemodbus/` | 25004 (gRPC) | Modbus TCP 协议桥接 | gRPC → Modbus TCP |
| bridgemqtt | `app/bridgemqtt/` | 25005 (gRPC) | MQTT 发布/订阅桥接 | gRPC + MQTT 双工 |
| bridgedump | `app/bridgedump/` | 25003 (gRPC) | 报文文件落地 | gRPC → 文件 |

端口事实以各服务 etc 目录下的 YAML 配置文件与[服务端口清单](../service-ports.md)为准。

## bridgegtw — HTTP → gRPC 代理

- 唯一 REST/gateway 服务，使用 go-zero gateway，将外部 HTTP 请求按 YAML upstreams 映射转发到内部 gRPC RPC。
- 默认提供 `GET /bridge/v1/ping`；外部电缆数据接口通过 upstreams 映射到 bridgedump：`POST /api/external/cable/workList|fault|faultWave` → `bridgedump.BridgeDumpRpc/*`。
- 代理需要 proto descriptor 文件（`bridgedump.pb`），新增映射需同步服务端 RPC 实现与 gateway 镜像中的 descriptor。

配置：`app/bridgegtw/etc/bridgegtw.yaml`（`Upstreams` 段定义映射）。
契约：[`app/bridgegtw/bridgegtw.api`](../../app/bridgegtw/bridgegtw.api)（REST 路由）、[`app/bridgedump/bridgedump.proto`](../../app/bridgedump/bridgedump.proto)（被代理的 RPC）。

## bridgekafka — Kafka 桥接

- 单一 gRPC RPC `Publish(topic, key, value)`：按 topic 查找对应 pusher 发布消息，未配置的 topic 报错。
- 消费侧通过 Kafka 消费者组（`KafkaConsumeConfig`）接收消息，由 TaskRunner 异步转发到 stream event gRPC 服务。
- 发布与消费使用独立配置（`KafkaPushConfig` / `KafkaConsumeConfig`）。

配置：`app/bridgekafka/etc/bridgekafka.yaml`。
契约：[`app/bridgekafka/bridgekafka.proto`](../../app/bridgekafka/bridgekafka.proto)。

## bridgemodbus — Modbus 桥接

- 提供配置管理（保存/删除/分页/详情）与 Modbus 读写 RPC，覆盖标准功能码（Bit Access：0x01/0x02/0x05/0x0F；Register Access：0x03/0x04/0x06/0x10/0x16/0x17/0x18；0x2B 设备识别），支持十进制数值读写。
- 核心机制为动态连接池：请求携带 `modbusCode`，服务按 code 从 PoolManager 解析连接池（未命中时查 DB 配置创建），`modbusCode` 为空时使用默认连接配置；连接使用后必须归还池。
- 连接配置可持久化到 DB（`ModbusSlaveConfig`），由 Converter 统一转换。

配置：`app/bridgemodbus/etc/bridgemodbus.yaml`（`ModbusPool`、默认 `ModbusClientConf`、`DB`）。
契约：[`app/bridgemodbus/bridgemodbus.proto`](../../app/bridgemodbus/bridgemodbus.proto)。

## bridgemqtt — MQTT 桥接

- 提供 `Publish` 与 `PublishWithTrace`（返回 OTel trace ID）两个发布 RPC。
- 订阅侧在连接就绪（OnReady）后按 `SubscribeTopics` 注册 handler，收到的消息双路 fan-out：
  1. 转发到 stream event gRPC（AI/实时事件服务）；
  2. 按 `EventMapping` 映射事件名广播到 socket push gRPC（WebSocket 房间）。
- 两条路径分别在独立 TaskRunner（16 并发）中异步执行，`TopicLogManager` 提供 per-topic 日志控制。

配置：`app/bridgemqtt/etc/bridgemqtt.yaml`（`MqttConfig`、`EventMapping`、`SocketPushConf`、`StreamEventConf`）。
契约：[`app/bridgemqtt/bridgemqtt.proto`](../../app/bridgemqtt/bridgemqtt.proto)。

## bridgedump — 报文文件落地

- 提供电缆在线监测数据接入：设备运行数据（`CableWorkList`）、故障结果（`CableFault`）、故障波形（`CableFaultWave`），每个 RPC 成功返回 `Code: 200, Msg: "成功"`。
- 核心逻辑将请求报文落地为文件：`{DumpPath}/{subDir}/{YYYYMMDD_HHmmss}_{traceID}_json.txt`，报文头为固定格式 `<!System=OMG Version=1.05 Code=utf-8 Data=1.0!>` + `<Bridge:=Free Size=N>` + JSON body。
- 无 Nacos 注册、无拦截器，是最简 zRPC 服务。

配置：`app/bridgedump/etc/bridgedump.yaml`（`DumpPath`）。
契约：[`app/bridgedump/bridgedump.proto`](../../app/bridgedump/bridgedump.proto)。

## 公共库

- `common/mqttx`：MQTT 客户端封装（订阅分发、请求-应答路由、OTel 追踪）。
- `common/modbusx`：Modbus 客户端与连接池（`ModbusClientPool`、`PoolManager`）。
- `common/stream`：桥接服务共用的流事件转发辅助。

## 部署

- 各服务均为独立 go-zero 服务，启动方式：

```bash
./<service> -f etc/<service>.yaml
```
- bridgekafka 依赖 Kafka 集群；bridgemqtt 依赖 MQTT Broker 与 socketpush/streamevent 服务；bridgemodbus 依赖可访问的 Modbus TCP 设备与数据库。
- 新增 bridgegtw 映射时需同步更新 proto descriptor 并重新构建 gateway 镜像。
