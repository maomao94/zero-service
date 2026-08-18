# 使用 antsx 表达并发与异步编排

antsx 是项目内自研的并发原语库（`common/antsx`），用 Go 的 goroutine、channel 与泛型实现 Promise/Await、Pipe 流处理、EventEmitter 发布订阅、Invoke/InvokeAllSettled 并发编排和协程池，被 djicloud、file、trigger 等服务使用。项目决策采用 antsx 而非 Java WebFlux/RxJava 式响应式框架，也不为每个场景引入独立异步框架。

原因是：Go 的 goroutine 模型天然适合表达并发，antsx 只用少量核心原语就覆盖了项目的编排需求，同时保留标准 `if err != nil` 错误处理与完整调用栈，避免了响应式框架的操作符链、背压心智负担和调试困难。antsx 不适用于跨语言通信（用 gRPC Stream）、持久化消息队列（用 Kafka）和复杂背压场景。详细对比见 [antsx vs 响应式框架对比](../antsx-vs-reactive.md)。
