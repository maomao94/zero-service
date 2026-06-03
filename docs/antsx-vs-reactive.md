# antsx vs 响应式框架对比

## 为什么不用 Java WebFlux / RxJava

### 一句话总结

antsx 用 Go 的 goroutine + channel + 泛型实现了响应式核心语义（流式处理、异步编排、发布订阅），避免了 Java 响应式框架的操作符地狱、背压心智负担和堆栈不可读。

---

## 核心对比

| 维度 | antsx (Go) | WebFlux (Java) |
|------|------------|----------------|
| **并发模型** | goroutine，操作系统线程调度 | 事件循环 + Reactor 线程池 |
| **异步原语** | `Promise[T]` + `Await` | `Mono<T>` + `subscribe` |
| **流处理** | `Pipe` + `Recv`（同步 for 循环） | `Flux<T>` + 操作符链 |
| **错误处理** | 标准 `if err != nil`，调用栈完整 | `onErrorResume` / `doOnError`，栈丢失 |
| **并发编排** | `Invoke` / `InvokeAllSettled` | `Mono.zip` / `Flux.merge` |
| **类型系统** | Go 泛型，编译期检查 | Java 泛型 + 类型擦除 |
| **调试体验** | 普通 goroutine，断点直观 | 操作符链断点跨越多个线程 |
| **学习曲线** | 会用 goroutine 就能上手 | 需要理解响应式编程范式 |

---

## 场景对比

### 1. 并发等待多个 API

**Java WebFlux**:
```java
Mono<String> user = webClient.get().uri("/user").retrieve().bodyToMono(String.class);
Mono<String> order = webClient.get().uri("/order").retrieve().bodyToMono(String.class);
Tuple2<String, String> result = Mono.zip(user, order).block();
```

心智负担：线程模型不透明，`.block()` 不能随便用，错误行号指向框架内部。

**antsx (Go)**:
```go
results, err := antsx.Invoke(ctx,
    antsx.Task[string]{Name: "user", Fn: func(ctx context.Context) (string, error) {
        return httpGet(ctx, "/user")
    }},
    antsx.Task[string]{Name: "order", Fn: func(ctx context.Context) (string, error) {
        return httpGet(ctx, "/order")
    }},
)
if err != nil {
    log.Println(err) // 完整的调用栈，能直接定位到你的代码
}
```

心智负担：0。就是起两个 goroutine 并行执行。

### 2. 流式处理 + 过滤 + 类型转换

**Java WebFlux**:
```java
Flux.fromIterable(list)
    .filter(i -> i > 0)
    .map(i -> "v" + i)
    .subscribe(...);
```

心智负担：防抖、背压策略（BUFFER/DROP/LATEST）、subscribeOn/publishOn 线程模型。

**antsx (Go)**:
```go
sr := antsx.StreamReaderWithConvert(
    antsx.StreamReaderFromArray(arr),
    func(i int) (string, error) {
        if i == 0 {
            return "", antsx.ErrNoValue // 过滤
        }
        return fmt.Sprintf("v%d", i), nil
    },
)
for {
    val, err := sr.Recv()
    if errors.Is(err, io.EOF) { break }
    fmt.Println(val)
}
```

心智负担：就是 for 循环 + 函数调用，调试时能看到每行代码。

### 3. 发布订阅

**Java WebFlux**（需借助 Reactor 的 Sinks 或 Project Reactor extra）:
```java
Sinks.Many<String> sink = Sinks.many().multicast().onBackpressureBuffer();
sink.asFlux().subscribe(System.out::println);
sink.tryEmitNext("hello");
```

**antsx (Go)**:
```go
emitter := antsx.NewEventEmitter[string]()
defer emitter.Close()

sr, cancel := emitter.Subscribe(ctx, "topic")
defer cancel()
go func() {
    for {
        val, err := sr.Recv()
        if errors.Is(err, io.EOF) { break }
        fmt.Println(val)
    }
}()

emitter.Emit("topic", "hello")
```

心智负担：就是 channel Pub/Sub，goroutine 退出由 context 控制。

---

## antsx 提供了什么亮点

### 1. 零依赖的 Promise 模型

```go
p := antsx.NewPromise[int]()
go func() { p.Resolve(42) }()
val, err := p.Await(ctx) // 阻塞直到 Resolve/Reject 或 ctx 取消
```

没有操作符链，不需要理解 subscribe/publish 的区别。`Await` 就是等结果。

### 2. 编译期展开的静态 select

多流合并时，≤5 路使用编译时展开的 `select{}` 语句，零反射开销；>5 路才降级 `reflect.Select`。

### 3. 零拷贝 fan-out

`Copy` 用链表 + `sync.Once` 实现广播，多个消费者共享同一份数据，无需复制。

### 4. 三层 panic 防护

goroutine → GoSafe 边界 → 任务执行层，任何一层 panic 都不会导致 `WaitGroup` 泄漏。

### 5. 类型安全的流转换

`StreamReaderWithConvert` 用 Go 泛型做类型转换 + 过滤，编译期保证类型正确。

### 6. 协程池集成

所有原语都有 `WithReactor` 变体，可在 ants 协程池内执行，控制并发度。

---

## 项目中使用场景

| 服务 | 使用的 antsx 原语 | 场景 |
|------|-------------------|------|
| **djicloud**（间接） | `PendingRegistry` | DJI Cloud API MQTT 请求-响应关联，异步 RPC 超时控制 |
| **mcpx** | `Invoke` + `EventEmitter` + `Reactor` + `Promise` | MCP 工具并行调用编排，进度事件分发，协程池并发控制 |
| **file** | `TeeWriter` + `Reactor` + `InvokeAllSettled` | 文件流同时写入 OSS + 计算 MD5，多目标并行上传 |
| **aiapp/ssegtw** | `EventEmitter` + `PendingRegistry` | SSE 事件流推送，流完成信号通知 |
| **aiapp/aichat** | `Promise` + `Stream` | AI 对话流式输出，异步流接收带超时控制 |
| **trigger** | `Invoke` + `InvokeAllSettled` + `Reactor` | 计划任务并行执行，fast-fail 失败快速取消，HTTP/gRPC 回调编排 |

---

## 什么场景不适合 antsx

- 需要跨语言通信（用 gRPC Stream）
- 需要持久化消息队列（用 Kafka）
- 需要复杂的背压策略（antsx 的 `Pipe` 只有 channel 缓冲，无动态背压）
- 需要完整的响应式操作符生态（antsx 只提供核心原语，不是框架）

---

## 一句话

antsx 的价值不是"实现了一个响应式框架"，而是**用 Go 现有的 goroutine + channel 思维，把并发、流式、异步这些事做到最简单**。
