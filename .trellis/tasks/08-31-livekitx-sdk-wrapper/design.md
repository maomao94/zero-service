# `common/livekitx` 技术设计

## 1. 目标与边界

`common/livekitx` 是 zero-service 的 LiveKit 机制层，基于锁定的 `github.com/livekit/server-sdk-go/v2 v2.18.1`，为业务服务提供一个初始化入口、原生管理 API 访问、Token 构造、实时参与者连接、Data/RPC、Webhook 校验和统一 Hook 分发。

核心原则：

- 业务服务依赖 `livekitx`，`livekitx` 不依赖任何 `app/*`、业务 proto、数据库 model 或具体消息队列。
- 请求/响应优先直接使用 LiveKit SDK 的 Protocol 类型，不复制 DTO；README 用功能目录解决可发现性。
- 一个运行实例复用管理 client 和共享 HTTP client；请求始终接收调用方 context。
- Hook 是业务扩展边界：公共包传递事件和底层错误，业务 handler 决定落库、审核、状态同步和业务重试。
- Cloud Agents、Connector、AgentSimulation 排除在 MVP 外，未来作为独立扩展，不污染核心构造路径。

## 2. 包结构

```text
common/livekitx/
├── config.go          # Config、Option、默认值和校验
├── client.go          # Client、服务入口、Close
├── room.go            # Room/Participant 原生服务入口或薄代理
├── token.go           # Join/SIP token 便捷构造
├── realtime.go        # 实时 Room 连接与生命周期
├── data.go            # Data、RPC 发送和注册
├── hooks.go           # Hook 类型、注册、注销、分发
├── webhook.go         # 原始 body/Authorization 校验与事件接收
├── errors.go          # 可判断的本地错误，不覆盖 SDK 错误
├── testing.go         # 测试辅助，仅在测试需要时提供
├── README.md
└── *_test.go
```

如果 SDK 的公开服务 client 已足够稳定，`room.go`、`egress.go` 等只提供分组访问器和中文文档，不重复声明每一个 SDK 方法。对 SDK 没有统一 accessor 的能力，增加最薄的原生类型转发，保持参数和返回值可追踪。

## 3. Client 与配置

建议公开入口：

```go
type Config struct {
    URL        string
    APIKey     string
    APISecret  string
    HTTPClient *http.Client
}

type Client struct { /* 管理 client、配置、dispatcher、关闭状态 */ }

func New(opts ...Option) (*Client, error)
func (c *Client) API() *lksdk.LiveKitAPI
func (c *Client) Close() error
```

`New` 负责规范化 `http://`/`https://` 或 SDK 支持的 URL 形式、校验 URL 和成对凭据、创建共享 client 与 Hook dispatcher。请求超时完全由调用方 context 控制；本包不派生或覆盖 context。`Close` 幂等，停止由 `livekitx` 创建的实时连接和异步 dispatcher；不关闭调用方注入的 HTTP client。

管理服务访问器返回 SDK 原生 client：

```go
c.API().Room()
c.API().Egress()
c.API().Ingress()
c.API().SIP()
c.API().AgentDispatch()
c.API().Connector() // 非 MVP，可不暴露在首版文档入口
```

MVP 文档入口重点覆盖 Room、Participant、Egress、Ingress、SIP、AgentDispatch；未纳入的扩展能力必须列在 README 的边界章节。

## 4. Hook 模型

Hook 分为两类：

1. `Webhook`：服务端通过 HTTP 接收的完整事件，必须先用 signing key 校验原始 body，再解析事件。
2. `Realtime`：Go 参与者连接收到的 Room、Participant、Track、Data、RPC、Connection 回调。

采用带注销句柄的实例级注册：

```go
type Subscription interface { Unsubscribe() }
type Handler[T any] func(context.Context, T) error
func (c *Client) OnChatMessage(h Handler[ChatMessageEvent]) Subscription
```

事件 dispatcher 规则：

