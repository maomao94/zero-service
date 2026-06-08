# IEC 104 控制命令与集群广播规范

> ieccaller 控制方向 gRPC 接口和集群广播（MQTT）的 canonical source。覆盖 proto 定义、client 封装、logic 编排、MQTT 广播全链路。

## When to read

- 新增或修改 ieccaller 控制方向（C_*）gRPC 接口。
- 修改 `common/iec104/client/core.go` 的 Send*Cmd 方法或 doSend 分发逻辑。
- 修改 `mqtt/broadcast.go` 消费者 case。
- 修改集群广播初始化、topic 派生、PublishWithTrace 或 reply pool 逻辑。
- 对接方询问控制命令的 typeId 和字段含义。

## 全链路文件清单

修改顺序强制：proto → gen.sh → client → logic → server → mqtt consumer。

| 顺序 | 文件 | 修改内容 |
|------|------|---------|
| 1 | `app/ieccaller/ieccaller.proto` | 新增 RPC + Request message + enum |
| 2 | `app/ieccaller/gen.sh` 执行 | 生成 pb.go / grpc pb / zrpc |
| 3 | `common/iec104/client/core.go` | 新增 Send*Cmd 方法 + doSend case |
| 4 | `app/ieccaller/internal/logic/` | 新建 *logic.go |
| 5 | `app/ieccaller/internal/server/ieccallerserver.go` | 新增 handler 方法 |
| 6 | `app/ieccaller/mqtt/broadcast.go` | 新增 consumer case |

## Scenario: 新增 typed 控制命令 RPC

### 1. Scope / Trigger

- Trigger: 每次新增控制方向 gRPC 接口，类型为 C_SC/C_DC/C_RC/C_SE/C_BO。
- Scope: 覆盖 6 层文件修改，从 proto 到 MQTT consumer。

### 2. Signatures

**Proto RPC 签名**：
```protobuf
// 命令名 (C_XX_NA_1=<typeId> 不带时标 / C_XX_TA_1=<typeId+13> 带CP56Time2a时标，由withTime字段控制)
rpc SendXxxCommand(SendXxxCommandReq) returns (SendXxxCommandRes);
```

**Proto Request Message 签名**：
```protobuf
message SendXxxCommandReq {
  string host = 1;
  uint32 port = 2;
  uint32 coa = 3;
  uint32 ioa = 4;
  <typedValue> value = 5;  // 语义注释
  bool withTime = 6; // true=带CP56Time2a时标(C_XX_TA_1=<id>), false=不带(C_XX_NA_1=<id>)
}
```

**Proto Response Message 签名**——每个命令有独立的 typed Res：
```protobuf
message SendXxxCommandRes {
  <typedValue> value = 1; // 从站回显的命令值（语义注释）
}
```

**Client 方法签名**：
```go
func (c *Client) SendXxxCmd(ctx context.Context, coa uint16, ioa asdu.InfoObjAddr, value <GoType>, withTime bool, opts ...client.CommandOption) (*client.CommandAck, error)
```

### 3. Contracts

#### 3.1 typeId 映射（控制方向）

| typeId | ASDU | withTime | 推荐接口 |
|--------|------|----------|---------|
| 45 | C_SC_NA_1 | false | `SendSingleCommand` |
| 46 | C_DC_NA_1 | false | `SendDoubleCommand` |
| 47 | C_RC_NA_1 | false | `SendStepCommand` |
| 48 | C_SE_NA_1 | false | `SendSetpointNormalized` |
| 49 | C_SE_NB_1 | false | `SendSetpointScaled` |
| 50 | C_SE_NC_1 | false | `SendSetpointFloat` |
| 51 | C_BO_NA_1 | false | `SendBitstringCommand` |
| 58 | C_SC_TA_1 | true | `SendSingleCommand` |
| 59 | C_DC_TA_1 | true | `SendDoubleCommand` |
| 60 | C_RC_TA_1 | true | `SendStepCommand` |
| 61 | C_SE_TA_1 | true | `SendSetpointNormalized` |
| 62 | C_SE_TB_1 | true | `SendSetpointScaled` |
| 63 | C_SE_TC_1 | true | `SendSetpointFloat` |
| 64 | C_BO_TA_1 | true | `SendBitstringCommand` |

_TA_1 非简单的 _NA_1 + 13，某些 typeId 区间有跳跃。不要用算术计算，用 go-iecp5 库常量。

#### 3.2 DataType 枚举命名规范

控制命令类型必须使用 `Set` 前缀，与 proto 的 `SendXxxCommand` 命名风格对齐：

| DataType 枚举 | 说明 | 对应 proto RPC |
|--------------|------|---------------|
| `SetSingleCommand` | 单点命令 | `SendSingleCommand` |
| `SetDoubleCommand` | 双点命令 | `SendDoubleCommand` |
| `SetStepCommand` | 档位命令 | `SendStepCommand` |
| `SetSetpointNormalized` | 归一化设点 | `SendSetpointNormalized` |
| `SetSetpointScaled` | 标度化设点 | `SendSetpointScaled` |
| `SetSetpointFloat` | 浮点设点 | `SendSetpointFloat` |
| `SetBitstringCommand` | 位串命令 | `SendBitstringCommand` |

**禁止**：使用 `SingleCommandInfo`、`DoubleCommandInfo` 等无 `Set` 前缀的命名。

#### 3.3 Value 类型对照（proto → logic → client → go-iecp5）

| 命令 | Proto 类型 | Logic 转换 | Client 参数 | go-iecp5 Info struct 字段 |
|------|-----------|-----------|-------------|-------------------------|
| Single | `bool` | 直接传递 | `bool` | `SingleCommandInfo.Value: bool` |
| Double | `DoubleCommandValue` enum | `asdu.DoubleCommand(in.Value)` | `asdu.DoubleCommand` | `DoubleCommandInfo.Value: DoubleCommand` |
| Step | `StepCommandValue` enum | `asdu.StepCommand(in.Value)` | `asdu.StepCommand` | `StepCommandInfo.Value: StepCommand` |
| SetpointNormalized | `int32` | `int16(in.Value)` | `int16` | `SetpointCommandNormalInfo.Value: Normalize` |
| SetpointScaled | `int32` | `int16(in.Value)` | `int16` | `SetpointCommandScaledInfo.Value: int16` |
| SetpointFloat | `double` | `float32(in.Value)` | `float32` | `SetpointCommandFloatInfo.Value: float32` |
| Bitstring | `uint64` | `uint32(in.Value)` | `uint32` | `BitsString32CommandInfo.Value: uint32` |

