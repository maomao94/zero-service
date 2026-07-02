# gnetx 技术设计（design.md）

> 任务：.trellis/tasks/07-02-gnetx ｜ 包：`common/gnetx`（扁平单包，package `gnetx`）
> 依据：research/netmc-design.md、research/gnet-design.md、prd.md D1-D5

## 1. 定位与边界

gnetx 是 gnet 之上的**开箱即用 TCP 框架**，不是 netmc 的 Go 移植。核心只做"分帧 → 解码 → 处理 → 回包 + 生命周期"；路由与请求-响应是 opt-in 层。纯推送/遥测协议只用 Core，零 tid/路由包袱。

| 层 | 谁用 | 内容 |
|---|---|---|
| Core | 所有协议 | Server/Client 启动、Codec（分帧+序列化）、Session、Handler、空闲/心跳、优雅 drain、慢处理告警、logx 日志 |
| opt-in | 需要才用 | Router（按 id 路由）、请求-响应（tid/antsx ReplyPool 响应式） |

**依赖**：`github.com/panjf2000/gnet/v2`（新增）、`common/antsx`（ReplyPool/Reactor/Promise）、`github.com/zeromicro/go-zero/core/logx`（日志，项目既有）、`github.com/zeromicro/go-zero/core/collection`（TimingWheel，经 antsx 间接）。Go 1.26。

**不暴露**：用户不直接实现 gnet `EventHandler`、不直接调 `c.Peek/Discard/Write`；gnetx 内部实现 `EventHandler` 适配上层 `Handler`。`Session` 不暴露 raw `gnet.Conn`。

## 2. 包文件划分（扁平，package gnetx）

| 文件 | 职责 |
|---|---|
| `options.go` | `ServerOptions`/`ClientOptions` + `With*` 函数式选项 + 校验 |
| `codec.go` | `Codec`/`Framer`/`Serializer` 接口、`ErrIncompletePacket`、`funcCodec` 适配器、公共编解码工具 |
| `codec_lengthprefix.go` | `LengthPrefixFramer`（uint16/uint32 BE/LE，offset/adjust） |
| `codec_delimiter.go` | `DelimiterFramer`（字节标记 + strip 选项，多分隔符） |
| `codec_fixed.go` | `FixedLengthFramer` |
| `serializer.go` | 内置 `RawSerializer`、`JSONSerializer`（快速原型）；二进制协议用户自实现 `Serializer` |
| `message.go` | opt-in 接口 `Identifiable`/`Correlatable`/`Response`/`ClientIdentifiable` |
| `session.go` | `Session` + `SessionManager` + `SessionListener`（默认空实现嵌入桩） |
| `handler.go` | `Handler` 接口、`HandlerFunc`、`AsyncHandler` 标记、慢处理告警 |
| `router.go` | `Router`（实现 `Handler`）+ `Handle`/`HandleTyped[T]` |
| `request.go` | 请求-响应：`ReplyPool` 集成、`Session.Request/Notify`、`resolveResponse` |
| `server.go` | `Server`（实现 gnet `EventHandler`）+ OnTraffic/OnOpen/OnClose/OnBoot + 优雅 Shutdown |
| `client.go` | `Client`（实现 gnet `EventHandler`）+ Dial + Shutdown |
| `idle.go` | 空闲扫描（独立 goroutine，非 OnTick）+ 心跳钩子 |
| `errors.go` | 哨兵错误 |
| `*_test.go` | 同名测试 |

## 3. 核心契约

### 3.1 Codec（分帧 + 序列化解耦）

把"字节流切帧"与"帧↔消息转换"解耦，最大化开箱即用：

```go
// Framer：纯字节流分帧。只用 gnet.Conn 的 Peek/Discard，返回一帧的原始字节。
// 半包返回 ErrIncompletePacket；不可恢复错误（magic 错等）返回非 nil error。
type Framer interface {
    NextFrame(c gnet.Conn) ([]byte, error)
}

// Serializer：原始帧字节 ↔ 消息转换（用户为自定义协议实现）。
type Serializer interface {
    Decode(raw []byte, sess *Session) (any, error)
    Encode(msg any, sess *Session) ([]byte, error)
}

// Codec：Framer + Serializer 的组合，框架内部用，也允许用户整体自定义。
type Codec interface {
    Decode(c gnet.Conn, sess *Session) (any, error) // ErrIncompletePacket = 半包
    Encode(msg any, sess *Session) ([]byte, error)
}
```