- 同一事件类型的 handler 按注册顺序快照执行，注册/注销不会阻塞已开始的 handler。
- 默认同步执行，保证业务可以明确处理错误；提供显式异步选项时必须有 bounded queue、关闭等待和丢弃/失败指标，不隐式启动无限 goroutine。
- handler 返回的第一个错误向调用方返回；其余 handler 仍按契约执行或由 `FailFast` 选项明确停止，首版选择继续执行并聚合错误。
- handler panic 被 recover，转换为带事件类型的错误并记录日志；不能导致实时读循环退出或进程崩溃。
- `context.Context` 由请求/连接生命周期派生；关闭时取消，handler 必须尊重取消。
- 无 handler 时安全忽略；未知 Webhook 事件保留 event ID/type 并安全记录。

事件结构保留 `RoomName`、参与者 identity/SID、Track SID/source、topic、payload、Webhook event ID/time 等可审计字段；不把业务 ID 或业务角色写进公共事件类型。

## 5. 实时连接、Data 与 RPC

`Connect` 使用 SDK 的实时连接入口和业务传入的参与者 Token。`Client` 负责连接选项、生命周期和统一回调桥接，业务服务负责 Token 授权和是否发布媒体。活动连接索引通过 `Store` 保存可序列化的节点、房间、身份、会话、状态和时间字段；SDK Room 指针只存在于本进程，不能写入 Redis/DB。

Data API 直接复用 SDK 支持的可靠/不可靠、目标身份和 topic 参数；聊天 Hook 只解析约定的聊天 payload，不把聊天历史存储内置到包中。RPC 注册与调用保留 SDK 原生 timeout、response/error 语义，并将回调错误传回 SDK。

媒体能力首版提供 SDK 原生 Room/Track/Publication 访问，不重新实现 RTP/编码器；需要文件媒体或特殊采集时由业务使用 SDK `media` 包，README 给出示例和原生依赖限制。

## 6. Webhook 接收

提供接收函数或 HTTP handler 适配器，输入必须能够读取完整原始 body 和 `Authorization`。校验失败返回明确的 4xx/错误，不调用业务 handler；成功后按 event ID/事件类型分发。去重存储接口保持可选，由业务服务注入自己的数据库/Redis 实现；公共包只提供幂等判断所需的 event ID，不建立默认永久内存去重。

## 7. 错误与可观测性

- 保留 `lksdk.ServerError`、Twirp code/message 和原始 error chain，使用 `%w` 包装本地上下文。
- 配置错误、关闭后调用、Hook panic、签名失败定义可 `errors.Is/As` 判断的本地错误。
- 日志禁止输出 API secret、Webhook signing key、Token 和完整 Authorization；请求日志仅输出服务、方法、状态和 request/event ID。
- 所有网络请求和 Hook 分发记录可选 trace/context 字段，但不在公共包强制绑定具体日志实现。

## 8. 测试分层

- 纯单元测试：配置、Token、错误、dispatcher 顺序/注销/并发/panic/关闭、Webhook 签名和重复事件。
- HTTP/Twirp mock 测试：所有管理服务入口的请求方法、路径、认证、请求体、错误响应和 context 取消。
- 本地集成测试：通过环境变量连接 `http://127.0.0.1:7880`，创建/查询/删除房间、生成 Token、连接实时参与者、发送 Data/RPC、验证参与者 Webhook；未配置时跳过并给出原因。
- `go test -race ./common/livekitx/...` 覆盖 dispatcher 和实时生命周期；最终运行全仓测试。

## 9. 兼容与演进

首版不维护 SDK 类型镜像。升级 LiveKit SDK 时必须重新核对公开入口、Protocol 类型、Hook 签名和 README；新增能力通过新 accessor/option，不改变已有 handler 的事件语义。扩展能力未来放入 `common/livekitx/cloudagents` 或独立包，不改变核心 `New` 的必要配置。