#### 3.3 Qualifier 约定

- 单点/双点/档位命令使用 `QualifierOfCommand{Qual: QOCNoAdditionalDefinition, InSelect: false}`
- 设点命令使用 `QualifierOfSetpointCmd{Qual: 0, InSelect: false}`
- 位串命令无 qualifier 字段（IEC 104 协议规定 C_BO 无 QOC/QOS 字节）

#### 3.4 MQTT 广播契约

**Producer**（logic 层，自动生成后已包含）：

fire-and-forget 广播（非 ACK 方法）：
```go
err = l.svcCtx.PushPbBroadcast(l.ctx, ieccaller.IecCaller_SendXxxCommand_FullMethodName, in)
```

ACK 等待广播（7 个 typed 命令，集群模式）：
```go
var res ieccaller.SendXxxCommandRes
err = l.svcCtx.PushPbBroadcastWithAck(l.ctx, ieccaller.IecCaller_SendXxxCommand_FullMethodName, in, &res)
```

**Consumer**（mqtt/broadcast.go，必须手动添加）：

非 ACK 型（fire-and-forget）：
```go
case ieccaller.IecCaller_SendInterrogationCmd_FullMethodName:
    in := &ieccaller.SendInterrogationCmdReq{}
    err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
    // GetClient → cli.SendXxxCmd(...)
```

ACK 型（用 WithAck + publishAckReply）：
```go
case ieccaller.IecCaller_SendSingleCommand_FullMethodName:
    in := &ieccaller.SendSingleCommandReq{}
    err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
    cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
    if err != nil {
        logx.WithContext(ctx).Errorf("get client error: %v", err)
        return nil  // 非 owner 实例静默跳过
    }
    ack, err := cli.SendSingleCmd(ctx, ..., client.WithAck())
    if err != nil {
        l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", err)
        return nil
    }
    value, ok := ack.Value.(bool)
    if !ok {
        l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", fmt.Errorf("unexpected ack value type"))
        return nil
    }
    resJson, _ := jsonx.Marshal(&ieccaller.SendSingleCommandRes{Value: value})
    l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, true, string(resJson), nil)
```

**tId 约定**：使用 `BroadcastBody.Tid` 做为请求-响应关联 key。Originator 生成 UUID（`tool.SimpleUUID()`），在 `BroadcastReplyPool` 注册，consumer 原样回写到 `BroadcastAckBody.Tid`，ACK consumer 从中读取解析 reply pool。tId 不使用 OTel trace ID（避免无 trace 上下文时断裂）。

### 4. Validation & Error Matrix

| 条件 | 错误 |
|------|------|
| host 为空 | `tool.NewErrorByPbCodeWrap(Code__1_06_RPC, ...)` |
| port 不在 1-65535 | ClientManager 拒绝连接 |
| value 超出类型范围（如 int16 超限） | `cast.ToXxxE` 返回 error |
| typeId 不在 doSend switch 中 | `fmt.Errorf("unknown type id %d")` |
| 客户端未连接 | `NotConnected` sentinel error |
| ACK 超时（`antsx.ErrReplyExpired`） | `wrapCommandAckError` → `Code__1_00_TIMEOUT` (504) |
| 同一控制点已有未完成命令（`antsx.ErrDuplicateID`） | `wrapCommandAckError` → `Code__1_05_BIZ_REPEAT` (409) |
| ACK 被拒绝（从站返回 IsNegative=true） | `wrapCommandAckError` → `Code__1_05_BIZ_STATE` (409) via `CommandRejectedError` |
| ACK 意外 COT | `wrapCommandAckError` → `Code__1_05_BIZ_STATE` (409) via `CommandRejectedError` |

#### 4.1 CommandRejectedError

IEC 从站拒绝命令时，`clienthandler.go` 的 `resolveCommandAck` 创建 `CommandRejectedError`，携带完整 ACK 元数据：

```go
// common/iec104/client/errors.go
type CommandRejectedError struct {
    TypeID     int
    Coa        uint
    Ioa        uint
    Cot        string
    CotCause   int
    IsNegative bool
    Status     CommandAckStatus
}
```

`wrapCommandAckError` 通过 `errors.As(err, &rejected)` 匹配该类型，映射到 `Code__1_05_BIZ_STATE` (409)。
Logic 层**不需要**单独处理此类型；拦截器统一通过 `%+v` 打印完整错误链。

### 5. Good/Base/Bad Cases

**Good** — 新增 typed RPC 完整示例（以 SendDoubleCommand 为例）：
```
proto: rpc SendDoubleCommand(SendDoubleCommandReq) returns (SendDoubleCommandRes);
       message SendDoubleCommandReq { DoubleCommandValue value; bool withTime; }
       message SendDoubleCommandRes { DoubleCommandValue value = 1; }
client: func (c *Client) SendDoubleCmd(ctx, coa, ioa, asdu.DoubleCommand, withTime, ...CommandOption) (*CommandAck, error)
logic:  ack, err := cli.SendDoubleCmd(l.ctx, uint16(in.Coa), ioa, asdu.DoubleCommand(in.Value), in.WithTime, client.WithAck())
        value, err := ackDoubleCommandValue(ack.Value)
server: func (s *IecCallerServer) SendDoubleCommand(ctx, *SendDoubleCommandReq) (*SendDoubleCommandRes, error)
mqtt:  case IecCaller_SendDoubleCommand_FullMethodName: ... cli.SendDoubleCmd(ctx, ...)
```

**Base** — 现有 `SendCommand` 通用接口（保留用于选控、自定 typeId 等高级场景）。

**Bad** — 只加 proto + logic，不加 mqtt/broadcast.go consumer case → cluster 部署时其他实例收不到广播命令。

### 6. Tests Required

- [ ] `gen.sh` 执行零错误（每次 proto 变更后）
- [ ] `go build ./...` 零错误（ieccaller 和 common/iec104/client 两个包）
- [ ] `go vet ./...` 零 warning
- [ ] server handler 方法未被 gen.sh 覆盖（`grep -c 'SendXxxCommand' server/*.go`）
- [ ] mqtt/broadcast.go 每个新 RPC 有对应 case
- 集成测试（需真实 IEC 104 从站或模拟器）：gRPC 调用 → 从站返回激活确认 COT=7

