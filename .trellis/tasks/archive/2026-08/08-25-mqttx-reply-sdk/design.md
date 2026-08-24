# Design: mqttx 广播层 SDK（broadcast 子包）

关联 PRD：`.trellis/tasks/08-25-mqttx-reply-sdk/prd.md`

## 1. 架构与边界

```
zero-service/common/mqttx/            (协议中立底座，不动)
└── broadcast/                        (新增：广播协议泛化层)
    ├── topic.go                       djisdk 风格主题定义 + Pattern + 分组 + doc
    ├── body.go                        标准 BroadcastBody / BroadcastAckBody
    ├── ack.go                         通用 ack 解码器 + NewAckReplyRouter
    ├── errors.go                      errorKind 常量 + 错误↔kind 映射等价器
    ├── dispatcher.go                  消费分发骨架：method→executor 注册、防回环、ack 回发
    ├── client.go                      Broadcaster：fire-and-forget / 等待 ack 两个发送模式 + consume 入口
    └── *_test.go                      主题一致性测试 + 分发/errorKind 回环测试

app/oryxserver/    删除 relay.BroadcastTopic 等常量、RelayBody/RelayAckBody、DecodeRelayAck；改注册 executor
app/ieccaller/     删除内联主题串、types.BroadcastBody/AckBody、decodeBroadcastAck、publishAckReply 模板；改注册 executor
```

- 决策：放在 `common/mqttx/broadcast`（mqttx 的子包，import `zero-service/common/mqttx/broadcast`），满足"作为 mqttx 基础能力"；`common/mqttx` 根包保持协议中立不膨胀。
- 决策：业务 method 分发留在各 app：app 注册 `method → executor`，骨架负责路由与 ack 回发。

## 2. 主题契约（djisdk 风格）

**该功能未上线**，无线兼容包袱，允许一次线格式优化：ack 通道对齐 djisdk `_reply` 命名（`broadcast-ack/` → `broadcast_reply/`）；广播主题值不变。

| 语义 | 值（线格式） | 新 API |
|---|---|---|
| 广播主题 | `{prefix}/broadcast`，如 `oryx/relay/broadcast`、`iec/broadcast` | `BroadcastTopic(prefix string) string` |
| 广播订阅模式 | 同值（具体订阅，无通配） | `BroadcastTopicPattern(prefix string) string`（为一致性测试提供锚点） |
| ack 主题 | `{prefix}/broadcast_reply/{id}` | `BroadcastAckTopic(prefix, instanceID string) string` |
| ack 订阅模式 | `{prefix}/broadcast_reply/+` | `BroadcastAckTopicPattern(prefix string) string`（新增通配订阅能力：全景订阅/监控） |

- 每个函数带三要素注释（路径格式/方向/用途），`====` 分组，包级 doc.go。
- `topic_test.go`：`TestTopicPatternsMatchConcreteTopics` 断言 `Pattern` 与具体主题一一匹配（仿 djisdk）。
- 前缀：由各 app 传入（**oryxserver 用 `oryx/server`**——容器级命名，不带 relay 业务属性；ieccaller 用 `iec`），SDK 不硬编码业务前缀；提供 `Prefix(const ...string) string`（用 "/" 连接）小工具 + `WithPrefix` option，见 client 部分。

## 3. Body 契约（通用层只含协议字段，业务数据走 opaque payload）

SDK 是**协议中立层**：通用 body 只含关联/路由字段；任何具体业务数据（如 oryxserver 的 taskId、iec104 的 protojson）一律经 `Body`/`ResponseBody` 承载，字段格式由业务 executor 自行约定，SDK 不感知。

```go
type BroadcastBody struct {
    Tid      string `json:"tId,omitempty"`      // 请求方自动生成的关联 ID（调用方不暴露）
    AckTopic string `json:"ackTopic"`           // 请求方自己的 ack 主题（SDK 内部填充）
    Method   string `json:"method"`             // 业务方法名（SDK 接收 method 参数）
    Body     string `json:"body,omitempty"`     // opaque 业务 payload（string 承载字节）
}

type BroadcastAckBody struct {
    Tid          string `json:"tId"`
    Method       string `json:"method"`
    Success      bool   `json:"success"`
    ResponseBody string `json:"responseBody,omitempty"` // opaque 业务结果 payload
    Error        string `json:"error,omitempty"`
    ErrorKind    string `json:"errorKind,omitempty"`    // 见 §5
}
```