内置组合（`codec_*.go` 提供 Framer；`serializer.go` 提供常用 Serializer；`codec.go` 提供把 Framer+Serializer 拼成 Codec 的 `NewCodec(framer Framer, ser Serializer) Codec`）：
- `NewLengthPrefixCodec(lengthBytes int, endianness binary.ByteOrder, ser Serializer, opts ...LengthPrefixOption) Codec`
- `NewDelimiterCodec(delimiter []byte, strip bool, ser Serializer) Codec`
- `NewFixedLengthCodec(length int, ser Serializer) Codec`
- `NewFuncCodec(decode func(gnet.Conn, *Session) (any, error), encode func(any, *Session) ([]byte, error)) Codec` — 完全自定义分帧+序列化一体。

**半包/粘包**：`NextFrame` 用 `c.Peek(n)` → `io.ErrShortBuffer` 即半包，返回 `ErrIncompletePacket`；OnTraffic 收到后 `break`，剩余字节留 buffer 等下次可读事件；批量解码到上限且有剩余 → `c.Wake(nil)` 重触发（参照 gnet simple_protocol）。

**线程安全**：`Framer.NextFrame`/`Codec.Decode` 只在 on-loop 的 OnTraffic 调（仅用 Peek/Discard，on-loop 安全）。`Encode` 不读 conn，可 on-loop/off-loop 调（用于回包与 `Session.Send`）。

### 3.2 Session 与 SessionManager

```go
type Session struct {
    id         string          // 框架分配（server: 远端地址派生；client: 拨号派生）
    alias      string          // opt-in: 用户 Register 设的业务 id（设备号等）
    conn       gnet.Conn
    mgr        *SessionManager
    isClient   bool
    created    time.Time
    lastActive atomic.Int64   // unix nano
    attrs      sync.Map        // typed key-value（用户存业务状态）
    pool       *antsx.ReplyPool[any] // opt-in，懒初始化（首次 Request 或启用响应式时建）
    poolOnce   sync.Once
    closeOnce  sync.Once
    closed     atomic.Bool
}
```

方法：
- `ID() string`、`Alias() string`、`RemoteAddr() net.Addr`、`IsClient() bool`
- `SetAttribute(key, val any)` / `Attribute(key any) any` / `DeleteAttribute(key any)`
- `Register(alias string)` — opt-in，把自己加入 SessionManager 的 alias 索引（设备注册场景）
- `Send(msg any) error` — 编码 + `conn.AsyncWrite`（**off-loop 安全**，用户业务 goroutine 推送用）
- `Notify(ctx, msg any) error` — Send 的语义别名（fire-and-forget），命名对齐 antsx 语义
- `Request(ctx, msg Correlatable, ttl time.Duration) (any, error)` — opt-in 请求-响应（见 §3.4）
- `Close() error` — 幂等；触发 pool.Close + mgr 移除 + conn.Close
- framework 内部：`resolveResponse(tid string, resp any) bool`、`touch()`（更新 lastActive）

**SessionManager**：
```go
type SessionManager struct {
    mu    sync.RWMutex
    byID  map[string]*Session   // 框架 id
    alias map[string]*Session   // opt-in 业务 id
    listener SessionListener
}
func (m *SessionManager) Get(id string) *Session        // 先查 alias 再查 byID
func (m *SessionManager) All() []*Session               // 快照，用于广播
func (m *SessionManager) Count() int
func (m *SessionManager) Register(s *Session, alias string)
func (m *SessionManager) Remove(s *Session)
```

**SessionListener**（默认空实现嵌入桩，用户按需 override）：
```go
type SessionListener interface {
    OnCreated(s *Session)        // OnOpen 时
    OnRegistered(s *Session)     // Session.Register 时
    OnDestroyed(s *Session)      // OnClose 时
}
type noopSessionListener struct{}
func (noopSessionListener) OnCreated(*Session)    {}
func (noopSessionListener) OnRegistered(*Session) {}
func (noopSessionListener) OnDestroyed(*Session)  {}
```