---

## Scenario: 集群 Broadcast ACK Reply (MQTT)

### 1. Scope / Trigger

- Trigger: 集群部署下，ACK 型控制命令的 broadcast 分支通过 MQTT 发布，owner 实例通过 per-instance ACK topic 回传结果。
- Scope: ServiceContext 的 MQTT 初始化、`PushPbBroadcastWithAck`、`BroadcastReplyPool`；`mqtt/broadcast.go` 消费者用 `client.WithAck()` 执行并 `publishAckReply`；`mqtt/broadcast_ack.go` 消费 ACK 并 resolve replypool。

### 2. Signatures

**BroadcastBody / BroadcastAckBody**：
```go
type BroadcastBody struct {
    Tid      string `json:"tId,omitempty"`  // 请求-响应关联 UUID，ACK 型必填
    AckTopic string `json:"ackTopic"`        // originator 的 ACK topic（per-instance）
    Method   string `json:"method"`
    Body     string `json:"body"`
}

type BroadcastAckBody struct {
    Tid          string `json:"tId"`          // 原样回写 originator 的 tId
    Method       string `json:"method"`
    Success      bool   `json:"success"`
    ResponseBody string `json:"responseBody"`
    Error        string `json:"error,omitempty"`
    ErrorKind    string `json:"errorKind,omitempty"`
}
```

**PushPbBroadcastWithAck**：
```go
func (svc ServiceContext) PushPbBroadcastWithAck(ctx context.Context, method string, in any, res any) error
```
- 生成 tId (UUID) → 注册 BroadcastReplyPool → pushBroadcast(带 tId) → Await → 检查 ErrorKind → 解析 ResponseBody 到 res

**pushBroadcast**：
```go
func (svc ServiceContext) pushBroadcast(ctx context.Context, method string, in any, optTid ...string) error
```
- 构建 BroadcastBody（需置入 AckTopic=topic）→ PublishWithTrace → 发布到固定广播 topic `iec/broadcast`

**publishAckReply helper**：
```go
func (l *Broadcast) publishAckReply(ctx context.Context, tId, ackTopic, method string, success bool, responseBody string, ackErr error)
```
- 构建 BroadcastAckBody（回写 tId）→ PublishWithTrace → 发布到 `ackTopic`（即 originator 的 `broadcastBody.AckTopic`）

### 3. Contracts

**MQTT 初始化契约**（`NewServiceContext`）：
- 预生成 UUID（`random.UUIdV4()`）作为 `broadcastInstanceId` = `"iec-caller-" + uid`
- 同一 UUID 作为 `cfg.ClientID`——MQTT client 和 broadcast instance 使用同一 ID
- 集群模式 `cfg.Qos = 1`（覆盖配置的 `Qos:0`，广播必须可靠）
- 只创建一个 `MqttClient` 实例，同时服务 ASDU 推送和广播
- `broadcastTopic` = `"iec/broadcast"`（固定，所有实例共享订阅）
- `broadcastAckTopic` = `"iec/broadcast-ack/{instanceId}"`（per-instance，只订阅自己的 ACK）

**tId 传递契约**：使用 `BroadcastBody.Tid` 显式传递，不在 OTel trace context 中隐式依赖。
- **Why**：OTel trace 在无 gRPC context 时不生成，以 trace ID 做 reply pool key 会断裂。UUID 保证唯一且可靠。
- `PushPbBroadcastWithAck` → `tool.SimpleUUID()` → 注册 replypool + 写入 `BroadcastBody.Tid`
- `publishAckReply` → 参数接收 tId → 写入 `BroadcastAckBody.Tid`
- `broadcast_ack.go` → 读取 `ackBody.Tid` → `Resolve(tId, ackBody)`

**ACK topic 路由契约**：
- Originator 将自身 `AckTopic`（`iec/broadcast-ack/{myId}`）写入 `BroadcastBody.AckTopic`
- Consumer 发布 ACK 时**必须使用** `broadcastBody.AckTopic`（originator 的 topic），**禁止**使用自身 `BroadcastAckTopic()`
- **Why 禁止**：consumer 的 `BroadcastAckTopic()` 指向自己的 topic（`.../B`），发到这里 originator（`.../A`）永远收不到 → 命令超时

**自广播过滤契约**：
- 所有实例订阅 `iec/broadcast`，发布的实例也会收到自己的消息
- 使用 `broadcastBody.AckTopic == l.svcCtx.BroadcastAckTopic()` 判断自广播——两者 ID 相同则为自身
- **Why AckTopic 而非 BroadcastGroupId**：`AckTopic` 已包含 instance ID，无需冗余字段

**多实例竞争保护**：非 owner 实例在 `GetClient` 失败时静默跳过（return nil），不发送 ACK reply。只有持有目标 client 的实例执行 `client.WithAck()` 并发布 response。

**ErrorKind 映射**：
- `publishAckReply` 对传入 error 做 `errors.Is(err, antsx.ErrReplyExpired)` → ErrorKind="timeout"
- `publishAckReply` 对传入 error 做 `errors.Is(err, antsx.ErrDuplicateID)` → ErrorKind="duplicate"
- 其他 error → ErrorKind="unknown"

**PushPbBroadcastWithAck 错误处理**：
- `ack.Success==false` + ErrorKind="timeout" → 返回 `antsx.ErrReplyExpired`
- `ack.Success==false` + ErrorKind="duplicate" → 返回 `antsx.ErrDuplicateID`
- `ack.Success==false` + 其他 → 返回 `fmt.Errorf("broadcast command error: %s", ack.Error)`

**PublishWithTrace 追踪**：
- 使用 `mqttx.Client.PublishWithTrace` 发布广播和 ACK，自动在 Message headers 注入 OTel span context
- Consumer 端 `processMessage` 自动解包并还原 trace context，无需手动处理
- PublishWithTrace 返回的 traceID 仅用于日志，不用于 reply pool 关联

### 4. Validation & Error Matrix

