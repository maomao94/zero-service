# antsx 并发原语指南

> antsx 是项目内自研的并发工具包（`common/antsx`），用 Go 的 goroutine、channel 与泛型实现 Promise/Await、流处理、发布订阅、并发编排和协程池。不引入 Java 响应式框架，不引入额外依赖。

---

## 目录

- [设计原则](#设计原则)
- [核心 API 速查](#核心-api-速查)
  - [Promise](#promise)
  - [Invoke — 并发编排](#invoke--并发编排)
  - [StreamReader / StreamWriter — 流处理](#streamreader--streamwriter--流处理)
  - [EventEmitter — 发布订阅](#eventemitter--发布订阅)
  - [Reactor — 协程池](#reactor--协程池)
  - [ReplyPool — 请求-响应关联](#replypool--请求-响应关联)
  - [TeeWriter — 写入分发](#teewriter--写入分发)
  - [UnboundedChan — 无界阻塞队列](#unboundedchan--无界阻塞队列)
- [项目中的使用场景](#项目中的使用场景)
- [不适合 antsx 的场景](#不适合-antsx-的场景)

---

## 设计原则

1. **goroutine 即并发** — 不引入事件循环、线程池调度或操作符链。并发就是 `go func()`，等待就是 `Await`。
2. **标准错误处理** — 所有异步路径最终收敛到 `if err != nil`，调用栈完整，调试时断点直指业务代码。
3. **泛型类型安全** — 编译期保证 `Promise[T]`、`StreamReader[T]`、`Task[T]` 的类型正确性，无运行时反射开销。
4. **零可选依赖** — 仅依赖 `errgroup`、`ants`（协程池）、`go-zero TimingWheel`（TTL）和 `threading.GoSafe`（panic 恢复），不绑定任何框架。
5. **组合优于继承** — 每个原语独立可用，也可组合使用。`Promise` 可被 `Invoke` 调度，`StreamReader` 可被 `EventEmitter` 订阅，`Reactor` 可限制任意操作的并发度。

---

## 核心 API 速查

### Promise

异步结果容器。`Resolve` / `Reject` 只生效一次（`sync.Once`），所有方法 goroutine 安全。

```go
p := antsx.NewPromise[int]()

// 后台写入
go func() { p.Resolve(42) }()

// 阻塞等待
val, err := p.Await(ctx) // val == 42
```

| 方法 | 作用 |
|------|------|
| `Resolve(val)` | 完成，写入成功值（仅首次生效） |
| `Reject(err)` | 完成，写入错误（仅首次生效） |
| `Await(ctx)` | 阻塞直到完成或 ctx 取消 |
| `AwaitWithTimeout(d)` | 带超时的 Await |
| `Done()` | 返回 `<-chan struct{}`，可用于 `select` |
| `Get()` | 非阻塞获取 `(val, err, ok)` |
| `Catch(ctx, fn)` | 异步错误回调 |
| `FireAndForget(ctx)` | 后台等待，忽略结果 |

#### 链式组合

```go
// Then: 成功时转换值，失败时透传错误
p2 := antsx.Then(ctx, p1, func(v int) (string, error) {
    return fmt.Sprintf("result:%d", v), nil
})

// Map: 成功时转换值，不处理错误
p3 := antsx.Map(ctx, p1, func(v int) string {
    return fmt.Sprintf("result:%d", v)
})

// FlatMap: 成功时返回内部 Promise（异步链）
p4 := antsx.FlatMap(ctx, p1, func(v int) *Promise[string] {
    return antsx.Go(ctx, func(ctx context.Context) (string, error) {
        return fetchFromAPI(ctx, v)
    })
})
```

#### 批量等待

```go
// PromiseAll: 全部成功才成功，任一失败立即返回错误
results, err := antsx.PromiseAll(ctx, p1, p2, p3)

// PromiseAllSettled: 等待全部完成，独立返回每个结果
results := antsx.PromiseAllSettled(ctx, p1, p2, p3)
for _, r := range results {
    if r.Succeeded() {
        fmt.Println(r.Val)
    } else {
        fmt.Println(r.Err)
    }
}

// PromiseRace: 返回第一个完成的结果（不论成功或失败）
val, err := antsx.PromiseRace(ctx, p1, p2)

// PromiseAny: 返回第一个成功的结果，全部失败才返回错误
val, err := antsx.PromiseAny(ctx, p1, p2)
```

#### Go — 启动 goroutine 并返回 Promise

```go
p := antsx.Go(ctx, func(ctx context.Context) (string, error) {
    return httpGet(ctx, "/api/data")
})
// p 是 *Promise[string]，可以后续 Await / Then / Map
```

`Go` 内部自动恢复 panic，转为 `*panicErr`（包含完整堆栈）。

---

### Invoke — 并发编排

定义多个任务，并行执行，统一等待结果。

```go
results, err := antsx.Invoke(ctx,
    antsx.Task[string]{
        Name: "user",
        Fn: func(ctx context.Context) (string, error) {
            return httpGet(ctx, "/user")
        },
    },
    antsx.Task[string]{
        Name: "order",
        Fn: func(ctx context.Context) (string, error) {
            return httpGet(ctx, "/order")
        },
    },
)
// results[0] = user 结果, results[1] = order 结果（按输入顺序）
// 任一任务失败，ctx 被取消，其余任务快速失败
```

| 函数 | 行为 |
|------|------|
| `Invoke(ctx, tasks...)` | 并行执行，fast-fail（errgroup），结果按输入顺序 |
| `InvokeWithReactor(ctx, reactor, tasks...)` | 同上，但限制最大并发数（使用 Reactor 协程池） |
| `InvokeAllSettled(ctx, tasks...)` | 并行执行，独立结果（GoSafe），不相互取消 |
| `InvokeAllSettledWithReactor(ctx, reactor, tasks...)` | 同上 + 协程池限制 |
| `InvokeCallback(ctx, tasks, callback)` | Invoke + 聚合回调，一步到位 |

#### Task 带超时

```go
results, err := antsx.Invoke(ctx,
    antsx.Task[string]{
        Name:    "slow-api",
        Fn:      fetchSlowData,
        Timeout: 5 * time.Second, // 单个任务超时
    },
)
```

#### Invoke vs InvokeWithReactor

`Invoke` 使用 `errgroup`，goroutine 数量无上限。当任务数很大时，用 `InvokeWithReactor` 限制并发：

```go
reactor, _ := antsx.NewReactor(10) // 最多 10 个并发
defer reactor.Release()

results, err := antsx.InvokeWithReactor(ctx, reactor,
    antsx.Task[string]{Name: "t1", Fn: fn1},
    // ... 100 个任务
)
```

> **设计细节：** `InvokeWithReactor` 内部使用 `WaitGroup + errOnce + cancel` 而非 `errgroup`，因为 `ants.Pool.Submit` 在池满时会阻塞，无法响应 `errgroup` 的 ctx 取消信号。

---

### StreamReader / StreamWriter — 流处理

核心流抽象。`Pipe` 创建配对的 reader/writer，`StreamReaderFromArray` 从切片创建，`StreamReaderWithConvert` 做类型转换和过滤。

```go
// 基本管道
sr, sw := antsx.Pipe[string](16) // 缓冲区大小 16

go func() {
    sw.Send("hello", nil)
    sw.Send("world", nil)
    sw.Close()
}()

for {
    val, err := sr.Recv()
    if errors.Is(err, io.EOF) { break }
    fmt.Println(val)
}
```

#### 从切片创建零开销流

```go
sr := antsx.StreamReaderFromArray([]int{1, 2, 3, 4, 5})
for {
    val, err := sr.Recv()
    if errors.Is(err, io.EOF) { break }
    fmt.Println(val) // 1, 2, 3, 4, 5
}
```

#### 类型转换 + 过滤

```go
// ErrNoValue 作为过滤哨兵，跳过该元素
sr := antsx.StreamReaderWithConvert(
    antsx.StreamReaderFromArray([]int{0, 1, 2, 0, 3}),
    func(i int) (string, error) {
        if i == 0 {
            return "", antsx.ErrNoValue // 过滤掉 0
        }
        return fmt.Sprintf("item-%d", i), nil
    },
)
// 输出: item-1, item-2, item-3
```

#### Fan-out — 一个流分发给多个消费者

```go
sr := antsx.StreamReaderFromArray([]int{1, 2, 3})
children := sr.Copy(3) // 创建 3 个独立子 reader

for i, child := range children {
    go func(id int, c *StreamReader[int]) {
        for {
            val, err := c.Recv()
            if errors.Is(err, io.EOF) { break }
            fmt.Printf("consumer %d: %d\n", id, val)
        }
    }(i, child)
}
// 所有子 reader 关闭后，源 reader 自动关闭（零拷贝，链表 + sync.Once）
```

#### Fan-in — 多流合并

```go
sr1 := antsx.StreamReaderFromArray([]string{"a", "b"})
sr2 := antsx.StreamReaderFromArray([]string{"c", "d"})

merged := antsx.MergeStreamReaders(sr1, sr2)
// 输出顺序取决于到达时间: a, c, b, d 或任意交错
```

`MergeStreamReaders` 内部：≤5 路使用编译时展开的 `select{}`（零反射），>5 路降级 `reflect.Select`。

#### 带命名的合并（可识别来源）

```go
merged := antsx.MergeNamedStreamReaders(map[string]*StreamReader[string]{
    "sensor-a": sr1,
    "sensor-b": sr2,
})
// 任一 source EOF 时返回 *SourceEOF，可通过 antsx.GetSourceName(err) 获取来源名
```

#### 错误包装

```go
sr := antsx.StreamReaderWithConvert(
    source,
    convertFn,
    antsx.WithErrWrapper(func(err error) error {
        return fmt.Errorf("stage-xyz: %w", err)
    }),
)
```

---

### EventEmitter — 发布订阅

Topic 级别的 Pub/Sub。每个 subscriber 获得独立的 `StreamReader`。

```go
emitter := antsx.NewEventEmitter[string]()
defer emitter.Close()

// 订阅
sr, cancel := emitter.Subscribe(ctx, "sensor-data")
defer cancel()

// 消费（后台 goroutine）
go func() {
    for {
        val, err := sr.Recv()
        if errors.Is(err, io.EOF) { break }
        fmt.Println(val)
    }
}()

// 发布
emitter.Emit("sensor-data", "temperature:25.3")
emitter.Emit("sensor-data", "humidity:60")
```

| 方法 | 作用 |
|------|------|
| `Subscribe(ctx, topic, ...bufSize)` | 返回 `(*StreamReader[T], cancel)`，ctx 取消自动退订 |
| `Emit(topic, value)` | 广播给该 topic 所有订阅者 |
| `TopicCount()` | 活跃 topic 数 |
| `SubscriberCount(topic)` | 指定 topic 的订阅者数 |
| `Close()` | 关闭 emitter，所有订阅者收到 `io.EOF`（幂等） |

**典型用法：SSE 事件推送**

```go
// SSE 网关
emitter := antsx.NewEventEmitter[SSEEvent]()

// 客户端连接时订阅
sr, cancel := emitter.Subscribe(ctx, "notifications", 64)
defer cancel()

// 业务逻辑中推送事件
emitter.Emit("notifications", SSEEvent{Data: "hello"})
```

---

### Reactor — 协程池

基于 `ants.Pool` 的并发控制。所有需要限制并发的操作都可以通过 Reactor 限流。

```go
reactor, err := antsx.NewReactor(20) // 最多 20 个并发 goroutine
if err != nil {
    log.Fatal(err)
}
defer reactor.Release()

// 提交任务，返回 Promise
p, err := antsx.Submit(ctx, reactor, func(ctx context.Context) (string, error) {
    return heavyComputation(ctx)
})
val, err := p.Await(ctx)

// Fire-and-forget
err = antsx.Post(ctx, reactor, func(ctx context.Context) {
    backgroundCleanup(ctx)
})

// 方法形式
err = reactor.Go(ctx, func(ctx context.Context) {
    backgroundCleanup(ctx)
})
```

| 函数/方法 | 作用 |
|-----------|------|
| `NewReactor(size)` | 创建协程池 |
| `Submit(ctx, reactor, fn)` | 提交任务，返回 `(*Promise[T], error)` |
| `Post(ctx, reactor, fn)` | Fire-and-forget，自动 panic 恢复 |
| `reactor.Go(ctx, fn)` | 方法形式的 Post |
| `reactor.Release()` | 释放池资源 |
| `reactor.ActiveCount()` | 当前活跃 goroutine 数 |

---

### ReplyPool — 请求-响应关联

基于 correlation-ID 的异步请求-响应注册表。用于 MQTT、TCP 等协议的 RPC 模式：发送请求时注册一个 ID，收到响应时通过 ID 匹配并唤醒等待者。

```go
pool := antsx.NewReplyPool[string](
    antsx.WithDefaultTTL(30 * time.Second),
    antsx.WithName("mqtt-rpc"),
)
defer pool.Close()

// 方式一：手动注册
p, err := pool.Register("msg-123")
if err != nil {
    log.Fatal(err)
}

// 发送请求（实际网络操作）
err = sendMQTTRequest("msg-123", payload)

// 等待响应
response, err := p.Await(ctx)

// 收到响应时（在消息回调中）
pool.Resolve("msg-123", "response-payload")

// 方式二：一步完成（推荐）
response, err := antsx.RequestReply(ctx, pool, "msg-123", func() error {
    return sendMQTTRequest("msg-123", payload)
})
```

| 方法 | 作用 |
|------|------|
| `Register(id, ...ttl)` | 注册待响应请求，返回 `*Promise[T]` |
| `Resolve(id, val)` | 通过 ID 唤醒等待者，返回是否成功 |
| `Reject(id, err)` | 通过 ID 拒绝等待者 |
| `Has(id)` | 检查 ID 是否在等待 |
| `Len()` | 当前等待数 |
| `Close()` | 关闭池，所有待响应请求收到 `ErrReplyClosed` |

**哨兵错误：**
- `ErrReplyExpired` — TTL 过期
- `ErrDuplicateID` — 重复注册
- `ErrReplyClosed` — 池已关闭

**自动统计：** ReplyPool 每分钟通过 `logx.Statf` 输出等待数、过期数等指标。

---

### TeeWriter — 写入分发

`io.Writer` 接口的 fan-out 实现。一次写入同时发送到内部 pipe 和所有附加 writer。

```go
tw := antsx.NewTeeWriter(ossWriter, md5Hasher)

// 写入会同时到达 ossWriter 和 md5Hasher
io.Copy(tw, file)

// 从 pipe 读取（例如做 further processing）
data, _ := io.ReadAll(tw.Reader())
tw.Close()
```

| 方法 | 作用 |
|------|------|
| `NewTeeWriter(additionalWriters...)` | 创建，可选附加 writer |
| `Write(p)` | 同时写入 pipe + 所有附加 writer |
| `Reader()` | 返回 pipe reader |
| `Close()` | 关闭 pipe writer |
| `CloseWithError(err)` | 关闭并传递错误给 reader |

---

### UnboundedChan — 无界阻塞队列

无界 MPMC（多生产者多消费者）阻塞队列。使用 `sync.Mutex` + `sync.Cond` 实现，不受固定缓冲区限制。

```go
ch := antsx.NewUnboundedChan[int]()

// 生产者
go func() {
    for i := 0; i < 1000; i++ {
        ch.Send(i) // 永不阻塞（除非已 Close）
    }
    ch.Close()
}()

// 消费者（多个 goroutine 安全）
for i := 0; i < 3; i++ {
    go func() {
        for {
            val, ok := ch.ReceiveContext(ctx)
            if !ok { break } // 已关闭或 ctx 取消
            process(val)
        }
    }()
}
```

| 方法 | 作用 |
|------|------|
| `Send(val)` | 发送；已关闭时 panic（与 channel 语义一致） |
| `TrySend(val)` | 非阻塞发送，返回是否成功 |
| `Receive()` | 阻塞接收，关闭后仍可消费缓冲区 |
| `ReceiveContext(ctx)` | 带 ctx 取消的接收 |
| `Close()` | 关闭，唤醒所有接收者（幂等） |
| `Len()` | 当前缓冲区长度 |

---

## 项目中的使用场景

| 服务 | 使用的 antsx 原语 | 场景 |
|------|-------------------|------|
| **mcpx** | `Invoke` + `EventEmitter` + `Reactor` + `Promise` | MCP 工具并行调用编排，进度事件分发，协程池并发控制 |
| **file** | `TeeWriter` + `Reactor` + `InvokeAllSettled` | 文件流同时写入 OSS + 计算 MD5，多目标并行上传 |
| **aiapp/ssegtw** | `EventEmitter` + `ReplyPool` | SSE 事件流推送，流完成信号通知 |
| **aiapp/aichat** | `Promise` + `StreamReader` | AI 对话流式输出，异步流接收带超时控制 |
| **trigger** | `Invoke` + `InvokeAllSettled` + `Reactor` | 计划任务并行执行，fast-fail 失败快速取消，HTTP/gRPC 回调编排 |
| **mqttx** | `ReplyPool` + `RequestReply` | MQTT 请求-响应关联，异步 RPC 超时控制 |
| **iec104** | `ReplyPool` | 控制命令 ACK 匹配 |
| **gnetx** | `ReplyPool` + `Promise` | TCP session 请求-响应匹配 |

---

## 不适合 antsx 的场景

| 场景 | 推荐方案 |
|------|----------|
| 跨语言通信 | gRPC Stream |
| 持久化消息队列 | Kafka |
| 复杂背压策略 | antsx 的 `Pipe` 只有 channel 缓冲，无动态背压 |
| 完整响应式操作符生态 | antsx 只提供核心原语，不是框架 |
