# gnetx 重构：全局 ReplyPool + 复合 TID + 短连接 Client

## Goal

优化 gnetx 的 ReplyPool 架构，消除每次新连接创建新池的开销；同时扩展 Client 支持短连接模式。

## 现状分析

### ReplyPool 当前架构（问题）

- **每 Session 一个 `antsx.ReplyPool[any]`**（`session.go:40-41`），懒初始化在 `ensurePool()`（`session.go:160-173`）
- 池生命周期绑 Session：`OnOpen` 创建 Session → 首次 `Request()` 创建池 → `OnClose` 通过 `Session.Close()` 关闭池
- 每个 antsx.ReplyPool 内部有：map[pending]、TimingWheel（300 槽）、statLoop goroutine（每分钟）
- **问题**：Server 每接入一个新客户端连接，就创建一个新池（含 TimingWheel + goroutine），资源浪费

### TID 当前机制

- `Correlatable.TID()` 返回纯字符串，通常由业务用 int 序号转换（如 `strconv.Itoa(m.Serial)`）
- 无 session 前缀，TID 只在单 Session 池内唯一
- 入站 Response 通过 `sess.resolveResponse(resp.ResponseTID())` 直接在 Session 自己的池中 Resolve

### Client 当前模型

- **单一长连接模型**：构造即拨号，断线自动重连（固定间隔）
- 一个 Client = 一个远端地址
- Client 本身不持有 ReplyPool，委托给内部 Session（也是每 Session 一个池）
- 无短连接/一次性的 dial 能力

## Requirements

### R1: Server 级全局 ReplyPool

- Server 构造时创建一个 `antsx.ReplyPool[any]`，所有 Session 共享
- Session 不再独立创建池，移除 `ensurePool()`、`pool`、`poolOnce` 字段
- Session.Request 委托到 Server 的共享池
- Server.OnTraffic 中 Response 匹配也用共享池

### R2: 复合 TID = session_id + original_tid

- 在 Session.Request 中将 `sessionID + 分隔符 + msg.TID()` 作为 Register 的 id
- 在 resolveResponse（Server.OnTraffic 中）用 `sessionID + 分隔符 + resp.ResponseTID()` 作为 Resolve 的 id
- 用户代码无需改动：`Correlatable.TID()` 和 `Response.ResponseTID()` 接口不变
- 分隔符需选择一个不存在于 sessionID 中的字符（sessionID 来自 `RemoteAddr().String()`，如 "127.0.0.1:12345"）

### R3: Client 支持短连接模式

- 保留现有单连接长连接 Client（不变）
- 新增短连接拨号器（Dialer），支持 `Dial(ctx, network, address)` 模式
- Dialer **不持有 ReplyPool**，短连接直接用 `antsx.Promise` 做单次请求-响应
- `Dial(ctx, network, address) (*Session, error)` — 返回 Session（仅支持 Send/Notify，replyPool=nil）
- `Request(ctx, network, address, msg Correlatable) (any, error)` — 一键拨号+发请求+等回包+关连接

## Acceptance Criteria

- [ ] Server 全局只创建一个 ReplyPool，不再每连接创建
- [ ] 复合 TID 保证多 Session 共享池时 TID 不冲突
- [ ] 现有所有测试（server_test、client_test、bidirectional_test）通过
- [ ] 长连接 Client 的 Request 功能正常工作
- [ ] Dialer 支持 `Dial` 拨号返回 Session
- [ ] Dialer 支持 `Request` 一键请求-响应（基于 Promise，无 ReplyPool）
- [ ] 无 goroutine 泄漏（TimingWheel、statLoop 正确回收）
- [ ] ReplyPool 的 Close 在 Server Shutdown 时正确清理
- [ ] Server OnTick 输出连接统计：active、connects/min、disconnects/min

## Out of Scope

- antsx.ReplyPool 本身的接口变更（不需要前缀清理/批量 Reject 等功能）
- ServerOptions/ClientOptions 的兼容性（如需要新增选项则设计向后兼容）

## Decisions

1. **Client 重连时在途请求**：等 TTL 自然过期。不修改 antsx.ReplyPool。池在 Client 级别存活，断连期间在途请求不立即 Reject。
2. **短连接客户端命名**：`Dialer`。API：`Dial(ctx, network, address) (*Session, error)` + `Request(ctx, network, address, msg, ttl) (any, error)`。
3. **复合 TID 分隔符**：`"|"`。格式 `sessionID + "|" + originalTID`，如 `"127.0.0.1:54321|1"`。sessionID 不含 `|`，仅作 key 用，永不拆分。
4. **Dialer.Request 去掉 TTL**：`Request(ctx, network, address, msg)`，内部从 `ctx.Deadline()` 推导 TTL，API 更简洁，符合 HTTP 风格。