| 条件 | 错误 |
|------|------|
| 非 cluster 模式调用 PushPbBroadcastWithAck | `fmt.Errorf("not in cluster mode")` |
| MqttClient 为 nil | `fmt.Errorf("mqtt client is nil")` |
| BroadcastReplyPool 为 nil | `fmt.Errorf("broadcast reply pool is nil")` |
| tId 生成失败 | `fmt.Errorf("register reply pool error: %w", err)` |
| replypool 注册重复 tId | `fmt.Errorf("register reply pool error: %w", err)` |
| MQTT 推送广播失败 | 仅日志 Error，不阻塞 |
| 远程 ACK 超时（promise.Await 超时） | `wrapCommandAckError` → `Code__1_00_TIMEOUT` |
| 远程命令重复 pending | ErrorKind="duplicate" → `Code__1_05_BIZ_REPEAT` |
| 远程命令执行/解析失败 | ErrorKind="unknown" → `Code__1_06_THIRD_PARTY` |
| GetClient 失败（本实例无目标 client） | 静默跳过（return nil），不发 ACK |
| publishAckReply ackTopic 为空 | 静默跳过 |

### 5. Good/Base/Bad Cases

**Good** — 集群 ACK 完整流程：
```
1. logic: cli==nil → PushPbBroadcastWithAck → 生成 tId="uuid-A" → 注册 replypool
2. pushBroadcast: broadcastBody{AckTopic:".../A", Tid:"uuid-A"} → PublishWithTrace → iec/broadcast
3. owner 实例(B): Consume → processMessage 解包还原 trace → 收到 broadcastBody
4. owner 实例(B): GetClient(成功) → cli.SendXxxCmd(WithAck()) → Await → ack.Value
5. owner 实例(B): publishAckReply(ctx, "uuid-A", ".../A", method, true, resJson, nil)
6. PublishWithTrace → iec/broadcast-ack/A
7. origin 实例(A): broadcast_ack.go → Consume → ackBody.Tid="uuid-A"
8. Resolve("uuid-A", ackBody) → PushPbBroadcastWithAck: Await → success
9. unmarshal → return typed res
```

**Base** — owner 实例执行命令失败：
```
1-4. 同上
5. owner 实例(B): publishAckReply(ctx, "uuid-A", ".../A", method, false, "", err)
6-8. PushPbBroadcastWithAck: ErrorKind 映射 → 返回对应 sentinel error
```

**Bad** — ACK 发到错误 topic（已修复）：
```go
// Wrong: publishAckReply 用自身 ack topic，ACK 发回自己
ackTopic := l.svcCtx.BroadcastAckTopic()  // ".../B"
PublishWithTrace(ctx, ackTopic, data)      // → iec/broadcast-ack/B（实例 A 永远收不到）

// Correct: 从 broadcastBody 读取 originator 的 ack topic
l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, ...)
// → PublishWithTrace(ctx, broadcastBody.AckTopic, data) → iec/broadcast-ack/A ✅
```

**Bad** — 依赖 OTel trace context 做 reply pool 关联（已修复）：
```go
// Wrong: 无 OTel trace 时，每个 startSpan 创建新 trace ID，reply pool key 断裂
traceId := TraceIdFromContext(ctx)  // originator: "A", consumer: "B"（不匹配）
pool.Register(traceId)             // originator: "A"
pool.Resolve(traceId, ackBody)     // consumer: "B" → 找不到

// Correct: 使用显式 UUID tId
tId, _ := tool.SimpleUUID()       // 唯一，不变
pool.Register(tId)
// ... consumer 原样回写 BroadcastAckBody.Tid = tId
pool.Resolve(ackBody.Tid, ackBody) // 精确匹配 ✅
```

**Bad** — 在 `BroadcastAckBody` 中冗余存储 `BroadcastGroupId`（已移除）：
```go
// Wrong: BroadcastGroupId 是 Kafka 遗留，MQTT 已不需要
// - 自广播过滤改用 AckTopic 比较
// - ACK 路由由 per-instance topic 订阅天然隔离
// - ackBody.BroadcastGroupId 的检查完全冗余

// Correct: BroadcastBody / BroadcastAckBody 均无 BroadcastGroupId 字段
```

**Bad** — 为 broadcast 创建独立 MQTT client（已修复）：
```go
// Wrong: 两个 client 连接同一个 broker，浪费资源且 ClientID 不一致
svcCtx.MqttClient = mqttx.MustNewClient(cfg)           // ClientID=auto
svcCtx.MqttBroadcastClient = mqttx.MustNewClient(cfg2) // ClientID=auto2

// Correct: 单 client，预生成 UUID 作为 ClientID=broadcastInstanceId
uid, _ := random.UUIdV4()
svcCtx.broadcastInstanceId = "iec-caller-" + uid
cfg.ClientID = svcCtx.broadcastInstanceId  // 统一
svcCtx.MqttClient = mqttx.MustNewClient(cfg)
```

### 6. Tests Required

- [ ] `go build ./app/ieccaller/...` 零错误
- [ ] `PushPbBroadcastWithAck` 在非 cluster 模式返回 error
- [ ] `PushPbBroadcastWithAck` timeout/duplicate ErrorKind 映射正确
- [ ] `publishAckReply` 空 tId 不发送 ACK
- [ ] `BroadcastReplyPool.Close` 在 `ServiceContext.Close` 中调用
- [ ] 自广播过滤：`broadcastBody.AckTopic == BroadcastAckTopic()` 逻辑正确
- [ ] ACK 路由：ACK 发到 `broadcastBody.AckTopic` 而非 `BroadcastAckTopic()`
- [ ] 非 ACK 型 broadcast（总召等）保持 fire-and-forget
- [ ] 残留引用检查：`KafkaBroadcastAckPusher`、`MqttBroadcastConfig`、`MqttBroadcastClient`、`BroadcastGroupId` 全部移除

### 7. Wrong vs Correct

**Wrong** — 把 PushPbBroadcastWithAck 用在非 ACK 型方法：
```go
// Wrong: SendCommand 没有 ACK 返回值
if cli == nil && l.svcCtx.IsBroadcast() {
    var res ieccaller.SendCommandRes  // 空结构体
    err = l.svcCtx.PushPbBroadcastWithAck(l.ctx, ..., in, &res)
    ...
}
```

**Correct** — 非 ACK 型方法继续用 fire-and-forget：
```go
// Correct: SendCommand 保持 fire-and-forget
if cli == nil && l.svcCtx.IsBroadcast() {
    err = l.svcCtx.PushPbBroadcast(l.ctx, ..., in)
    ...
}
```