### 3.3 Handler 与 Router

```go
// Handler：统一分发入口。Server/Client 的 OnTraffic 解码后总调 handler.Handle。
// 返回 (reply, err)：reply 非 nil 框架编码后回包；err 进 interceptor/日志。
type Handler interface {
    Handle(sess *Session, msg any) (any, error)
}
type HandlerFunc func(*Session, any) (any, error)  // 实现 Handler

// AsyncHandler：标记 handler 要 offload 到 antsx Reactor（D1 显式异步）。
// 框架遇 AsyncHandler 时不 on-loop 调，而是 reactor.Submit 后在回调里 Session.Send(reply)。
type AsyncHandler interface {
    Handle(sess *Session, msg any) (any, error)
    Async() bool
}
```

**Router**（opt-in，本身实现 Handler）：
```go
type Router struct {
    mu       sync.RWMutex
    handlers map[int]Handler
    fallback Handler
}
func NewRouter() *Router
func (r *Router) Handle(id int, h Handler)                          // 注册
func (r *Router) HandleFunc(id int, fn HandlerFunc)                 // 便捷
func (r *Router) HandleTyped[T any](id int, fn func(*Session, T) (any, error))  // 泛型类型安全
func (r *Router) Async(id int, h Handler)                           // 注册为 async
func (r *Router) Fallback(h Handler)                                // 未命中 id 时
func (r *Router) Handle(sess *Session, msg any) (any, error)        // 按 msg.(Identifiable).MessageID() 查
```
`HandleTyped[T]` 内部用 `func(s, m) (any,error){ return fn(s, m.(T)) }` 包裹；类型不匹配在注册时通过 factory 校验或在运行时 type-assert 失败走 fallback+告警。配合 `MessageRegistry`（`router.RegisterType(id, factory func() any)`）让 decoder 能按 wire id 实例化具体类型，避免用户手写反射扫描（修复 netmc 痛点）。

**慢处理告警**：on-loop 同步 handler 超过阈值（默认 50ms，可配）打 logx 慢处理日志；async handler 不计入（已 offload）。

### 3.4 请求-响应（opt-in，tid 响应式）

opt-in 接口（`message.go`）：
```go
type Correlatable interface { TID() string }       // 请求侧：提供关联 id
type Response    interface { ResponseTID() string } // 回包侧：对应请求的 tid
type Identifiable interface { MessageID() int }     // Router 路由用
type ClientIdentifiable interface { ClientID() string } // opt-in 设备身份（Session.Register 用，非必需）
```

**Session.Request**（`request.go`）：
```go
func (s *Session) Request(ctx context.Context, msg Correlatable, ttl time.Duration) (any, error) {
    s.initPool()
    tid := msg.TID()
    return antsx.RequestReply(ctx, s.pool, tid, func() error { return s.Send(msg) }, ttl)
}
func (s *Session) initPool() {
    s.poolOnce.Do(func() { s.pool = antsx.NewReplyPool[any](antsx.WithName("gnetx-"+s.id), antsx.WithDefaultTTL(30*time.Second)) })
}
```
- 统一 tid 多路复用（无单挂 footgun）。每 Session 一个 ReplyPool，key=tid，生命周期绑 Session。
- `Send(msg)` = 编码 + `conn.AsyncWrite`（off-loop 安全，Request 多从业务 goroutine 调）。
- **禁止在 on-loop handler 里调 Request**（会阻塞 event-loop）；需要则用 AsyncHandler offload 后再 Request。

**入站回包自动路由**（server/client OnTraffic，`server.go`/`client.go`）：
```go
if r, ok := msg.(Response); ok {
    if s.resolveResponse(r.ResponseTID(), msg) { continue } // 命中在途请求，完成，跳过 handler
    // 无在途匹配 → 落到 handler 当意外报文处理
}
reply, err := handler.Handle(s, msg)
```
`resolveResponse` = `s.pool.Resolve(tid, resp)`（pool 未初始化返回 false）。

**断连清理**：`Session.Close` → `pool.Close()` 自动 Reject 所有在途请求（ErrReplyClosed）。 antsx ReplyPool 幂等 Close。

### 3.5 Server（`server.go`，实现 gnet.EventHandler）