- 发送 API 只暴露 `(method, payload []byte)`，Tid/AckTopic/Method 由 SDK 内部填充——**调用方与主题数据零接触**，避免非业务数据污染发布主题。
- oryxserver 的 `RelayBody`/`RelayAckBody` 与 ieccaller 的 `BroadcastBody`/`BroadcastAckBody` 在各自代码中删除，业务 payload（taskId/protojson）改走 `Body`/`ResponseBody`（迁移前先 grep 确认无其他引用）。

## 4. 客户端与数据流

### Broadcaster（`broadcast.Client` 包装）

```go
type Executor func(ctx context.Context, method string, payload []byte) ([]byte, error)

type Broadcaster interface {
    // Fire-and-forget：只发 method + 业务 payload，不等待 ack（对齐 oryxserver PublishRelayStop）
    Broadcast(ctx context.Context, method string, payload []byte) error
    // 等待 ack：成功返回业务结果字节；ack.Success==false 按 errorKind 还原为领域错误（不对齐 ieccaller PushPbBroadcastWithAck）
    BroadcastReply(ctx context.Context, method string, payload []byte, timeout time.Duration) ([]byte, error)
    // 挂到 mqttx.Client 上消费广播（防回环 + 分发）
    AddBroadcastHandler() error
    // 注册 method 业务执行器
    AddExecutor(method string, fn Executor)
}
```

构造：`NewBroadcaster(c mqttx.Client, globalInstanceID string, opts...)`；实现细节：
- **协议字段与调用方隔离**：SDK 在发送时构造完整 `BroadcastBody`（Tid 自动生成、AckTopic=本实例 ack 主题、Method=参数）与完整 `BroadcastAckBody`（Success/Error/ErrorKind/ResponseBody 由骨架填充）；调用方只见 `(method, payload) ↔ (payload, error)`。
- ack 解码器：一个通用 `DecodeAck`（原 `DecodeRelayAck`/`decodeBroadcastAck` 的并集实现），`AckReplyRouter[*BroadcastAckBody]` 由 SDK 导出 `NewAckReplyRouter(ttl, name)` 构造（TTL 默认 10s，可配），app 在创建 `mqttx.Client` 时经 `mqttx.WithReplyRouter(BroadcastAckTopic(prefix, instanceID), router)` 绑定（mqttx.Client 无事后注册 reply handler 的公开入口）。
- instanceId：仍由 app 生成（`"oryx-relay-"+uid` / `"iec-caller-"+uid`），作为 `globalInstanceID` 传入；防回环判定：`body.AckTopic == BroadcastAckTopic(prefix, globalInstanceID)` → 忽略（两 app 现状一致）。
- 发送：`BroadcastReply` 内走 `mqttx.RequestReply[*BroadcastAckBody]`（等价 ieccaller 现状）；`Broadcast` 走 `PublishWithTrace`（等价 oryxserver 现状）。
- prefix：经 `WithPrefix` 绑定（含 `Prefix(parts ...string) string` 拼接小工具）；**oryxserver 用 `"oryx/server"`（容器级命名，不带 relay 业务属性，未来其他能力沿用）、ieccaller 用 `"iec"`**。

### 消费分发骨架（dispatcher.go）

- 骨架消费流程：非 broadcast 模式（app 已自行判断）→ `jsonx.Unmarshal` → ack 回环忽略 → 查 executor map → 未注册 method：回 ack `Success=false, Error="unknown method", ErrorKind=KindUnknown` → 执行 → 成功回 ack（`ResponseBody`=executor 返回字节）→ 失败：`NormalizeErrorKind(err)` 归一后回 ack。
- **骨架不做业务反序列化**：`Executor` 收到的是 `(method, payload []byte)`，业务 executor 自行定义 payload 格式（oryxserver：taskId JSON；ieccaller：protojson）。
- `ErrSkipAck` 哨兵：executor 返回它表示成功但不回 ack（oryxserver found==false、ieccaller ClearPointMappingCache 语义保留）。