**Correct** — 新增 MQTT consumer case 模板：
```go
case ieccaller.IecCaller_SendDoubleCommand_FullMethodName:
    in := &ieccaller.SendDoubleCommandReq{}
    err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
    cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
    if err != nil {
        return nil  // 非 owner 静默跳过
    }
    ack, err := cli.SendDoubleCmd(ctx, uint16(in.Coa), ioa, asdu.DoubleCommand(in.Value), in.WithTime, client.WithAck())
    if err != nil {
        l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", err)
        return nil
    }
    value, ok := ack.Value.(asdu.DoubleCommand)
    if !ok {
        l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, false, "", fmt.Errorf("unexpected ack value type"))
        return nil
    }
    resJson, _ := jsonx.Marshal(&ieccaller.SendDoubleCommandRes{Value: ieccaller.DoubleCommandValue(int32(value))})
    l.publishAckReply(ctx, broadcastBody.Tid, broadcastBody.AckTopic, broadcastBody.Method, true, string(resJson), nil)
```

---

## Scenario: ACK replyPool + Command Option + Helpers

### 1. Scope / Trigger

- Trigger: 所有 typed 控制命令（SendSingleCommand 等 7 个）需要同步等待从站 ACK。
- Scope: Client 层 Option 模式、Logic 层 helper 函数、pool 封装。

### 2. Signatures

**CommandOption 类型**：
```go
type CommandOption func(*commandOptions)

type commandOptions struct {
    awaitAck bool
}

func WithAck() CommandOption
```

**Client sendWithAck 内部 helper**：
```go
func (c *Client) sendWithAck(ctx context.Context, coa uint16, typeId asdu.TypeID, ioa asdu.InfoObjAddr, cmd *command, opts []CommandOption) (*CommandAck, error)
```

**Pool 对外 API**（仅限 clienthandler 内部使用）：
```go
func (c *Client) ResolveCommandAck(key string, ack *CommandAck) bool
func (c *Client) RejectCommandAck(key string, err error) bool
func (c *Client) HasCommandAck(key string) bool
```

**wrapCommandAckError helper**：
```go
func wrapCommandAckError(err error, fallbackMsg string) error
// errors.Is(err, antsx.ErrReplyExpired) → Code__1_00_TIMEOUT
// errors.Is(err, antsx.ErrDuplicateID) → Code__1_05_BIZ_REPEAT
// default → Code__1_06_THIRD_PARTY
```

**ackXxxValue type assertion helpers**：
```go
func ackBoolValue(ackValue any) (bool, error)           // SendSingleCommand
func ackDoubleCommandValue(ackValue any) (asdu.DoubleCommand, error)   // SendDoubleCommand
func ackStepCommandValue(ackValue any) (asdu.StepCommand, error)       // SendStepCommand
func ackSetpointNormalizedValue(ackValue any) (asdu.Normalize, error)  // SendSetpointNormalized
func ackInt16Value(ackValue any) (int16, error)         // SendSetpointScaled
func ackFloat32Value(ackValue any) (float32, error)     // SendSetpointFloat
func ackUint32Value(ackValue any) (uint32, error)       // SendBitstringCommand
```

### 3. Contracts

**WithAck 行为契约**：
- 通过 `WithAck()` option 启用，发送前在 `cmdReplyPool` 注册 key（coa:typeId:ioa）。
- `sendWithAck` 内部 Register → doSend → Await 三步。
- `doSend` 失败不显式 Reject（pool TTL 10s 自动清理）。
- `Await` 返回的 error 可能来自：签名冲突（ErrDuplicateID）、ACK 超时（ErrReplyExpired）、从站拒绝（RejectCommandAck → fmt.Errorf）、意外 COT（RejectCommandAck → fmt.Errorf）。

**无需 WithAck 的场景**：
- `SendCommand`（通用 typeId 接口）：不接入 replyPool，cluster 模式下 fire-and-forget。
- 非 ACK 型 gRPC 方法（总召、累计量召、读命令、测试命令、清缓存）：cluster 模式下 fire-and-forget。
- cli == nil && !IsBroadcast：业务层直接返回空 Res。

**需要 WithAck 的场景**：
- 本地直连（cli != nil）：7 个 typed 命令使用 `cli.SendXxxCmd(..., client.WithAck())`，通过 `cmdReplyPool` 等待。
- 集群广播（cli == nil && IsBroadcast）：7 个 typed 命令使用 `PushPbBroadcastWithAck`，owner 实例通过 `client.WithAck()` 执行并发布 ACK reply。broadcast consumer 内部也使用 WithAck，与本地直连路径一致。

**Logic 层标准模板**：
```go
ack, err := cli.SendXxxCmd(l.ctx, uint16(in.Coa), asdu.InfoObjAddr(in.Ioa), in.Value, in.WithTime, client.WithAck())
if err != nil {
    return nil, wrapCommandAckError(err, "IEC发送xxx命令失败")
}
value, err := ackXxxValue(ack.Value)
if err != nil {
    return nil, wrapCommandAckError(err, "IECxxx命令ACK解析失败")
}
return &ieccaller.SendXxxCommandRes{Value: value}, nil
```

### 4. Validation & Error Matrix

| 条件 | error type | gRPC 错误码 |
|------|-----------|------------|
| ACK 超时（pool TTL 到期） | `antsx.ErrReplyExpired` | `Code__1_00_TIMEOUT` |
| 同一 key 重复注册 | `antsx.ErrDuplicateID` | `Code__1_05_BIZ_REPEAT` |
| 从站拒绝（IsNegative=true） | `fmt.Errorf("command rejected: ...")` | `Code__1_06_THIRD_PARTY` |
| 意外 COT（非 ActivationCon） | `fmt.Errorf("unexpected COT: ...")` | `Code__1_06_THIRD_PARTY` |
| 发送失败（doSend error） | 原始 error | `Code__1_06_THIRD_PARTY` |
| ACK value 类型断言失败 | `fmt.Errorf("unexpected ... type")` | `Code__1_06_THIRD_PARTY` |

### 5. Good/Base/Bad Cases

