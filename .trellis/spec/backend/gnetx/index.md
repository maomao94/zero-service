# gnetx — 基于 gnet 的开箱即用 TCP 框架

`common/gnetx` 在 `github.com/panjf2000/gnet/v2` 之上封装编解码、会话、路由、请求-响应、空闲检测和优雅停止，让开发者无需直接接触 gnet 的 `EventHandler`/`Peek`/`Discard` 即可搭建自定义二进制 TCP 协议。

## Pre-Development Checklist

1. **确定改动层级** — Core 层（codec/session）还是 opt-in 层（Router / Request-Response / PacketContextProvider）
2. **阅读对应 topic spec** — 下方列出的 spec 文件
3. **gnet 线程契约** — `OnTraffic` 内禁止阻塞；重活用 `Async`/`AsyncFunc` mark offload
4. **Codec 半包契约** — `Decode` 返回 `ErrIncompletePacket` 时不消费字节
5. **Session 生命周期** — OnOpen 创建 / OnClose 清理
6. **ReplyPool 所有权** — Session 持有非拥有型引用；Server/Client 管理池生命周期
7. **PacketContextProvider** — 如果 Codec 需要在回包时填 ack/seq，让消息实现 `PacketContextProvider`；框架在 dispatch 阶段注入 ctx
8. **OnConnect 回调** — 每次连接（含重连）触发，独立 goroutine 执行；`ConnectTimeout` 控制超时；已取代单次 `OnReady`
9. **DebugSerializer** — 包装 Serializer，debug 级别输出 hex 日志；不再使用 base64

## Package Architecture

```
common/gnetx/
├── codec*.go         # 编解码器：Codec + Serializer 接口 + LengthPrefix/Delimiter/FixedLength + DebugSerializer
├── serializer.go     # RawSerializer / JSONSerializer
├── server.go         # Server：gnet.EventHandler + go-zero service.Service
├── client.go         # Client：长连接 + 固定间隔自动重连
├── dialer.go         # Dialer：短连接 + 共享 gnet.Client 引擎 + Promise 匹配
├── session.go        # Session + SessionManager + SessionListener
├── handler.go        # Handler/HandlerFunc/AsyncHandler + Async/AsyncFunc
├── router.go         # Router：按 messageID 路由的 Handler 容器
├── message.go        # 消息 opt-in 接口：Identifiable/Correlatable/Response/PacketContextProvider + PacketContextKey
├── errors.go         # 包级哨兵错误
├── options.go        # ServerOptions/ClientOptions + With* 选项函数
├── idle.go           # idleSweeper：独立 goroutine 空闲扫描
├── logger.go         # logxAdapter：gnet 日志 → go-zero logx
├── trace.go          # OTel tracing
└── doc.go            # go doc 包级文档
```

## Spec Files

| 文件 | 说明 |
|------|------|
| [codec.md](codec.md) | Codec/Serializer 接口、LengthPrefix（含 leading/trailing/stripBytes）、Delimiter、FixedLength、DebugSerializer、PacketContext 回包 |
| [server.md](server.md) | Server：构造、共享 ReplyPool、生命周期、dispatch ctx 注入、service.Group 集成 |
| [client.md](client.md) | Client（长连接）+ Dialer（短连接）：构造、ReplyPool、重连、Promise 匹配、dispatch ctx 注入 |
| [handler.md](handler.md) | Handler/Router：签名、ctx 含 PacketContext、sync/async 分发、Router 注册 |
| [session.md](session.md) | Session/SessionManager：NextSendSeq、复合 TID、共享 ReplyPool、生命周期 |
| [request-response.md](request-response.md) | 请求-响应 opt-in：Correlatable/Response、复合 TID、线程约束 |

## Quality Check

```bash
go test -race -count=1 ./common/gnetx/   # 所有测试必须通过
go vet ./common/gnetx/                    # 无 vet 警告
```
