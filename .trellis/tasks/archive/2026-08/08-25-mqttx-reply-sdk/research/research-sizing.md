# 调研报告：oryxserver / ieccaller mqtt reply 逻辑抽取 SDK（mqttx 基础能力）规模评估

- **Query**: 抽取 oryxserver 与 ieccaller 两处 MQTT 广播 grpc 集群 ack 逻辑为通用 SDK（mqttx 基础能力）；参考 djsdk 的主题定义方式改进现有主题方案
- **Scope**: internal（代码库检索）+ 外部对照（djisdk 主题定义规范）
- **Date**: 2026-08-25

## 结论摘要（优先看这里）

1. **两处实现都在 `zero-service` 单仓库内**（module `zero-service`, go 1.26.0），不在 `/Users/hehanpeng/GolandProjects/oryx`（该目录是 oryxproject 流媒体项目，无相关关键字命中）。
   - oryxserver: `app/oryxserver/`（发送端 + 消费端）
   - ieccaller: `app/ieccaller/`（发送端 + 消费端）
2. **两处已共享底层 `common/mqttx`**（client / ReplyRouter / RequestReply / decoder），重复的不是 MQTT 基础设施，而是**协议层模板**：主题常量、请求/ack body 结构、ack 解码器、ReplyRouter 初始化、instanceId 约定、publishAck 回包逻辑。
3. **`djsdk` 实际是 `djisdk`**（`zero-service/common/djisdk/`，全仓库对 "djsdk" 精确匹配 0 命中）；其主题定义风格为**函数式 + 成对 Pattern + 方向/用途注释 + 分组 + 一致性测试**，与现有裸字符串常量（`BroadcastTopic = "oryx/relay/broadcast"`、硬编码 `"iec/broadcast-ack/%s"`）差异明显。
4. `mqttx` 已存在且成熟（1393 行，被 4 个 app + djisdk 复用），**建议方向**：在 mqttx 之上新增 protocol-neutral 的广播/ack 泛化层（主题定义 + 标准 body + 消费分发骨架 + errorKind 归一），两个 app 收敛到该层，保留各自的业务 method 分发。

---

## 1. oryxserver 的 mqtt reply / grpc 广播 ack 逻辑

### 1.1 位置与模块归属

| 项 | 值 |
|---|---|
| 代码目录 | `zero-service/app/oryxserver/` |
| go.mod module | `zero-service`（单 go.mod monorepo，无 go.work） |
| 所属 app | oryxserver（zrpc 服务，Nacos 注册，`app/oryxserver/oryxserver.go` 启动） |
| 业务语义 | 转推（relay）集群模式：一个节点停止 FFmpeg 转推任务时，通过 MQTT 广播通知其他节点 |

### 1.2 主题常量定义（现状）

`app/oryxserver/internal/relay/broadcast.go:12-20`：

```go
// MQTT 广播常量（cluster 模式跨节点停止转推，对齐 ieccaller 模式）
const (
	// MethodStreamRelayStop 广播方法名（对齐 gRPC full method）
	MethodStreamRelayStop = "/oryxserver.OryxServer/StreamRelayStop"
	// BroadcastTopic 跨节点停止命令广播主题
	BroadcastTopic = "oryx/relay/broadcast"
	// BroadcastAckTopicPrefix 每实例 ack 回复主题前缀（后面拼接 instanceId）
	BroadcastAckTopicPrefix = "oryx/relay/broadcast-ack/"
)
```

### 1.3 消息体结构

`app/oryxserver/internal/relay/broadcast.go:23-37`：