**Good** — 正常 ACK 流程（使用 `WithAck()`）：
```
1. logic: cli.SendSingleCmd(ctx, coa, ioa, value, withTime, WithAck())
2. sendWithAck: Register(key) → 成功
3. sendWithAck: doSend(C_SC_NA_1) → 成功
4. 从站返回 C_SC_NA_1, COT=ActivationCon, IsNegative=false
5. clienthandler.resolveCommandAck: ResolveCommandAck(key, ack) → ack.Value=bool
6. sendWithAck: Await(ctx) → (*CommandAck, nil)
7. logic: ackBoolValue(ack.Value) → (in.Value, nil)
8. logic: return &SendSingleCommandRes{Value: in.Value}, nil
```

**Base** — 从站拒绝命令：
```
1. logic: cli.SendSingleCmd(ctx, coa, ioa, value, withTime, WithAck())
2-3. 同上
4. 从站返回 C_SC_NA_1, COT=ActivationCon, IsNegative=true
5. clienthandler: RejectCommandAck(key, fmt.Errorf("command rejected: ..."))
6. sendWithAck: Await(ctx) → (nil, error)
7. logic: wrapCommandAckError(err) → Code__1_06_THIRD_PARTY
```

**Bad** — 对 `WithAck()` 返回的 error 直接用 `tool.NewErrorByPbCodeWrap`，不区分超时/重复/拒绝（已修复）：
```go
// Wrong
if err != nil {
    return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "IEC命令失败")
}

// Correct
if err != nil {
    return nil, wrapCommandAckError(err, "IEC命令失败")
    // 自动区分超时 vs 重复 vs 拒绝
}
```

### 6. Tests Required

- [ ] `wrapCommandAckError` 对 `ErrReplyExpired` 返回 `Code__1_00_TIMEOUT`
- [ ] `wrapCommandAckError` 对 `ErrDuplicateID` 返回 `Code__1_05_BIZ_REPEAT`
- [ ] `wrapCommandAckError` 对普通 error 返回 `Code__1_06_THIRD_PARTY`
- [ ] 每个 `ackXxxValue` helper 正确类型断言

---

### 7. Wrong vs Correct

**Wrong** — 遗漏 MQTT consumer case：
```go
// mqtt/broadcast.go 的 switch 中没有新命令的 case
default:
    logx.WithContext(ctx).Errorf("unknown method:%s", broadcastBody.Method)
    // cluster 部署时其他实例收到广播后丢弃，命令丢失
```

**Correct** — 每个新 RPC 都在 consumer 中注册：
```go
case ieccaller.IecCaller_SendDoubleCommand_FullMethodName:
    in := &ieccaller.SendDoubleCommandReq{}
    err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
    // ... GetClient + SendDoubleCmd
```

---

## Design Decision: withTime 字段 vs 分离 RPC

**Context**: 控制方向每种命令有 2 个 typeId（_NA_1 不带时标 / _TA_1 带时标）。选择用一个 RPC + `withTime` 字段还是 14 个独立 RPC？

**Options Considered**:
1. 14 个 RPC（每种命令拆成 2 个）：接口膨胀，但参数完全无歧义
2. 7 个 RPC + `withTime` bool 字段：接口精简，调用方只需知道传 true/false

**Decision**: 选项 2。绝大多数场景用 `withTime=false`（默认），高级场景才用 true。typeId 映射在 client 层完成，调用方无需关心 45 vs 58 的区别。

**Example**:
```go
func (c *Client) SendSingleCmd(ctx context.Context, coa uint16, ioa asdu.InfoObjAddr, value bool, withTime bool, opts ...CommandOption) (*CommandAck, error) {
    typeId := asdu.C_SC_NA_1 // 45
    if withTime {
        typeId = asdu.C_SC_TA_1 // 58
    }
    return c.sendWithAck(ctx, coa, typeId, ioa, &command{typeId: typeId, ...}, opts)
}
```

---

## Convention: Enum 值对齐 go-iecp5 库常量

| Proto enum | Proto 值 | go-iecp5 常量 | 语义 |
|-----------|---------|--------------|------|
| `DoubleCommandValue.DCO_NOT_ALLOWED` | 0 | `DCONotAllow0` | 不允许 |
| `DoubleCommandValue.DCO_ON` | 1 | `DCOOn` | 合 |
| `DoubleCommandValue.DCO_OFF` | 2 | `DCOOff` | 分 |
| `StepCommandValue.SCO_NOT_ALLOWED` | 0 | `SCONotAllow0` | 不允许 |
| `StepCommandValue.SCO_DOWN` | 1 | `SCOStepDown` | 降一步 |
| `StepCommandValue.SCO_UP` | 2 | `SCOStepUP` | 升一步 |

proto enum 首值必须为 0（protobuf 规范），与 go-iecp5 的 `*NotAllow0` 常量对齐。

---

## Gotcha: gen.sh 不覆盖手动添加的 server handler

> **Warning**: `gen.sh` 生成的 `ieccallerserver.go` 只包含 `IecCallerServer` 结构体声明和注册方法。手动添加的 handler 方法（`func (s *IecCallerServer) SendXxxCommand(...)`）不会被覆盖。
>
> 但每次 gen.sh 后应验证：`grep -c 'SendXxx' internal/server/ieccallerserver.go` 数值不变。

---

## Scenario: ASDU 回执处理

### 1. Scope / Trigger

- Trigger: 每次新增控制方向命令或修改 `OnASDU` 处理逻辑
- Scope: 控制命令回执（C_* typeId + COT=7/9/10）、读命令负响应（C_RD_NA_1 + negative COT）的接收、日志、推送决策

### 2. Signatures

**go-iecp5 cs104 调度逻辑**（`cs104/client.go` 518-541行）：
```go
switch asduPack.Identifier.Type {
case asdu.C_IC_NA_1:    → InterrogationHandler     // 100
case asdu.C_CI_NA_1:    → CounterInterrogationHandler // 101
case asdu.C_RD_NA_1:    → ReadHandler               // 102
case asdu.C_CS_NA_1:    → ClockSyncHandler           // 103
case asdu.C_TS_NA_1:    → TestCommandHandler         // 104 (注意不是 107)
case asdu.C_RP_NA_1:    → ResetProcessHandler        // 105
case asdu.C_CD_NA_1:    → DelayAcquisitionHandler    // 106
default:                → ASDUHandler                // 所有其他
}
```

