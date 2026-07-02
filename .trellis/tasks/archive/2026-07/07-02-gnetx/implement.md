# gnetx 执行计划（implement.md）

> 任务：.trellis/tasks/07-02-gnetx ｜ 包：`common/gnetx`（扁平，package gnetx）
> 设计依据：design.md ｜ 决策依据：prd.md D1-D5

## 前置

- [ ] 0.1 加依赖：`go get github.com/panjf2000/gnet/v2@v2.9.8`（确认 go.mod/go.sum 更新；Go 1.26 兼容）。
- [ ] 0.2 读 `common/antsx` 公开 API（ReplyPool/RequestReply/Reactor/NewPromise）确认调用签名（已读 replypool.go，实现时再核 reactor.go/promise.go）。
- [ ] 0.3 读项目 logx 用法约定（`common/antsx` 里 `logx.Debugw/Statf` 范式）保持日志风格一致。

## 实现顺序（每步独立可编译，配同名 _test.go）

### 阶段 A — 契约与分帧（无 gnet 运行时依赖，纯逻辑可单测）

- [ ] A1 `errors.go`：哨兵错误 `ErrIncompletePacket`/`ErrSessionClosed`/`ErrFrameTooLarge`/`ErrNoHandler`/`ErrPendingNotFound` + `errors_test.go`。
- [ ] A2 `message.go`：opt-in 接口 `Identifiable`/`Correlatable`/`Response`/`ClientIdentifiable` + `message_test.go`（用小 stub struct 验证类型断言）。
- [ ] A3 `codec.go`：`Codec`/`Framer`/`Serializer` 接口 + `funcCodec`/`NewCodec`/`NewFuncCodec` + 半包错误映射工具。
- [ ] A4 `codec_lengthprefix.go` + 测试：`LengthPrefixFramer`（uint16/uint32 BE/LE，offset/adjust/maxFrameLength）；测半包（Peek ShortBuffer→ErrIncompletePacket）、整包、超长、多帧粘包。
- [ ] A5 `codec_delimiter.go` + 测试：`DelimiterFramer`（单/多分隔符，strip 选项）；测分隔符跨包边界、strip true/false。
- [ ] A6 `codec_fixed.go` + 测试：`FixedLengthFramer`；测不足半包、整包。
- [ ] A7 `serializer.go` + 测试：`RawSerializer`（原样字节）、`JSONSerializer`（快速原型）。

验证：`go build ./common/gnetx/... && go vet ./common/gnetx && go test ./common/gnetx -run 'Codec|Framer|Serializer|Message|Error'`。

### 阶段 B — Session 与 Manager（仍无 gnet 运行时，用 fake conn 接口测）

- [ ] B1 `session.go`：`Session` 结构 + 方法（ID/Alias/RemoteAddr/attrs/Register/Send/Close/touch/resolveResponse）+ `SessionManager` + `noopSessionListener` + `SessionListener`。
  - `Send` 暂用 `conn.AsyncWrite`（需 gnet.Conn，先定义最小 conn 接口供测试 mock，或留到阶段 D 接真 gnet.Conn）。
  - `resolveResponse`/`initPool` 先占位（pool 在阶段 C 接 antsx）。
- [ ] B2 `session_test.go` + `manager_test.go`：attrs 并发、Register/Get/Remove、Close 幂等、listener 回调。

验证：`go build ./common/gnetx && go test ./common/gnetx -run 'Session|Manager'`。

### 阶段 C — Handler/Router + 请求-响应（接 antsx）

- [ ] C1 `handler.go`：`Handler`/`HandlerFunc`/`AsyncHandler` 接口 + 慢处理计时封装。
- [ ] C2 `router.go` + 测试：`Router` 实现 Handler；`Handle`/`HandleFunc`/`HandleTyped[T]`/`Async`/`Fallback`/`RegisterType`；测 id 命中/未命中 fallback/类型断言失败。
- [ ] C3 `request.go`：`Session.initPool`/`Request`/`Notify`/`resolveResponse` 接 `antsx.ReplyPool`/`RequestReply`；断连 `pool.Close`。
- [ ] C4 `request_test.go`：用 mock conn + 真实 antsx ReplyPool 测 Request 成功/超时/断连 Reject/重复 tid。