```go
// RelayBody 转推广播请求体
type RelayBody struct {
	Tid      string `json:"tId,omitempty"`
	AckTopic string `json:"ackTopic"`
	Method   string `json:"method"`
	TaskId   string `json:"taskId"`
}

// RelayAckBody 转推广播 ack 响应体
type RelayAckBody struct {
	Tid       string `json:"tId"`
	Method    string `json:"method"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	ErrorKind string `json:"errorKind,omitempty"`
}
```

### 1.4 ack 解码器（mqttx.ReplyDecoder 适配）

`app/oryxserver/internal/relay/broadcast.go:40-59`：`DecodeRelayAck(ctx, payload, topic, topicTemplate)` — jsonx.Unmarshal → 校验 Tid 非空（`mqttx.ErrEmptyReplyTid`）→ 返回 `mqttx.ReplyMessage[*RelayAckBody]`。

### 1.5 客户端初始化（servicecontext）

`app/oryxserver/internal/svc/servicecontext.go:42-63`：

```go
// MQTT 初始化条件：cluster 模式必需，standalone 不依赖
if svcCtx.IsBroadcast() && len(c.MqttConfig.Broker) == 0 {
	logx.Must(fmt.Errorf("relay broadcast is enabled (deployMode=cluster), but mqtt config is empty"))
}
uid, err := tool.SimpleUUID()
...
svcCtx.relayInstanceId = "oryx-relay-" + uid
if svcCtx.IsBroadcast() {
	svcCtx.relayTopic = relay.BroadcastTopic
	svcCtx.relayAckTopic = fmt.Sprintf("%s%s", relay.BroadcastAckTopicPrefix, svcCtx.relayInstanceId)
	relayReplyRouter := mqttx.NewReplyRouter[*relay.RelayAckBody](
		mqttx.ReplyDecoderFunc[*relay.RelayAckBody](relay.DecodeRelayAck),
		mqttx.WithReplyRouterName("mqtt-ack-reply-"+uid),
		mqttx.WithReplyRouterTTL(10*time.Second),
	)
	cfg := c.MqttConfig.MqttConfig
	cfg.ClientID = svcCtx.relayInstanceId
	cfg.Qos = 1
	svcCtx.MqttClient = mqttx.MustNewClient(cfg, mqttx.WithReplyRouter(svcCtx.relayAckTopic, relayReplyRouter))
}
```

- `IsBroadcast()` = `DeployMode == "cluster"`（`servicecontext.go:82-84`）
- instanceId 约定：`"oryx-relay-" + uid`（`servicecontext.go:50`）
- ReplyRouter TTL：10s；ClientID = instanceId；Qos = 1

### 1.6 发送端

`app/oryxserver/internal/svc/servicecontext.go:96-117` `PublishRelayStop(ctx, tid, taskID)`：构造 `RelayBody{Tid, AckTopic: relayAckTopic, Method: MethodStreamRelayStop, TaskId}` → json.Marshal → `PublishWithTrace(pushCtx(10s), relayTopic, byteData)`。
注意：**oryxserver 发送端是 fire-and-forget，没有调用 `mqttx.RequestReply` 等待本节点 ack**（ReplyRouter 只挂着接收，无 pending 等待者）；gRPC 入口 `app/oryxserver/internal/logic/streamrelaystoplogic.go:60` 调用 `PublishRelayStop(l.ctx, tid, in.TaskId)`。

### 1.7 消费端 + ack 回发

`app/oryxserver/mqtt/broadcast.go:26-69` `Broadcast.Consume`（实现 `mqttx.ConsumeHandler` 签名）：

- 非 cluster 直接返回；(line 36-38) 检测 `broadcastBody.AckTopic == RelayAckTopic()` 时忽略自身消息（防回环）；
- `switch Method`（line 41-67）：仅 `MethodStreamRelayStop` → `RelayManager.Stop(TaskId)`；`found=true` 时 `publishAck(ctx, tid, ackTopic, method, true, "")`（line 61）；非本节点任务不回 ack（line 52-59）；
- `publishAck`（line 71-98）：构造 `RelayAckBody{Tid, Method, Success, Error}` → `PublishWithTrace(10s, ackTopic, data)`。**未做 errorKind 分类**。

### 1.8 消费注册

`app/oryxserver/oryxserver.go:50-58`：

```go
if ctx.MqttClient != nil && ctx.IsBroadcast() {
	if err := ctx.MqttClient.AddHandlerFunc(
		ctx.RelayBroadcastTopic(),
		mqtt.NewBroadcast(ctx).Consume,
	); err != nil { logx.Must(err) }
}
```

---

## 2. ieccaller 的同类逻辑

### 2.1 位置与模块归属

| 项 | 值 |
|---|---|
| 代码目录 | `zero-service/app/ieccaller/` |
| go.mod module | `zero-service`（同一 monorepo） |
| 业务语义 | IEC104 命令集群广播：任一节点下发遥控/遥调命令时，经 MQTT 广播到集群其他节点执行；命令可等待执行 ack 回传 |

### 2.2 主题定义（现状：内联字符串，无常量）

`app/ieccaller/internal/svc/servicecontext.go:70-71`：

```go
svcCtx.broadcastTopic = "iec/broadcast"
svcCtx.broadcastAckTopic = fmt.Sprintf("iec/broadcast-ack/%s", svcCtx.broadcastInstanceId)
```

- instanceId 约定：`"iec-caller-" + uid`（`servicecontext.go:65`）

### 2.3 消息体结构（在 common/iec104/types）

`common/iec104/types/types.go:11-26`：

```go
type BroadcastBody struct {
	Tid      string `json:"tId,omitempty"`
	AckTopic string `json:"ackTopic"`
	Method   string `json:"method"`
	Body     string `json:"body"`
}