```go
type Server struct {
    gnet.BuiltinEventEngine
    opts    ServerOptions
    codec   Codec
    handler Handler
    mgr     *SessionManager
    eng     gnet.Engine
    asyncWG sync.WaitGroup   // 在途 async handler
    idleStop chan struct{}   // 停空闲扫描
}
func NewServer(opts ServerOptions) (*Server, error)   // 校验：Addr/Codec/Handler/MaxFrameLength 必填
func (s *Server) Run() error                          // 阻塞，gnet.Run
func (s *Server) Shutdown(ctx context.Context) error  // 优雅
func (s *Server) Manager() *SessionManager
```

EventHandler 实现：
- `OnBoot(eng)` → 存 eng，启动空闲扫描 goroutine（若 IdleTimeout>0）。
- `OnOpen(c)` → 建 Session（id=remoteAddr 派生），`c.SetContext(sess)`，mgr 加入，listener.OnCreated。返回 nil, None。
- `OnTraffic(c)`：
  ```
  sess := c.Context().(*Session); sess.touch()
  for i:=0; i<batchLimit; i++ {
      msg, err := codec.Decode(c, sess)
      if errors.Is(err, ErrIncompletePacket) { break }
      if err != nil { /* 不可恢复：可配关闭策略，默认 Close */ return gnet.Close }
      if resp, ok := msg.(Response); ok && sess.resolveResponse(resp.ResponseTID(), msg) { continue }
      dispatch(sess, msg)  // sync on-loop 或 async offload
  }
  if 命中 batchLimit 且 c.InboundBuffered()>0 { c.Wake(nil) }
  return gnet.None
  ```
  dispatch(sync)：on-loop 调 `handler.Handle`，reply 非 nil → `codec.Encode` → `c.Write`（on-loop 安全）。慢处理计时告警。
  dispatch(async)：`asyncWG.Add(1)` + `reactor.Submit` 跑 handler，回调里 `sess.Send(reply)` + `asyncWG.Done()`。
- `OnClose(c, err)` → listener.OnDestroyed + mgr.Remove + sess.Close（pool.Close）。
- `OnShutdown(eng)` → 关 idleStop。

**优雅 Shutdown**：`Shutdown(ctx)` = `eng.Stop(ctx)`（停止接受+关连接）+ `asyncWG.Wait`（等在途 async 完成，受 ctx 约束；超时则记日志返回）。MVP 不发"going away"帧（可配后续加）。

### 3.6 Client（`client.go`）

```go
type Client struct {
    gnet.BuiltinEventEngine
    opts   ClientOptions
    codec  Codec
    handler Handler
    mgr    *SessionManager
    gcli   *gnet.Client
}
func NewClient(opts ClientOptions) (*Client, error)
func (c *Client) Start() error
func (c *Client) Dial(network, address string) (*Session, error)      // 阻塞到连上，返回 Session
func (c *Client) DialContext(network, address string, ctx any) (*Session, error)
func (c *Client) Shutdown(ctx context.Context) error
func (c *Client) Manager() *SessionManager
```
- `Dial` 内部 `gcli.Dial` → 得 `gnet.Conn` → 建 Session（id=本地派生，isClient=true）→ mgr 加入 → OnOpen/OnTraffic/OnClose 同 server 逻辑（解码+Response 路由+handler）。
- Client 的 Session 同样支持 `Request`（主动发请求等回包）与 `Send`（推送）。双向对称。
- 重连/连接池 = defer（D4）。

### 3.7 空闲/心跳（`idle.go`）

**独立扫描 goroutine**（非 gnet OnTick，规避 N× 问题）：
```
tick := IdleTimeout/2（下限 1s）
for {
    select { case <-idleStop: return; case <-ticker.C: }
    for _, s := range mgr.All() {
        if time.Since(s.lastActive) > IdleTimeout { s.Close() }  // Close 跨 loop 安全
    }
}
```
- 读写/全空闲：MVP 只做读空闲（静默超时关连接），对齐 netmc readerIdleTime。写/全空闲后续加。
- 应用层 ping/pong：opt-in 钩子 `OnIdle(sess) error`（返回 error 触发 Close，可发心跳帧）；MVP 提供钩子位但不内置协议。

### 3.8 错误与日志