验证：`go build ./common/gnetx && go test ./common/gnetx -run 'Router|Request|Reply'`。

### 阶段 D — Server（接 gnet 运行时）

- [ ] D1 `options.go`：`ServerOptions`/`ClientOptions` + `With*` 选项 + `validate()`（Addr/Codec/Handler/MaxFrameLength 必填，MaxFrameLength 合理上限）。
- [ ] D2 `idle.go`：空闲扫描 goroutine + `OnIdle` 钩子位。
- [ ] D3 `server.go`：`Server` 实现 `gnet.EventHandler`（OnBoot/OnOpen/OnTraffic/OnClose/OnShutdown）+ `NewServer`/`Run`/`Shutdown` + 同步/异步 dispatch + 慢处理告警 + decode 错误策略。
- [ ] D4 `server_test.go`：用 `net.Dial` 起真 client 对打（LengthPrefix + echo handler）测：连接/收发/半包粘包/空闲关闭/优雅 Shutdown/Response 自动路由 Request 回包。用 `go test` 启 server，goroutine 里 net.Dial 写读。

验证：`go build ./common/gnetx && go vet ./common/gnetx && go test ./common/gnetx -run Server -count=1`。

### 阶段 E — Client

- [ ] E1 `client.go`：`Client` 实现 `gnet.EventHandler` + `NewClient`/`Start`/`Dial`/`DialContext`/`Shutdown` + Session 创建（isClient=true）。
- [ ] E2 `client_test.go`：起 server（D3）+ gnetx Client.Dial 对打测 Request/Notify/Response 路由/断连清理。

验证：`go build ./common/gnetx && go vet ./common/gnetx && go test ./common/gnetx -run Client -count=1`。

### 阶段 F — 收尾

- [ ] F1 集成示例：在 `common/gnetx/example_test.go`（或 `socketapp` 下加示例）写一个 < 50 行的最小自定义协议（LengthPrefix + 1 Message + 1 handler + Client Request），对应 AC1。
- [ ] F2 全量验证：`go vet ./common/gnetx && go build ./... && go test ./common/gnetx -count=1 -race`。
- [ ] F3 逐条核对 AC1-AC11。
- [ ] F4 加包注释 `doc.go`（package gnetx 用法概览 + 一个最小示例片段）。

## 验证命令汇总

```bash
go build ./common/gnetx/...
go vet ./common/gnetx
go test ./common/gnetx -count=1 -race
go build ./...                 # 确认不破坏整体编译
```

## 风险与回滚点

- **gnet Peek/Discard 半包语义**：必须正确映射 `io.ErrShortBuffer → ErrIncompletePacket` 并 `break`（不 Discard）。写错会丢字节。回滚：阶段 A 测试先行，红绿确保正确再进 D。
- **on-loop 阻塞**：sync handler 若用户写重活会拖死 loop。文档+慢处理告警兜底；不可控则建议改 AsyncHandler。
- **AsyncWrite 回包时机**：async handler 回包走 AsyncWrite（off-loop），需确认 gnet AsyncCallback 不阻塞。回滚：async 路径先简单 `AsyncWrite(buf,nil)`，再优化回调。
- **ReplyPool 每 Session 开销**：连接数极大时 TimingWheel 多。MVP 接受；若压测有问题，后续改共享 TW + 每 Session map（design §7 已记）。
- **macOS 多 loop 无 SO_REUSEPORT LB**：测试在 mac 跑，accept 不线性扩展；功能不受影响，仅性能。文档注明。

## 实现前 review gate

- prd.md / design.md / implement.md 三件齐备 → 用户 review → `task.py start` → 进 Execute。
