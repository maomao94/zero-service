# gnetx — 基于 gnet 的开箱即用 TCP 框架

> **EXPERIMENTAL** — 此包尚未经过生产环境验证，API 和行为可能在后续版本中变化。生产使用前需充分测试。

`common/gnetx` 在 `github.com/panjf2000/gnet/v2` 之上封装编解码、会话、路由、请求-响应、空闲检测和优雅停止，让开发者无需直接接触 gnet 的 `EventHandler`/`Peek`/`Discard` 即可搭建自定义二进制 TCP 协议。

## Pre-Development Checklist

在 `common/gnetx/` 中开发新功能或修复 bug 前：

1. **确定改动层级** — Core 层（所有协议都用）还是 opt-in 层（Router / Request-Response）
2. **阅读对应 topic spec** — 根据改动范围查阅下方列出的 spec 文件
3. **gnet 线程契约** — `OnTraffic` 内禁止阻塞操作；重活用 `Async`/`AsyncFunc` 标记 offload
4. **Codec 半包契约** — `Decode` 在缓冲区不足时必须返回 `ErrIncompletePacket`，不消费字节
5. **Session 生命周期** — OnOpen 创建 / OnClose 清理；`SetContext` 只能在 event-loop 线程（OnOpen）调用
6. **⚠️ 实验性约束** — 新增功能标记 `// EXPERIMENTAL`，改动前检查兼容性

## Package Architecture

```
common/gnetx/
├── codec*.go         # 编解码器：Codec 接口 + LengthPrefix/Delimiter/FixedLength 内置实现
├── serializer.go     # Serializer 接口 + RawSerializer/JSONSerializer 内置实现
├── server.go         # Server：gnet.EventHandler + go-zero service.Service
├── client.go         # Client：单连接模型，构造即拨号，固定间隔自动重连
├── session.go        # Session + SessionManager + SessionListener
├── handler.go        # Handler/HandlerFunc/AsyncHandler + Async/AsyncFunc 标记
├── router.go         # Router：按 messageID 路由的 Handler 容器（opt-in）
├── message.go        # 消息 opt-in 接口：Identifiable/Correlatable/Response/ClientIdentifiable
├── request.go        # 请求-响应 opt-in 层文档说明
├── errors.go         # 包级哨兵错误（ErrIncompletePacket/ErrSessionClosed 等）
├── options.go        # ServerOptions/ClientOptions + With* 选项函数
├── idle.go           # idleSweeper：独立 goroutine 空闲扫描
├── logger.go         # logxAdapter：gnet 日志 → go-zero logx
├── trace.go          # OTel tracing（span 创建 + attributes）
└── doc.go            # go doc 包级文档
```

## Spec Files

| 文件 | 说明 |
|------|------|
| [codec.md](codec.md) | Codec/Serializer：接口契约、内置实现、自定义、frameLimit 注入 |
| [server.md](server.md) | Server：构造、生命周期、service.Group 集成、空闲扫描 |
| [client.md](client.md) | Client：单连接模型、重连、响应式 API（Send/Notify/Request） |
| [handler.md](handler.md) | Handler/Router：handler 签名、sync/async 分发、Router 注册与异步 |
| [session.md](session.md) | Session/SessionManager：生命周期、属性、alias 冲突、ReplyPool |
| [request-response.md](request-response.md) | 请求-响应 opt-in：Correlatable/Response 接口、tid 关联、线程约束 |

## Quality Check

完成代码改动后执行：

```bash
go test -race -count=1 ./common/gnetx/   # 所有测试必须通过（含 race detector）
go vet ./common/gnetx/                    # 无 vet 警告
gofmt -l common/gnetx/                   # 无格式差异
```

- Server/Client 集成测试用 `freePort` + `startServer` helper，不占固定端口
- 新增测试优先考虑边界场景（断开态、错误路径、重连竞争）
- 保持测试函数独立：`t.Helper()` 标记 helper，`defer func(){}()` 清理