- 哨兵错误（`errors.go`）：`ErrIncompletePacket`、`ErrSessionClosed`、`ErrFrameTooLarge`、`ErrNoHandler`、`ErrPendingNotFound`。用 `errors.Is` 判。
- 不可恢复 decode 错误策略：`ServerOptions.OnDecodeError func(sess, err) Action`，默认 `Close`，可配 `LogOnly`。
- 日志：`logx` 结构化；连接建立/断开、慢处理、decode 错误、请求超时均记。 Arabsx ReplyPool 自带每分钟 stat。

## 4. 数据流

### 入站（server/client 共用）
```
gnet readable → OnTraffic(on-loop)
  → Codec.Decode(c, sess): Framer.NextFrame(Peek/Discard) → Serializer.Decode → msg any
  → sess.touch()
  → msg.(Response)? → resolveResponse(tid) 命中则 continue
  → handler.Handle(sess, msg):
       sync  → on-loop 跑 → reply → Codec.Encode → c.Write
       async → reactor.Submit → 回调 sess.Send(reply)=Encode+AsyncWrite
```

### 出站（业务主动）
```
业务 goroutine → sess.Send(msg): Codec.Encode → conn.AsyncWrite  (fire-and-forget)
业务 goroutine → sess.Request(ctx, msg, ttl): ReplyPool.Register(tid) → Send(msg) → Await(ctx)
                                                              ↑ 回包经入站 resolveResponse 完成
```

## 5. 线程契约（来自 gnet 研究关键约束）

- OnTraffic 在 event-loop goroutine，**不得阻塞**。sync handler 必须快（内存计算+立即回包）；重活用 AsyncHandler offload。
- on-loop only：`Codec.Decode`（Peek/Discard）、`c.Write`/`c.Writev`（sync 回包）、`c.Context/SetContext`、`c.RemoteAddr`。
- off-loop 安全：`conn.AsyncWrite`（Session.Send 用）、`conn.Close`、`SetDeadline`、`antsx.ReplyPool` 全部方法、`SessionManager` 全部方法。
- `Session.lastActive` 用 atomic，attrs 用 sync.Map → 跨 goroutine 安全。
- ReplyPool.pending 跨 goroutine（业务 Register + on-loop Resolve），antsx 内部已 mutex 保护。
- 禁止在 on-loop handler 调 `Session.Request`（阻塞 loop）；需 offload 到 AsyncHandler 再 Request。

## 6. 兼容性与回滚

- 新增独立包 `common/gnetx`，不触碰 netx/iec104/antsx 现有代码（仅在 go.mod 加 gnet 依赖、import antsx）。
- go.mod 加 `github.com/panjf2000/gnet/v2`（v2.9.8，Go 1.20+，1.26 安全）。
- 回滚点：每个文件独立提交；实现按 codec → session → router → request → server → client → idle 顺序，每步可独立编译验证。出问题可回退到上一个可编译态。

## 7. 取舍记录

- **Framer/Serializer 解耦** vs 单一 Codec：解耦让标准分帧开箱即用、自定义序列化按需；代价是多一层接口。选解耦。
- **独立 idle goroutine** vs gnet OnTick：OnTick 多核 N× 触发且无 per-loop 连接枚举 API；独立 goroutine 简单正确。选独立。
- **每 Session 一个 ReplyPool** vs 全局池：生命周期干净、断连自动清理、隔离好；代价是 N 个 TimingWheel（可接受，后续可优化为共享 TW）。选每 Session。
- **Handler 单入口 + Router 是 Handler 实现** vs Server 特判 Router：单一 seam，用户可直接传 Router 当 Handler，无特殊分支。选单入口。
- **Reply any 返回 + 框架回包** vs 用户自己 Send：统一签名，sync 路径 on-loop c.Write 高效，async 路径框架代发。选统一。
- **不做 TLS/连接池/metrics**（D4 defer）：MVP 聚焦核心 + opt-in 响应式，避免过大。

## 8. MVP 不做（defer 清单）

TLS、client 连接池/重连/健康检查/扇出、metrics/分布式追踪、UDP、AsyncBatch 批处理、offline 离线缓存/重投递、写/全空闲检测、going-away 帧、对 netx/iec104 迁移。