// BroadcastAckBody 广播ACK响应体，用于集群模式下回传指令执行结果
type BroadcastAckBody struct {
	Tid          string `json:"tId"`
	Method       string `json:"method"`
	Success      bool   `json:"success"`
	ResponseBody string `json:"responseBody"`
	Error        string `json:"error,omitempty"`
	ErrorKind    string `json:"errorKind,omitempty"`
}
```

### 2.4 客户端初始化

`app/ieccaller/internal/svc/servicecontext.go:58-85`：与 oryxserver 完全同构的模板（IsBroadcast 校验、`"iec-caller-"+uid`、`mqttx.NewReplyRouter[*types.BroadcastAckBody]` + `decodeBroadcastAck` + `WithReplyRouterTTL(10s)`、`cfg.ClientID = instanceId`、`cfg.Qos=1`、`mqttx.MustNewClient(cfg, WithReplyRouter(...))`）。差异点：**非 cluster 但配置了 broker 时也会建客户端**（line 81-84，用于数据推送，与广播无关）。

### 2.5 发送端（完整 request→ack 回环）

`app/ieccaller/internal/svc/servicecontext.go:276-340`：

- `PushPbBroadcast(ctx, method, in)`（line 276-281）：fire-and-forget，仅发布；被 `clearpointmappingcachelogic.go:55` 使用；
- `PushPbBroadcastWithAck(ctx, method, in, res)`（line 283-310）：**核心 ack 等待路径**——生成 tId → `mqttx.RequestReply[*types.BroadcastAckBody](ctx, svc.MqttClient, svc.broadcastAckTopic, tId, func() error { return svc.pushBroadcast(...) })`（line 295-297）→ `ack.Success == false` 时 `broadcastAckError(ack)`（line 302-304）→ `protojson.Unmarshal(ack.ResponseBody, res)`（line 306-308）；
- `pushBroadcast`（line 312-340）：`protojson.Marshal(in)` → 构造 `BroadcastBody{AckTopic, Method, Body}` → `PublishWithTrace(10s, broadcastTopic, byteData)`；
- `broadcastAckError`（line 342-357）：按 `ack.ErrorKind` 归一为领域错误——`timeout`→`antsx.ErrReplyExpired`；`duplicate`→`antsx.ErrDuplicateID`；`iec_rejected`→`client.CommandRejectedError`；default→fmt.Errorf。

调用方（`app/ieccaller/internal/logic/`）：
- `PushPbBroadcastWithAck`：sendcommand、sendsinglecommand、senddoublecommand、sendstepcommand、sendtestcmd、sendreadcmd、sendinterrogationcmd、sendcounterinterrogationcmd、sendsetpointnormalized、sendsetpointfloat、sendsetpointscaled、sendbitstringcommand（12 个，如 sendcommandlogic.go:37）
- `PushPbBroadcast`（无 ack）：clearpointmappingcachelogic.go:55

### 2.6 消费端（14 个 method 大 switch）

`app/ieccaller/mqtt/broadcast.go:32-341` `Broadcast.Consume`：

- 非 cluster 返回；line 42-44 自身 ack 回环忽略；
- `switch broadcastBody.Method`（line 46-339）：14 个 case，每个都是「protojson 反序列化请求 → ClientManager.GetClient(host,port) → 执行 iec104 client 命令 → 成功/失败 publishAckReply」；失败 `publishAckReply(..., false, "", err)`；成功 `publishAckReply(..., true, "{}" 或 protojson.Marshal(res), nil)`；
- case 列表：SendCounterInterrogationCmd、SendInterrogationCmd、SendReadCmd、SendTestCmd、SendCommand、SendSingleCommand、SendDoubleCommand、SendStepCommand、SendSetpointNormalized、SendSetpointScaled、SendSetpointFloat、SendBitstringCommand、ClearPointMappingCache（含无 ack 回包的 cache 清除）；
- `publishAckReply`（line 349-396）：构造 `types.BroadcastAckBody{...}`；**errorKind 归一**（line 353-367）：`timeout`（antsx.ErrReplyExpired）、`duplicate`（antsx.ErrDuplicateID）、`iec_rejected`（client.CommandRejectedError）、默认 `unknown`；`PublishWithTrace(10s, ackTopic, data)`。

### 2.7 消费注册

`app/ieccaller/ieccaller.go:95-102`：`AddHandlerFunc(ctx.BroadcastTopic(), iecmqtt.NewBroadcast(ctx).Consume)`。

### 2.8 ack 解码器

`app/ieccaller/internal/svc/servicecontext.go:390-410` `decodeBroadcastAck`：jsonx.Unmarshal → Tid 空则 `mqttx.ErrEmptyReplyTid` → `mqttx.ReplyMessage[*types.BroadcastAckBody]{Tid, Value}`。与 oryxserver 的 `DecodeRelayAck` 逐行同构（仅日志字段/topic 模板不同）。

---

## 3. "djsdk" 主题定义方式（实际为 djisdk）

### 3.1 名称确认

- 全仓库搜索 **"djsdk"**（精确）：**0 命中**。
- 实际模块为 **`djisdk`**：`zero-service/common/djisdk/`（package djisdk，23 个文件），被 `app/djicloud/internal/logic/drchelper.go:7` 引用（`import "zero-service/common/djisdk"`），用于 DJI 上云协议（services/property/DRC 等）。
- `mock-service` 仓库（module `mock-service`, go 1.25.0，独立仓库）也有旧拷贝 `mock-service/common/djisdk/`（diff 显示与 zero-service 版本有差异，为历史副本）。

### 3.2 djisdk 主题定义风格（topic.go，共 186 行）

**核心模式：每个 topic 一个构造函数 + 成对的通配订阅 Pattern 函数 + 三要素注释（路径格式/方向/用途）+ 分组注释 + 顶部文档指引。**

示例（`common/djisdk/topic.go:49-63`）：

```go
// ServicesTopic 返回云平台下发服务调用的 Topic。
// 路径格式: thing/product/{gateway_sn}/services
// 方向: 云平台 → 设备
// 用途: 云平台向网关设备下发服务指令（如航线任务下发、设备控制、固件升级等）。
func ServicesTopic(gatewaySn string) string {
	return fmt.Sprintf("thing/product/%s/services", gatewaySn)
}