**OnASDU 数据类型路由**（`app/ieccaller/internal/iec/clienthandler.go`）：
```
C_RD_NA_1       (读命令负响应) → onReadResponse → 仅日志，negative=true 打 Error
DataType 0-11  (M_* 监视) → pushASDU → Kafka/MQTT
DataType 12    (C_SC)     → onCommandAck → 仅日志
DataType 13    (C_DC)     → onCommandAck → 仅日志
DataType 14    (C_RC)     → onCommandAck → 仅日志
DataType 15    (C_SE_NA)  → onCommandAck → 仅日志
DataType 16    (C_SE_NB)  → onCommandAck → 仅日志
DataType 17    (C_SE_NC)  → onCommandAck → 仅日志
DataType 18    (C_BO)     → onCommandAck → 仅日志
DataType 19    (M_EI)     → onEndOfInitialization → 仅日志
DataType 20    (UNKNOWN)  → onUnknownASDU → 仅日志
```

### 3. Contracts

**COT（CauseOfTransmission）字段**：

| COT 值 | 含义 | 触发时机 |
|--------|------|---------|
| 1 (Periodic) | 周期上送 | 监视方向 M_* 周期数据 |
| 3 (Spontaneous) | 突发/变化上送 | 监视方向 M_* 变化数据 |
| 5 (Request) | 请求/读响应 | C_RD_NA_1 请求或读成功后的 M_* 响应 |
| 6 (Activation) | 主站下发激活 | 命令发出时（COT 由主站设置） |
| 7 (ActivationCon) | 从站确认接收 | 从站收到命令后立即返回 |
| 9 (DeactivationCon) | 从站取消确认 | 选控场景下取消命令 |
| 10 (ActivationTerm) | 从站确认执行完成 | 命令执行完成后 |
| 44 (UnknownTypeID) | 未知类型 | 从站拒绝请求或命令 |
| 45 (UnknownCOT) | 未知传送原因 | 从站拒绝请求或命令 |
| 46 (UnknownCA) | 未知公共地址 | 从站拒绝请求或命令 |
| 47 (UnknownIOA) | 未知信息对象地址 | 读不可读点、控制/请求未知 IOA |

**COT + IsNegative 使用边界**：

- `cot`/`cotCause`/`isNegative` 属于 ASDU envelope 日志字段，统一由 `asduLogContext` 注入。
- `isNegative=true` 用于判断请求/命令被从站否定：控制 ACK、读命令负响应、总召/时钟/测试/复位等系统命令响应。
- M_* 监视数据不要用 `isNegative` 决定是否丢弃；监视数据质量必须读取信息体质量位（QDS/QDP：`iv/nt/sb/bl/ov`）。
- `C_RD_NA_1` 读成功通常返回实际 M_* ASDU；只有失败时才返回 `C_RD_NA_1 + IsNegative=true + COT=44..47`。

**回执日志格式**：
```
Command ACK received asdu=C_SC_NA_1 typeId=45 coa=1 cot=ActivationCon isNegative=false
Read command rejected asdu=C_RD_NA_1 typeId=102 coa=1 ioa=10 cot=UnknownIOA cotCause=47 isNegative=true
```

**onCommandAck 不推送到 Kafka**：回执 value 是主站原始值的回声，不包含从站实际状态。消费者应通过 M_* 监视数据确认命令效果。

**onReadResponse 不推送到 Kafka**：读失败回执只说明请求失败；读成功数据会以实际 M_* ASDU 进入现有监视数据处理链路并按点位推送。

### 4. Validation & Error Matrix

| 条件 | 处理 |
|------|------|
| C_* typeId + COT=ActivationCon + IsNegative=true | 日志告警（从站拒绝命令） |
| C_* typeId + COT=ActivationCon + IsNegative=false | 日志 Info（正常确认） |
| C_RD_NA_1 + IsNegative=true + COT=44..47 | onReadResponse 日志 Error（读请求被拒绝） |
| C_RD_NA_1 读成功 | 从站返回实际 M_* ASDU，走监视数据处理并推送 |
| M_* 监视 ASDU + QDS/QDP 标记无效 | 仍按监视数据链路处理，质量位写入 body 供消费者判断 |
| 未知 typeId | onUnknownASDU 日志 Info |
| M_EI_NA_1 | onEndOfInitialization 日志 Info |

### 5. Good/Base/Bad Cases

**Good** — 正常的命令回执流程：
```
1. gRPC SendSingleCommand → doSend(C_SC_NA_1, COT=Activation)
2. 从站返回 C_SC_NA_1, COT=ActivationCon, IsNegative=false → onCommandAck 日志
3. 从站返回 C_SC_NA_1, COT=ActivationTerm → onCommandAck 日志
4. 从站返回 M_SP_NA_1 (监视更新) → pushASDU → Kafka 推送
```

**Base** — 命令被拒绝：
```
1. gRPC SendSingleCommand → doSend(C_SC_NA_1, COT=Activation)
2. 从站返回 C_SC_NA_1, COT=ActivationCon, IsNegative=true → onCommandAck 日志告警
```

**Bad** — 默认分支静默丢弃（已修复）：
```go
// 旧代码：default: return（所有 C_* 回执静默丢失）
// 新代码：onCommandAck 记录 COT + isNegative，不推送但有日志
```

**Bad** — 把读命令负响应当成未知 ASDU 或普通数据：
```go
// Wrong: C_RD_NA_1 + IsNegative=true + COT=47 落到 Unknown ASDU，无法直接看到 IOA
c.onUnknownASDU(ctx)

// Correct: 单独记录读失败，日志包含 ioa，公共 context 包含 cot/cotCause/isNegative
c.onReadResponse(ctx, packet)
```

**Bad** — 用 `isNegative` 过滤监视数据：
```go
// Wrong: 监视数据质量不是用 IsNegative 判断
if packet.Coa.IsNegative { return }

// Correct: 解析信息体 QDS/QDP，并把 iv/nt/sb/bl/ov 传给下游
obj.Iv = util.QdsIsInvalid(p.Qds)
```

### 6. Tests Required

