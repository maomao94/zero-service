# antsx 选型与项目实践

> `antsx` 是项目内自研的并发工具包（`common/antsx`），用 Go 的 goroutine、channel 与泛型实现 Promise/Await、流处理、发布订阅、并发编排和协程池。不引入响应式框架，不引入额外依赖。

本文只讲**为什么这样设计、什么时候用哪个原语、项目里怎么用**。各原语的完整签名、参数与示例见 [`common/antsx/README.md`](../common/antsx/README.md)。

## 设计原则

1. **goroutine 即并发** — 不引入事件循环、线程池调度或操作符链。并发就是 `go func()`，等待就是 `Await`。
2. **标准错误处理** — 所有异步路径最终收敛到 `if err != nil`，调用栈完整，调试时断点直指业务代码。
3. **泛型类型安全** — 编译期保证 `Promise[T]`、`StreamReader[T]`、`Task[T]` 的类型正确性，无运行时反射开销。
4. **零可选依赖** — 仅依赖 `errgroup`、`ants`（协程池）、`go-zero TimingWheel`（TTL）和 `threading.GoSafe`（panic 恢复），不绑定任何框架。
5. **组合优于继承** — 每个原语独立可用，也可组合使用。`Promise` 可被 `Invoke` 调度，`StreamReader` 可被 `EventEmitter` 订阅，`Reactor` 可限制任意操作的并发度。

## 选型速查

| 场景 | 用什么 |
| --- | --- |
| 单个后台计算，要拿返回值 | `Go(ctx, fn)` → `*Promise[T]` |
| 单个后台计算，要限并发 | `Submit(ctx, reactor, fn)` |
| 单个后台计算，不关心返回值 | `Post(ctx, reactor, fn)` 或 `reactor.Go(ctx, fn)` |
| N 个任务并行，全成功才算成功 | `Invoke(ctx, tasks...)` |
| N 个任务并行，各自独立互不影响 | `InvokeAllSettled(ctx, tasks...)` |
| 上面两种要限并发 | `InvokeWithReactor` / `InvokeAllSettledWithReactor` |
| 已有多个 Promise，等全部完成 | `PromiseAll` / `PromiseAllSettled` |
| 已有多个 Promise，取最快 | `PromiseRace`（不论成败）/ `PromiseAny`（要成功） |
| 发请求，等对方按 ID 回包 | `ReplyPool` + `RequestReply` |
| 流式数据逐条传递 | `Pipe` + `StreamReader` / `StreamWriter` |
| 一份数据广播给多个消费者 | `EventEmitter` 或 `StreamReader.Copy` |
| 生产者消费者解耦，无界缓冲 | `UnboundedChan` |

按场景选 API 的完整对照、`Submit` 与 `ReplyPool` 的区别，见 [`common/antsx/README.md`](../common/antsx/README.md#我该用哪个)。

## 项目中的使用场景

| 服务 | 使用的原语 | 场景 |
| --- | --- | --- |
| `common/mcpx` | `Invoke` + `EventEmitter` + `Reactor` + `Promise` | MCP 工具并行调用编排、进度事件分发、协程池并发控制 |
| `app/file` | `TeeWriter` + `Reactor` + `InvokeAllSettled` | 文件流同时写入 OSS 与计算 MD5，多目标并行上传 |
| `app/trigger` | `Invoke` + `InvokeAllSettled` + `Reactor` | 计划任务并行执行、fast-fail 取消、HTTP/gRPC 回调编排 |
| `common/mqttx` | `ReplyPool` + `RequestReply` | MQTT 请求-响应关联与异步 RPC 超时控制 |
| `common/iec104` | `ReplyPool` | 控制命令 ACK 匹配 |
| `common/gnetx` | `ReplyPool` + `Promise` | TCP session 请求-响应匹配 |

## 不适合 antsx 的场景

| 场景 | 推荐方案 |
| --- | --- |
| 跨语言通信 | gRPC Stream |
| 持久化消息队列 | Kafka |
| 复杂背压策略 | antsx 的 `Pipe` 只有 channel 缓冲，无动态背压 |
| 完整响应式操作符生态 | antsx 只提供核心原语，不是框架 |

## 相关文档

- 原语实现与示例：[`common/antsx/README.md`](../common/antsx/README.md)
- 错误码与超时约定：[错误码规范](./error-codes.md)