// servicesReplyTopicPattern 返回 ServicesReply Topic 的通配订阅模式。
// 路径格式: thing/product/+/services_reply
// 方向: 设备 → 云平台（云平台侧订阅）
// 用途: 云平台使用该模式订阅所有网关设备的服务调用响应。
func servicesReplyTopicPattern() string {
	return "thing/product/+/services_reply"
}
```

分组结构（`topic.go:10, 89, 109, 130, 160`）：

```go
// ==================== Thing Topic ====================
// Thing Topic 用于设备物模型相关的消息通信，包括遥测数据上报、
// 云端服务下发、设备事件上报等。
// 基础路径格式: thing/product/{gateway_sn}/{channel}
```

分组：Thing / Organization·Requests / Property（Dock3）/ Sys / DRC。每组顶部有指向官方文档链接（DJI Topic 总览等），包级 `doc.go` 提供全局行为约定（方向、result 码、DRC 通道与 services 的区别）。

Topic 函数清单（`topic.go`）：OsdTopic(Pattern)、StateTopic(Pattern)、ServicesTopic、servicesReplyTopicPattern、EventsTopic(Pattern)、EventsReplyTopic、RequestsTopicPattern、RequestsReplyTopic、PropertySetTopic、propertySetReplyTopicPattern、StatusTopic(Pattern)、StatusReplyTopic、DrcUpTopic(Pattern)、DrcDownTopic。

一致性测试（`topic_test.go:5-44`）：`TestTopicPatternsMatchConcreteTopics` 断言每个 Pattern 通配符 `+` 与具体 topic 的 `{gateway_sn}` 对齐，防止主题漂移。

### 3.3 与 BroadcastTopic 定义方式的差异

| 维度 | 现有（oryxserver/ieccaller） | djisdk 风格 |
|---|---|---|
| 定义载体 | 裸字符串常量（`BroadcastTopic = "oryx/relay/broadcast"`）；`BroadcastAckTopicPrefix + instanceId` 拼接 | 构造函数 `XxxTopic(sn)` + 成对 Pattern 函数 `XxxTopicPattern()` |
| 通配订阅 | 无（消费注册传具体 topic） | 成对提供 `+/` 通配模式（`thing/product/+/osd`） |
| 方向/用途注释 | 一行中文注释 | 三要素：路径格式 / 方向（云↔设备）/ 用途 |
| 分组 | 无（各 app 各自散落） | `====` 注释分组 + 组内文档链接 |
| 测试 | 无 | topic_test.go 校验 Pattern 与具体主题一致 |
| 参数化 | 前缀字符串 + uid 拼接 | 函数参数（gatewaySn），便于单元断言 |

---

## 4. mqttx 现状（已存在的底座）

### 4.1 位置与能力

`zero-service/common/mqttx/`（package mqttx，9 个 go 文件，1393 行）：

| 文件 | 能力 |
|---|---|
| `client.go` (420 行) | `Client` 接口 + `mqttClient` 实现：`AddHandler`/`AddHandlerFunc`/`Publish`/`PublishWithTrace`/`Close`/`GetClientID`；`MustNewClient`/`NewClient`；`WithOnReady`/`WithReplyRouter`；连接恢复订阅（restoreSubscriptions）、trace、metrics |
| `config.go` (58 行) | `MqttConfig`（Broker/ClientID/Username/Password/Qos/Timeout/KeepAlive/SubscribeTopics） |
| `reply_router.go` (139 行) | `ReplyRouter[T]`（基于 `antsx.ReplyPool`）、`ReplyDecoder[T]`、`ReplyDecoderFunc[T]`、`ReplyMessage[T]`、`NewReplyRouter`、`WithReplyRouterTTL/Name`、`Consume` |
| `request_replyer.go` (29 行) | `RequestReply[T](ctx, c Client, topicTemplate, tid, send, ttl...)` 公开泛型入口 |
| `dispatcher.go` (164 行) | 消息分发（reply router 优先 + 普通 handler） |
| `errors.go` | `ErrEmptyReplyTid`/`ErrNoReplyRouter`/`ErrReplyType`/`ErrReplyNotMatched`/`ErrNilDecoder` |
| `message.go` | 载荷解包（`Message` 含 trace headers） |
| `topic_log.go` | `TopicLogConfig` 日志限频配置 |

底层：`github.com/eclipse/paho.mqtt.golang` + `zero-service/common/antsx`（ReplyPool/RequestReply 通用请求-应答池）。

关键接口（`common/mqttx/client.go:37-47`）：

```go
type Client interface {
	AddHandler(topicTemplate string, handler ConsumeHandler) error
	AddHandlerFunc(topicTemplate string, fn func(context.Context, []byte, string, string) error) error
	Publish(ctx context.Context, topic string, payload []byte) error
	PublishWithTrace(ctx context.Context, topic string, payload []byte) (string, error)
	Close()
	GetClientID() string
}
```

### 4.2 引用方（谁在用 mqttx）

`rg -l "zero-service/common/mqttx"` 命中（zero-service 内）：

- `app/oryxserver/`（internal/relay/broadcast.go、internal/svc/servicecontext.go、internal/logic/streamrelaystoplogic.go、internal/config/config.go）
- `app/ieccaller/`（internal/svc/servicecontext.go、servicecontext_test.go、internal/config/config.go）
- `app/djicloud/`（internal/logic/drchelper.go、drchelper_test.go）
- `app/bridgemqtt/`（internal/svc/servicecontext.go、internal/handler/mqttstreamhandler.go、internal/config/config.go）
- `common/djisdk/`（client.go、handler.go、client_test.go、protocol_drc_test.go）

即：**mqttx 已是事实上的 MQTT 基础 SDK**，被 4 个 app + djisdk 依赖。两个 broadcast 实现已经是它的"上层协议"用例。

### 4.3 外部拷贝

`mock-service`（独立仓库）下有 `common/mqttx/`、`common/djisdk/`、`common/gnetx/` 的**旧拷贝**（`client.go` 与 zero-service 版本 diff 不一致 → 非同步副本；后续抽取需注意两块并存）。

---

## 5. 架构关系（两 app + monorepo）

- **单仓库单模块**：`zero-service/go.mod`（module `zero-service`，`go 1.26.0`）；**无 go.work**；app 间无 replace，共用同一 module。
- **replace** 仅 1 条：`github.com/doquangtan/socketio/v4 => github.com/maomao94/socket.io-golang/v4`（go.mod:351），无本地路径 replace。
- **目录结构**：
  - `app/`（23 个 app：oryxserver、ieccaller、djicloud、bridgemqtt、socketapp/*、iecagent、lalhook…）—— 每个 app 是 go-zero 风格 zrpc 服务；
  - `common/`（共享 SDK 层：mqttx、djisdk、iec104（含 client/types/util）、antsx、tool、gormx、grpcx、executorx、oryxx（oryx HTTP client）…）—— **没有 `sdk/`、`microsdk/` 目录，`common/` 即 SDK 层**；
  - `facade/`（proto 生成包：streamevent 等）；`model/`（gorm model）。
- **oryxserver 与 oryx 仓库的关系**：`/Users/hehanpeng/GolandProjects/oryx` 是 oryxproject 流媒体服务器（Rust 为主，go.mod 散于 platform/releases/test 子目录），与 zero-service 的 oryxserver（go-zero zrpc 服务，通过 `common/oryxx` HTTP 客户端调用 oryx HTTP API）**相互独立**；`BroadcastAckTopicPrefix` 等关键字在 /oryx 无命中。
- **两 app 共同依赖**：均为 `zero-service` module 内部包（common/mqttx、common/tool、common/antsx…），无外部私有依赖；ieccaller 额外依赖 common/iec104（client/types/util）、facade/streamevent 等。

---

## 6. 两处实现异同对比表

| 维度 | oryxserver | ieccaller |
|---|---|---|
| 主题常量 | `relay/broadcast.go:17-19` 常量 + 前缀拼接 | `servicecontext.go:70-71` 内联字符串拼接 |
| 主题值 | `oryx/relay/broadcast`、`oryx/relay/broadcast-ack/{instanceId}` | `iec/broadcast`、`iec/broadcast-ack/{instanceId}` |
| instanceId | `"oryx-relay-" + uid` (svc:50) | `"iec-caller-" + uid` (svc:65) |
| 请求体 | `RelayBody{Tid,AckTopic,Method,TaskId}` | `BroadcastBody{Tid,AckTopic,Method,Body}`（Body 为 protojson 字符串） |
| ack 体 | `RelayAckBody{Tid,Method,Success,Error,ErrorKind}` | `BroadcastAckBody{Tid,Method,Success,ResponseBody,Error,ErrorKind}` |
| ack 解码器 | `DecodeRelayAck` (relay/broadcast.go:40-59) | `decodeBroadcastAck` (svc/servicecontext.go:390-410) |
| 初始化模板 | IsBroadcast + ReplyRouter(10s) + ClientID + Qos1 + MustNewClient (svc:51-63) | 同构 (svc:69-80)；另有非 cluster 也建 MQTT 客户端用于数据推送 (svc:81-84) |
| 发送方式 | `PublishRelayStop`：**fire-and-forget**，不调 RequestReply (svc:97-117) | `PushPbBroadcastWithAck`：**等待 ack**（mqttx.RequestReply）；`PushPbBroadcast` 无 ack (svc:276-340) |
| 消费分发 | 单 method switch（mqtt/broadcast.go:41-67） | 14 method 大 switch（mqtt/broadcast.go:46-339） |
| 业务逻辑 | `RelayManager.Stop`；仅持有任务的节点回 ack（found 判断） | `ClientManager.GetClient` + iec104 命令执行 / 缓存清除 |
| errorKind | 无分类（仅 Error 字段） | 归一：timeout/duplicate/iec_rejected/unknown（publishAckReply:353-367） |
| ack 等待语义 | 无（不等待本节点 ack，接收侧隔离自身回环） | 有（等执行节点回 ack，`ack.Success` 判定 + `broadcastAckError` 映射） |
| 结构体放置 | app 内部 relay 包 | common/iec104/types（跨 app 可复用位置） |

**重复度评估**：初始化块（~15 行）+ ack 解码器（~20 行）+ 发布函数（~30 行）+ ack 回包（~45 行）≈ **110 行结构性重复**；两处 body 结构 95% 同构（仅 `TaskId` ↔ `Body`/`ResponseBody` 差异）；主题定义逻辑 100% 同构（仅字符串不同）。**不可重复部分**是业务 method 分发（oryx 1 个 case vs iec104 14 个 case）与 errorKind 映射规则。

## 7. 建议吸收方案草稿（不展开实现细节）

1. **底座不动**：`common/mqttx` 保持 protocol-neutral（client/reply-router/request-reply 已经抽象正确），两 app 继续依赖它。
2. **新增协议层**：在 mqttx 同层（或 mqttx 内子包）新增 broadcast 泛化能力：
   - **主题定义仿 djisdk**：`BroadcastTopic()` / `BroadcastTopicPattern()` / `BroadcastAckTopic(instanceId)` / `BroadcastAckTopicPattern()` 函数 + 三要素注释（路径格式/方向/用途）+ `====` 分组 + doc.go + topic 一致性单测；从 oryx/relay 前缀与 iec/ 前缀中抽象为可配置 prefix（如 `Prefix("oryx/relay")` 或 Options）。
   - **标准 body**：泛化 `BroadcastBody` / `BroadcastAckBody`（字段取并集：Tid/AckTopic/Method/Body/TaskId/ResponseBody/Success/Error/ErrorKind），oryxserver 的 `RelayBody`/`RelayAckBody` 与 ieccaller 的 `types.BroadcastBody/BroadcastAckBody` 标注为新类型的别名或迁移。
   - **标准 ack 解码器 + RequestReply 封装**：消除 `DecodeRelayAck`/`decodeBroadcastAck` 的逐行重复，提供 `NewBroadcastReplyRouter(ttl, name)` 或 `BroadcastClient` 包装（instanceId 生成、ClientID/Qos 设置、防回环判断）。
   - **消费分发骨架 + errorKind 归一**：把「方法分发 → ack 回发 → ErrorKind 分类」抽为通用 `Handler` 注册表（ieccaller 的 publishAckReply 为模板），业务只需注册 method→executor；oryxserver 也套用同一骨架并获得 errorKind。
3. **主题定义先落地**：优先以 djisdk 风格重建 `BroadcastTopic`/`BroadcastAckTopicPrefix`（含测试），因为这是用户指出的痛点且影响面小；body 与分发骨架随后。
4. **保留差异**：各 app 的 method 分发逻辑（iec104 命令、relay stop）留在各自 app；ack 是否等待（fire-and-forget vs RequestReply）作为广播客户端的可选模式。
5. **兼容注意**：`mock-service` 仓库存在 mqttx/djisdk 旧拷贝，若 SDK 升级需同步或明确该仓库冻结；`types.BroadcastBody` 位于 common/iec104/types，迁移应保持 ieccaller 依赖链不动（或重导出）。

## 8. Caveats / Not Found

- "djsdk" 精确匹配 0 命中（全 GoLandProjects 范围），确认目标应为 **djisdk**；如需第三处 "djsdk" 源码（如另一台机器/内部库），当前环境检索不到。
- `app/oryxserver/mqtt/` 与 `app/ieccaller/mqtt/` 各自只有一个 `broadcast.go`（消费端），无其他 MQTT 封装文件；mqttx client 是双方唯一 MQTT 客户端实现。
- oryxserver 的发送端未等待 ack（fire-and-forget），与 ieccaller 的 RequestReply 语义不对称——抽取时需决策是否统一。
- `common/iec104/types.BroadcastAckBody.ResponseBody`（protojson 字符串）与 oryxserver 的 `RelayAckBody` 无 ResponseBody 字段——并集方案需处理 protobuf 消息序列化的泛型约束（通用层建议用 `json.RawMessage` 或 string，由业务层转换）。
- 未验证 build/测试状态（纯调研，未运行 `go build`）。

