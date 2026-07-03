// Package gnetx 提供基于 gnet 的开箱即用 TCP 框架。
//
// gnetx 在 github.com/panjf2000/gnet/v2 之上封装了编解码、会话、路由、请求-响应、
// 空闲检测和优雅停止，让开发者无需直接接触 gnet 的 EventHandler/Peek/Discard 即可
// 快速搭建自定义二进制 TCP 协议的服务端与客户端。
//
// # 设计原则
//
//   - 不抄 netmc 的 Spring-MVC 注解那套，做 Go 标准的 TCP 框架。
//   - Core 层（所有协议都用）：Codec、Conn、Handler、Server/Client、空闲检测、优雅停止。
//   - opt-in 层（需要才用）：Router（按 messageID 路由）、请求-响应（tid 响应式，基于 antsx.ReplyPool）。
//   - 纯推送/遥测协议只用 Core，零 tid/路由包袱。
//
// # Codec —— 分帧与序列化一体
//
// Codec 单接口承载分帧+序列化（对齐 gnet v1 ICodec 的简洁形态）：
//
//	type Codec interface {
//	    Decode(c gnet.Conn, conn Conn) (any, error) // 半包返回 ErrIncompletePacket
//	    Encode(msg any, conn Conn) ([]byte, error)
//	}
//
// 内置 LengthPrefixCodec / DelimiterCodec / FixedLengthCodec 直接实现本接口，
// 配合 Serializer（只管 raw 字节↔消息转换）开箱即用；用户自定义协议实现 Codec 即可。
//
// # Server —— 接入 go-zero service.Group
//
// Server 实现 go-zero service.Service 接口（Start 阻塞 / Stop 停止），
// 可加入 service.NewServiceGroup() 统一管理生命周期：
//
//	srv, _ := gnetx.NewServer(gnetx.WithAddr("tcp://:9000"), ...)
//	sg := service.NewServiceGroup()
//	sg.Add(srv)
//	sg.Start() // 阻塞，proc 信号触发 Stop
//
// 也可直接 srv.Run() 阻塞运行，srv.Shutdown(ctx) 优雅停止。
//
// # Client —— 单连接模型
//
// Client 是单连接模型（对标 mqttx/modbusx），构造即拨号，断线按固定间隔自动重连：
//
//	cli := gnetx.MustNewClient("tcp", "127.0.0.1:9000",
//	    gnetx.WithClientCodec(myCodec),
//	    gnetx.WithClientHandler(myHandler),
//	    gnetx.WithClientMaxFrameLength(1<<20),
//	)
//	defer cli.Close()
//
// 响应式 API：cli.Send(ctx, msg) / cli.Request(ctx, msg, ttl)。
// 不实现 service.Service（无 Start/Stop），生命周期仅构造 + Close。
//
// # 请求-响应（opt-in，tid 响应式）
//
// 消息实现 Correlatable（请求侧 TID()）和 Response（回包侧 ResponseTID()）后，
// Requester.Request 发送请求并等待匹配 tid 的回包：
//
//	resp, err := conn.Request(ctx, &MyReq{Serial: 1}, 10*time.Second)
//
// 关联引擎为 Server/Client 级共享 common/antsx.ReplyPool，断连不会立即 Reject 在途请求。
// 入站回包（实现 Response）由框架自动路由到 ReplyPool.Resolve，不进业务 handler。
// 禁止在 on-loop handler 中调用 Request（会阻塞 event-loop）。
//
// # 线程契约
//
//   - OnTraffic 在 event-loop goroutine 同步执行，handler 必须快；
//     重活用 AsyncFunc/Async 标记，框架 offload 到 gnet 自带的 goroutine.DefaultWorkerPool，
//     回包走 AsyncWrite。
//   - on-loop only：Codec.Decode（Peek/Discard）、同步 c.Write、c.Context。
//   - off-loop 安全：Conn.Send/Request（内部 AsyncWrite）、
//     Conn.Close、SessionManager、antsx.ReplyPool。
//
// # 日志
//
// gnet 内部日志通过 gnet.WithLogger 注入 logx 适配器，与项目其他 common/*x 包统一走 go-zero logx。
//
// 详见 design.md 与各文件注释。
package gnetx