## 5. errorKind 契约（把 ieccaller 规则泛化）

错误 ↔ kind 双向映射，**SDK 只内置传输泛化类别**，业务类别由 app 注册：

| kind | 输出侧（错误→kind） | 输入侧（kind→错误） | 归属 |
|---|---|---|---|
| `timeout` | `antsx.ErrReplyExpired` | → `antsx.ErrReplyExpired` | SDK 内置 |
| `duplicate` | `antsx.ErrDuplicateID` | → `antsx.ErrDuplicateID` | SDK 内置 |
| `unknown` | 兜底 | → `errors.New(msg)` | SDK 内置 |
| `iec_rejected` | ieccaller 注册 | ieccaller 注册（`CommandRejectedError` 还原） | **业务** |

- `KindIECRejected` 不在 SDK 中定义——ieccaller 在 app 内声明 `const errKindIECRejected = "iec_rejected"` 并 `RegisterErrorKind(client.CommandRejectedError{}, errKindIECRejected, ...)` 注册双向映射（wire 值不变，语义与迁移前完全一致）。
- API：`RegisterErrorKind(src error, kind string, dst func(msg string) error)`；骨架消费时输出侧自动归一（`NormalizeErrorKind` 先查内置再查注册表；未命中 `unknown`）；`BroadcastReply` 输入侧按 kind 还原（`ErrorFromKind`）。
- oryxserver 收益：其消费端错误现在也被归一为 kind 回发（对调用方无既有依赖，仅增量）。

## 6. 兼容性与迁移

1. **无 wire 兼容约束**：广播集群功能未在任何生产环境部署（ieccaller 生产未启用集群模式；oryxserver 从未上线）。旧主题值（`oryx/relay/broadcast-ack/{id}`、`iec/broadcast-ack/{id}`）与旧 body 定义直接废除，不保留任何旧字符串。body 字段采用全 optional 的并集（`taskId`/`body`/`responseBody`/`errorKind`），结构自洽无需兼容历史数据。
2. app 迁移步骤（每 app 独立提交，先 ieccaller 后 oryxserver 或反之均安全）：
   a. 新增 `common/mqttx/broadcast` 包 + 测试（行为纯新增，无引用方变化）。
   b. ieccaller servicecontext/mqtt/broadcast.go 换用 SDK；grep 确认 `types.BroadcastBody` 无其他引用后删除（`common/iec104/types`）。
   c. oryxserver relay/servicecontext/mqtt/broadcast.go 换用 SDK；删除 `relay.BroadcastTopic` 常量族与 DecodeRelayAck。
3. 回滚：删除 broadcast 包 + 各 app 恢复原实现（git 层面单 commit revert）；wire 无变化使回滚窗口集群安全（不卸载则无风险）。

## 7. 明确不做

- 业务 method 分发逻辑不进 SDK（iec104 14 case、relay stop 判定留在 app executor）。
- `mock-service` 旧拷贝不同步（独立仓库，非本任务范围）。
- mqttx 根包 API 不变（`common/mqttx` 现有 client/reply_router 保持原样）。
- `errKind` 兜底语义、`iec_rejected` 注册后 ieccaller 恢复行为完全等同（不改变其现有错误类型）。

## 8. 风险与权衡

- 分发骨架引入注册式编排：ircon3 个 executor（14 case 的 protobuf 解包仍留 ieccaller），抽象不变式：**骨架不做业务反序列化**，`Executor` 收到的是已解码的 `BroadcastBody`，protojson 解析在业务 executor 内完成。
- `ErrSkipAck` 哨兵：需与现有"不回 ack"语义严格对标（oryxserver found==false；ieccaller ClearPointMappingCache 无 ack case）。
- 交付顺序：主题定义先行（独立可测、影响面小），再 body/解码器，最后分发骨架与两 app 迁移——每个 checkpoint 可构建可回滚。
