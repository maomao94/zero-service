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
- 按 `modbusCode` 复用长期客户端，同 code 串行收发：空 code 使用文件默认配置，非空 code 查启用的表配置，不回退。保存/删除配置后立即失效对应客户端；直接改库时由后台定期校验后生效（通常 30 秒内）；数据库查询失败保留现有连接。写操作不会自动重试。
- 连接配置可持久化到 DB（`ModbusSlaveConfig`），由 Converter 统一转换；表中 `slave` 不在 1–247 时按配置不可用处理，返回 `100044`，不会静默改用默认从站号。
- 入口统一校验 `address`（0–65535）与各功能码的 `quantity` 上限，超范围返回 `100009`；设备异常响应转成 `100058`（`vendor=modbus`，原始异常码在 `msg` 中），超时返回 `100003`，其余通信故障返回 `100057`。
- 配置管理接口校验必填编码、TCP `host:port` 地址、`slave`（1–247）、批量数组和查询状态，非法参数返回 `100009`；完整约束见[服务参数说明](../../app/bridgemodbus/README.md#协议与参数约束)。
- 写操作通信失败可能发生在设备执行之后，结果未知时必须核实设备状态后再决定是否重试。

配置：`app/bridgemodbus/etc/bridgemodbus.yaml`（默认 `ModbusClientConf`、`DB`）；缺省 `Timeout=10000ms`、`IdleTimeout=60000ms`、`ConnectDelay=0`、恢复窗口为 0。保持连接可显式设 `IdleTimeout=-1`。
契约：[`app/bridgemodbus/bridgemodbus.proto`](../../app/bridgemodbus/bridgemodbus.proto)。
服务说明：[`app/bridgemodbus/README.md`](../../app/bridgemodbus/README.md)。

### 开放平台接入

`app/openapi` 通过 `BridgeModbusConf` 调用本服务，`opengtw` 对外开放 15 个 I/O method：

- 读取：`modbus.readCoils`、`modbus.readDiscreteInputs`、`modbus.readInputRegisters`、`modbus.readHoldingRegisters`、`modbus.readFIFOQueue`、`modbus.readDeviceIdentification`、`modbus.readDeviceIdentificationSpecificObject`
- 写入：`modbus.writeSingleCoil`、`modbus.writeMultipleCoils`、`modbus.writeSingleRegister`、`modbus.writeSingleRegisterWithDecimal`、`modbus.writeMultipleRegisters`、`modbus.writeMultipleRegistersWithDecimal`、`modbus.readWriteMultipleRegisters`、`modbus.maskWriteRegister`

配置管理（保存/删除/查询）与本地换算暂不对外开放。字段、示例与安全提示见 [`docs/open-api/modbus.md`](../open-api/modbus.md)。

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
- `common/modbusx`：长期 Modbus 客户端与按 code 缓存的 `ClientManager`。
- `common/stream`：桥接服务共用的流事件转发辅助。

## 部署

- 各服务均为独立 go-zero 服务，启动方式：

```bash
./<service> -f etc/<service>.yaml
```
- bridgekafka 依赖 Kafka 集群；bridgemqtt 依赖 MQTT Broker 与 socketpush/streamevent 服务；bridgemodbus 依赖可访问的 Modbus TCP 设备与数据库。
- 新增 bridgegtw 映射时需同步更新 proto descriptor 并重新构建 gateway 镜像。