- [ ] `go build ./...` 通过（handle.go + clienthandler.go 变更后）
- [ ] `go vet ./...` 零 warning
- [ ] `go test ./app/ieccaller/internal/iec` 通过，覆盖 `asduLogContext` 统一字段
- 模拟器测试：发送命令 → 验证日志出现 `Command ACK received` + 正确 COT
- 模拟器测试：读不可读 IOA → 验证日志出现 `Read command rejected` + `cot=UnknownIOA` + `isNegative=true` + `ioa`
- 模拟器测试：从站回 M_SP_NA_1 → 验证 Kafka 推送正常

### 7. Wrong vs Correct

**Wrong** — 将命令回执推送到现有 Kafka 遥测流：
```go
case client.SingleCommandInfo:
    c.onSinglePoint(ctx, packet) // 复用监视数据处理 → 消费者混淆
```

**Correct** — 命令回执仅记录日志，不混入遥测流：
```go
case client.SingleCommandInfo, client.DoubleCommandInfo, ...:
    c.onCommandAck(ctx, packet) // 仅日志，消费者通过 M_* 确认效果
```

**Correct** — COT 三字段作为通用 ASDU 日志上下文，handler 只追加业务字段：
```go
func (c *ClientCall) asduLogContext(ctx context.Context, packet *asdu.ASDU) context.Context {
    return logx.ContextWithFields(ctx,
        logx.Field("cot", genCOTName(packet.Coa.Cause)),
        logx.Field("cotCause", int(packet.Coa.Cause)),
        logx.Field("isNegative", packet.Coa.IsNegative),
    )
}

func (c *ClientCall) onReadResponse(ctx context.Context, packet *asdu.ASDU) {
    ioa := packet.GetReadCmd()
    logx.WithContext(ctx).Errorw("Read command rejected", logx.Field("ioa", uint(ioa)))
}
```

---

## Gotcha: go-iecp5 C_TS 调度不匹配

> **Warning**: 项目发送测试命令使用 `C_TS_TA_1`（typeId=107, 带 CP56Time2a），但 go-iecp5 cs104 调度器仅匹配 `C_TS_NA_1`（typeId=104）到 `TestCommandHandler`。
>
> 若从站回执 `C_TS_TA_1`，将走 `ASDUHandler` 通用路径 → `GetDataType` 返回 `UNKNOWN` → `onUnknownASDU` 日志记录。功能不受影响（日志可见），但非预期路径。
>
> 此限制来自 go-iecp5 库，非项目代码问题。

---

## Gotcha: DataType 编号影响下游消费者

> **Warning**: `handle.go` 中 `DataType` iota 编号直接影响 Kafka 消息中的 `dataType` 字段（`int(client.GetDataType(packet.Type))`）。
>
> 新增 DataType **必须追加到末尾**（`UNKNOWN` 之前），不能插入中间，否则所有控制方向 DataType 编号移位，下游消费者解析错误。
>
> 当前编号：0-11=监视，12-18=控制命令（Set 前缀），19=初始化结束，20=UNKNOWN。

---

## Convention: IoaHexAddress 格式化规范

**What**: IOA（信息对象地址）的十六进制表示必须使用 6 位格式（`%06x`）。

**Why**: 文档中唯一键生成规则定义为 `host_coa_0x{ioa:06X}`，示例为 `127.0.0.1_1_0x0007D1`。使用 4 位格式会导致短 IOA（如 `0x0F`）输出 `0x000f` 而非 `0x00000f`，与文档不一致。

**Example**:
```go
// Correct
func IoaHexAddress[T ~uint](ioa T) string {
    return fmt.Sprintf("0x%06x", ioa)
}

// Wrong - 使用 %04x 会导致短 IOA 格式不一致
func IoaHexAddress[T ~uint](ioa T) string {
    return fmt.Sprintf("0x%04x", ioa) // ioa=0x0F → "0x000f" (应为 "0x00000f")
}
```

**Related**: `types.GetKey()` 使用此函数生成唯一键，消费者依赖固定长度格式。

---

## Convention: QDS/QDP 工具函数逻辑

**What**: `QdsContainsAll` 和 `QdpContainsAll` 函数检查是否包含**所有**指定标志位。

**Why**: 逻辑反转是常见 bug。正确逻辑是"缺少任意 flag 就返回 false"。

**Example**:
```go
// Correct - 检查是否包含所有 flag
func QdsContainsAll(qds asdu.QualityDescriptor, flags ...asdu.QualityDescriptor) bool {
    for _, flag := range flags {
        if (qds & flag) != flag {  // 缺少该 flag
            return false
        }
    }
    return true
}

// Wrong - 逻辑反转（包含任意 flag 就返回 false）
func QdsContainsAll(qds asdu.QualityDescriptor, flags ...asdu.QualityDescriptor) bool {
    for _, flag := range flags {
        if (qds & flag) == flag {  // 包含该 flag → 返回 false（错误）
            return false
        }
    }
    return true
}
```

---

## Gotcha: go-zero Starter 接口约束

> **Warning**: go-zero 的 `service.Starter` 接口定义为 `Start()` 无返回值。实现该接口时不能返回 error。
>
> ```go
> // go-zero 接口定义
> type Starter interface {
>     Start()  // 无返回值
> }
> ```
>
> 错误处理方式：使用 `logx.Errorf` 记录错误，不要 `log.Fatal`（会终止进程）。
>
> ```go
> // Correct
> func (c *Client) Start() {
>     if err := c.Connect(); err != nil {
>         logx.Errorf("IEC104 client %s:%d start failed: %v", c.cfg.Host, c.cfg.Port, err)
>     }
> }
>
> // Wrong - log.Fatal 会终止进程
> func (c *Client) Start() {
>     if err := c.Connect(); err != nil {
>         log.Fatal(err)  // 生产环境不应直接退出
>     }
> }
> ```

---

## Related

- [go-zero-conventions.md](./go-zero-conventions.md) — gen.sh 流程、ServiceContext、Logic 入口
- [error-handling.md](./error-handling.md) — `tool.NewErrorByPbCodeWrap` 用法
- [IEC 104 消息对接文档](../../../docs/iec104-protocol.md) — **唯一权威协议文档**（v1.4.0），全量 typeId 对照表、控制命令 gRPC 接口
- [IEC 104 数采平台架构](../../../docs/iec104.md) — 服务组件、数据流

> **Warning**: `common/iec104/IEC-104-doc.md` 已废弃删除，不再维护。所有协议文档以 `docs/iec104-protocol.md` 为准。
