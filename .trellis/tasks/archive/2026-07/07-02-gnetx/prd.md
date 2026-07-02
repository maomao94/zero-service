# gnetx: 基于 gnet 的开箱即用 TCP 工具

## Goal

在 `common/gnetx` 输出一个基于 [gnet](https://github.com/panjf2000/gnet)（Go reactor TCP 框架）的开箱即用 TCP 工具，定位类比 Java/Netty 生态的 [netmc](https://github.com/yezhihao/netmc)：让开发者快速搭建自定义二进制 TCP 协议的服务端与客户端，无需直接接触 gnet 的 `EventHandler`/`Peek`/`Discard` 等底层原语。

目标是未来开发自定义 TCP 服务端/客户端时"快速开发、快速使用、最佳实践开箱即用"，并原生支持 TCP 双向报文 + 基于 `common/antsx` ReplyPool 的 tid 响应式编程（请求-响应按关联 ID 匹配）。

## 背景与参考（confirmed facts）

### netmc（Java/Netty MVC 框架，研究见 research/netmc-design.md）
- 3 层架构：transport（Netty pipeline）/ dispatch（@Endpoint+@Mapping 前置控制器）/ session+codec（用户扩展点）。
- 核心抽象：`Message{getClientId, getMessageId, getSerialNo}`、`Response{getResponseSerialNo}` 标记、`@Endpoint/@Mapping(types=int[])` 按 messageId 路由、`Session`（每连接上下文 + attrs + serialNo + register/invalidate + Mono 的 notify/request/response）、`SessionManager`、`HandlerInterceptor`（5 方法 Around）、`SessionListener`（created/registered/destroyed 默认空实现）、`MessageEncoder/Decoder`（Session 感知签名）。
- 请求-响应：`Session.request(msg, respClass) -> Mono<T>`，按 `respClass + serialNo` 关联；非 Response 类是"每类单挂"（footgun）。
- **netmc 无 first-class client**，出站靠已连接设备的 Session。
- 需要避免的点：反射派发 + InvocationTargetException、依赖 protostar 注解做 id→class 映射（应由框架提供 MessageRegistry）、Reactor Mono 依赖（Go 应用 channel + antsx ReplyPool）、非 Response 类单挂 footgun、消费者把 maxFrameLength 设成 MAX_VALUE。

### gnet（Go reactor，研究见 research/gnet-design.md）
- `EventHandler{OnBoot, OnShutdown, OnOpen, OnClose, OnTraffic, OnTick}`，`BuiltinEventEngine` 空嵌入桩。
- `Conn` = `Reader`(Peek/Discard/Next/InboundBuffered，仅 on-loop) + `Writer`(Write/Writev 同步仅 on-loop；AsyncWrite/AsyncWritev 跨 goroutine 安全) + `Socket`(全安全) + Context(仅 on-loop)。
- **gnet 不提供**：codec/分帧、session、请求-响应、空闲/心跳、优雅 drain、连接枚举/广播。全靠用户用 Peek/Discard 拼（simple_protocol 示例为模板）。
- 线程契约严格：每连接绑定一个 event-loop goroutine；OnTraffic 同步阻塞会拖死同 loop 所有连接；OnTick 每 loop 一次（多核下 N×）。
- 有真 client：`NewClient/Dial/DialContext/Enroll/EnrollContext`，v2.9.0+ client 多 loop。
- TLS 未实现（roadmap）。Go 1.26 安全（go.mod 要求 go 1.20）。

### antsx ReplyPool（common/antsx/replypool.go）
- `ReplyPool[T]`：关联 ID 请求-响应匹配，`Register(id, ttl) -> *Promise`，`Resolve(id, val)`，`RequestReply(ctx, pool, id, sendFn, ttl)` 一步到位。go-zero TimingWheel 管 TTL。并发安全。
- 这是 netmc Reactor Mono 的 Go 原生等价物，作为 gnetx 请求-响应关联引擎。

### 现有相关包边界
- gnetx 独立，不与 `common/netx`（HTTP 风格 client）、`common/iec104`（IEC104 协议）联动。
- `common/antsx` 作为响应式编程底座（ReplyPool/Promise/Reactor）。
- 命名遵循 `common/*x` 约定。

## Requirements

### R1 TCP Server 开箱即用
- 配置化启动：端口、分帧、codec、handler 注册、interceptor、session listener、空闲时间、maxFrameLength（必填）、线程数。
- 用户不直接实现 gnet `EventHandler`；gnetx 内部实现并适配上层 handler/middleware 链。
- 生命周期：启动/停止/优雅 drain（停止接受 → 处理完在途 → 关闭）。

### R2 TCP Client 开箱即用
- 主动连接远端：`Dial`/`DialContext`，产出与 server 端一致的 `Session`，具备 `notify/request/response` 出站 API。
- 填补 netmc 无 client 的缺口。

### R3 编解码框架（Codec）
- 内置分帧：LengthPrefix（uint16/uint32，BE/LE）、Delimiter（字节标记 + strip 选项）、FixedLength。
- 可插拔 `Codec{Encode, Decode}` 接口（Decode 基于 gnet Peek/Discard，处理半包 io.ErrShortBuffer）。
- Encoder/Decoder 签名 Session 感知（参考 netmc）。

### R4 协议/报文抽象（opt-in，非强制）
- Core 不强制 Message 接口；Codec.Decode 返回 `any`（用户 typed struct）。
- 可选 `Router`：`router.Handle(id, fn)`；消息 opt-in 实现 `Identifiable{ MessageID() int }` 即按 id 路由，否则用户在单 Handler 里自行 type-switch。
- 可选 MessageRegistry：`Register(id, factory func() any)` 供 decoder 按 wire id 实例化（修复 netmc 需手写反射扫描的痛点），仅 Router/类型化 handler 用户需要。

### R5 连接级会话与状态（Session）
- 每连接 `Session`：attrs（typed key）、lastActiveAt、register/invalidate、serialNo 计数器。
- `SessionManager`：按 sessionId/clientId 查找、created/registered/destroyed 事件。
- 不向用户 handler 暴露 raw gnet.Conn。

### R6 心跳与空闲检测
- lastActiveAt 在 OnTraffic 更新；OnTick 每 loop 扫描本 loop 连接，超时 Close。
- 可配置读写/全空闲超时；支持应用层 ping/pong 钩子。

### R7 双向报文 + antsx tid 响应式编程（opt-in 层，非核心）
- 纯推送/遥测协议无需启用。
- server 端与 client 端 Session 均支持 `Notify(msg)` 与 `Request(ctx, msg, ttl) (resp, error)`（消息实现 Correlatable/Response 才启用）。
- 关联引擎 = `common/antsx.ReplyPool`，每 Session 一个，key = tid，统一多路复用（见 D2）。
- 入站回包（实现 Response）经 framework 自动路由到 `ReplyPool.Resolve(tid, msg)`。

### R8 最佳实践开箱即用
- maxFrameLength 必填且强制合理上限。
- 慢处理告警（>阈值打日志）。
- 不可恢复 decode/encode 异常的可配置关闭策略。
- 结构化日志（logx）。
- 资源释放契约清晰（Session/ReplyPool/Server/Client 的 Close）。

## Acceptance Criteria

- [ ] AC1 一个最小自定义协议（LengthPrefix 分帧 + 1 个 Message + 1 个 handler）能在 < 50 行业务代码内跑起 server，用 client 拨号并发送请求拿到响应。
- [ ] AC2 server 端 `Session.Request(ctx, msg, ttl)` 发出请求后，client 回包按 serialNo 正确匹配并返回；超时返回明确错误。
- [ ] AC3 client 端 `Session.Request(ctx, msg, ttl)` 同样工作（双向对称）。
- [ ] AC4 LengthPrefix/Delimiter/FixedLength 三种内置分帧各自有测试覆盖半包/粘包。
- [ ] AC5 MessageRegistry 能按 int id 实例化正确 Message 类型，无需用户手写反射扫描。
- [ ] AC6 空闲超时：连接静默超过配置时长后被 Close，触发 SessionListener.sessionDestroyed。
- [ ] AC7 优雅停止：停止接受新连接 → 在途请求完成或超时 → 关闭所有连接 → 返回。
- [ ] AC8 handler 可选择同步（on-loop，必须快）或异步（offload 到 antsx Reactor，AsyncWrite 回包）。
- [ ] AC9 HandlerInterceptor 的 Before/After/Success/Exception/NotSupported 5 钩子可注入并生效。
- [ ] AC10 单测覆盖：codec 分帧、MessageRegistry、Session 生命周期、ReplyPool 关联、空闲检测。
- [ ] AC11 `go vet` / `go build` / 单测在 Go 1.26 通过。

## Out of Scope（MVP 之外，后续迭代）

- TLS（gnet 原生未实现，等 gnet 或后续用 Enroll(tls.Conn) 方案）。
- 连接池/重连/健康检查/扇出（client 连接治理）。
- 指标/metrics/分布式追踪。
- UDP 支持（先聚焦 TCP）。
- `@AsyncBatch` 批处理高吞吐模式。
- offline 离线缓存/重投递。
- 对 common/netx、common/iec104 的迁移/重构。

## Decisions（已决）

- D1 Handler 注册与执行模型（按 gnet 标准 + Go 最佳实践）：
  - 注册：类型化注册函数 `gnetx.Handle[T any](id int, fn func(s *Session, m *T) (resp Message, err error))` + `HandleAny(func(*Session, Message) error)` 兜底。无反射注解、无包扫描、无 DI，编译期签名安全。
  - 执行：默认 on-loop 同步执行，handler 必须快，回包走 `c.Write`；显式 `HandleAsync` 才 offload 到 antsx Reactor 协程池，回包走 `c.AsyncWrite`。映射 netmc 的 sync/@Async 二分，学习成本低、fast-path 零调度开销。
- D2 请求-响应关联（opt-in 层，非核心）：
  - 统一只用 **tid** 多路复用，去掉 netmc 非 Response 单挂 footgun。
  - 关联引擎 = `common/antsx.ReplyPool`；**每 Session 一个 ReplyPool**，key = tid，生命周期绑 Session，断连 `pool.Close()` 自动 Reject 在途。
  - 命名与 antsx 一致：关联 id 统一称 `tid`（string）。opt-in 接口 `Correlatable{ TID() string }`（请求侧）/ `Response{ ResponseTID() string }`（回包侧），消息实现才启用，纯推送协议零 tid 包袱。
  - framework 自动识别入站回包（实现 Response）→ `ReplyPool.Resolve(tid, msg)`，不进业务 handler；提供 `Session.Request(ctx, msg, ttl) (resp, error)`。
- D3 核心抽象方向（不抄 netmc MVC，做 Go 标准 TCP 框架）：
  - **Core 层（所有协议）**：`Codec{ Encode(msg any, sess *Session) ([]byte, error); Decode(c gnet.Conn, sess *Session) (any, error) }`，codec 返回 `any`（用户 typed struct），**不强制 Message 接口**；内置 LengthPrefix/Delimiter/FixedLength 分帧 + 可插拔 Codec。`Handler func(*Session, any) error` 单 handler 自行 type-switch 分发。`Session` 每连接上下文（attrs/lastActive/Send/Close），不暴露 raw gnet.Conn。Server/Client 启动 + 空闲/心跳 + 优雅 drain。
  - **opt-in 层（需要才用）**：`Router`（`router.Handle(id, fn)` + 消息实现 `Identifiable{ MessageID() int }` 按 id 路由）；请求-响应 tid 响应式（见 D2，消息实现 Correlatable/Response 才启用）。
  - 纯推送/遥测协议 = Codec + Handler，零 tid/路由包袱；请求-响应协议 = 多实现 opt-in 接口，拿到 antsx ReplyPool 响应式能力。
- D4 MVP 边界（按推荐切分）：
  - MVP 做：Core（Server/Client 启动 + Codec 三种分帧 + 可插拔 Codec + Session + Handler + 空闲/心跳 + 优雅 drain + 慢处理告警 + 结构化日志 logx）+ opt-in Router + opt-in 请求-响应（tid/antsx ReplyPool）。
  - Defer：TLS、client 连接池/重连/健康检查/扇出、metrics/分布式追踪、UDP、AsyncBatch 批处理、offline 离线缓存/重投递、对 netx/iec104 的迁移。
- D5 包结构（遵循 common/*x 项目惯例，扁平单包）：
  - `common/gnetx/` 单 package `gnetx`，按关注点分文件（`server.go`/`client.go`/`session.go`/`codec.go`/`codec_lengthprefix.go`/`codec_delimiter.go`/`codec_fixed.go`/`router.go`/`request.go`/`message.go`/`idle.go`/`options.go`/`errors.go` 等）+ 同名 `_test.go`。无子目录（与 mqttx/socketiox/gormx/netx/wsx 一致）。

## Open Questions（阻塞规划的待决项）

1. ~~Handler 注册与执行模型~~ → 见 D1。
2. ~~请求-响应关联 key~~ → 见 D2。
3. ~~Message 接口严格度~~ → 见 D3（不强制 Message 接口，分层标准框架 + opt-in）。
4. ~~MVP 边界~~ → 见 D4。
5. ~~包结构~~ → 见 D5（扁平单包，遵循 common/*x 惯例）。

所有阻塞规划的待决项已解决，进入 design.md + implement.md。

## Notes

- prd 仅记需求/约束/验收；技术设计写 design.md，执行计划写 implement.md，复杂任务三件齐备后再 task.py start。
