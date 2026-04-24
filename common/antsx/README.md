# antsx

基于 ants 协程池的响应式编程工具包，提供流式处理、异步 Promise、并行编排、发布订阅等能力。

流式原语 API 设计参考字节跳动 [eino](https://github.com/cloudwego/eino) 框架，保持一致的 `Pipe / StreamReader / StreamWriter` 使用范式。

## 模块速查

| 模块 | 文件 | 核心能力 |
|------|------|---------|
| **Stream** | `stream.go` `select.go` | 流式管道：Pipe、Copy、Merge、Convert、FromArray |
| **Promise** | `promise.go` | 异步结果容器：Await、Then、Map、FlatMap、All、Race |
| **Reactor** | `reactor.go` | 协程池调度：Submit（带返回值）、Post（无返回值）、Go |
| **Invoke** | `invoke.go` | 并行流程编排：多任务并发执行、快速失败、超时控制 |
| **EventEmitter** | `emitter.go` | 发布/订阅：多 topic、ctx 取消自动退订 |
| **PendingRegistry** | `pending.go` | 关联 ID 注册表：请求-响应模式、TTL 自动过期 |
| **UnboundedChan** | `unbounded.go` | 无界通道：基于 mutex+cond 的无限缓冲 channel |

## 快速使用

### Stream（流式管道）

创建生产者-消费者管道：

```go
sr, sw := antsx.Pipe[string](10) // 缓冲区容量 10
go func() {
    defer sw.Close() // 必须调用，通知消费者 EOF
    sw.Send("hello", nil)
    sw.Send("world", nil)
}()

defer sr.Close() // 必须调用，即使已读到 EOF
for {
    val, err := sr.Recv()
    if errors.Is(err, io.EOF) {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(val)
}
```

流操作：

```go
// 多路合并（fan-in），按到达顺序交错输出
merged := antsx.MergeStreamReaders([]*antsx.StreamReader[int]{sr1, sr2, sr3})
defer merged.Close()

// 广播复制（fan-out），原始 reader 调用后不可再用
copies := sr.Copy(3)
// copies[0], copies[1], copies[2] 各自独立消费完整数据

// 类型转换（带过滤），返回 ErrNoValue 可跳过当前元素
converted := antsx.StreamReaderWithConvert(sr, func(i int) (string, error) {
    if i == 0 {
        return "", antsx.ErrNoValue // 跳过零值
    }
    return fmt.Sprintf("v%d", i), nil
})

// 具名流合并，可追踪每条源流的结束
merged := antsx.MergeNamedStreamReaders(map[string]*antsx.StreamReader[string]{
    "agent_a": srA,
    "agent_b": srB,
})
for {
    chunk, err := merged.Recv()
    if errors.Is(err, io.EOF) { break }
    if name, ok := antsx.GetSourceName(err); ok {
        fmt.Printf("%s finished\n", name)
        continue
    }
    process(chunk)
}

// 数组转流（零开销，无 goroutine）
sr := antsx.StreamReaderFromArray([]int{1, 2, 3})
```

### Promise（异步结果容器）

```go
// 创建并等待
p := antsx.NewPromise[int]()
go func() { p.Resolve(42) }()
val, err := p.Await(ctx) // 阻塞直到 Resolve/Reject 或 ctx 取消

// 带超时的等待
val, err := p.AwaitWithTimeout(5 * time.Second)

// 链式变换
result := antsx.Then(ctx, promise, func(v int) (string, error) {
    return fmt.Sprintf("got %d", v), nil
})

// 映射
mapped := antsx.Map(ctx, promise, func(v int) string {
    return strconv.Itoa(v)
})

// 扁平映射
flat := antsx.FlatMap(ctx, promise, func(v int) *antsx.Promise[string] {
    return antsx.Go(ctx, func(ctx context.Context) (string, error) {
        return fetchName(ctx, v)
    })
})

// 并发等待全部完成（fast-fail，任一失败立即返回错误）
results, err := antsx.PromiseAll(ctx, p1, p2, p3)

// 竞速取最快完成的
val, err := antsx.PromiseRace(ctx, p1, p2)

// 后台执行，返回 Promise
p := antsx.Go(ctx, func(ctx context.Context) (int, error) {
    return compute(ctx)
})
```

### Reactor（协程池调度）

```go
r, _ := antsx.NewReactor(100) // 创建 100 worker 的协程池
defer r.Release()

// Submit：带返回值，返回 Promise
p, _ := antsx.Submit(ctx, r, func(ctx context.Context) (int, error) {
    return compute(), nil
})
val, _ := p.Await(ctx)

// Post：无返回值，内置 panic 保护和日志记录
antsx.Post(ctx, r, func(ctx context.Context) error {
    doWork(ctx)
    return nil
})

// Go：直接提交函数到协程池，内置 panic 保护和日志记录
r.Go(ctx, func(ctx context.Context) { doSomething(ctx) })

// 查询活跃 worker 数
count := r.ActiveCount()
```

### Invoke（并行流程编排）

```go
// 并行执行多个任务，fast-fail（任一失败立即取消其他任务）
// 结果按任务定义顺序排列（非完成顺序）
results, err := antsx.Invoke(ctx,
    antsx.Task[int]{Name: "db", Fn: queryDB},
    antsx.Task[int]{Name: "cache", Timeout: 500*time.Millisecond, Fn: queryCache},
    antsx.Task[int]{Name: "api", Fn: callAPI},
)

// 使用指定 Reactor 协程池执行（受限并发数）
results, err := antsx.InvokeWithReactor(ctx, reactor, tasks...)

// 执行后回调聚合
sum, err := antsx.InvokeCallback(ctx, tasks, func(vals []int) (int, error) {
    total := 0
    for _, v := range vals { total += v }
    return total, nil
})
```

### EventEmitter（发布/订阅）

```go
emitter := antsx.NewEventEmitter[string]()
defer emitter.Close()

// 订阅 topic（ctx 取消时自动退订）
sr, cancel := emitter.Subscribe(ctx, "chat-room")
defer cancel()

// 自定义缓冲区大小
sr2, cancel2 := emitter.Subscribe(ctx, "logs", 100)
defer cancel2()

// 发布事件（广播给所有订阅者）
emitter.Emit("chat-room", "hello!")

// 接收事件
val, err := sr.Recv()

// 查询状态
emitter.TopicCount()            // 活跃 topic 数
emitter.SubscriberCount("chat") // 指定 topic 的订阅者数
```

### PendingRegistry（请求-响应模式）

```go
reg := antsx.NewPendingRegistry[Response](
    antsx.WithDefaultTTL(30 * time.Second),
    antsx.WithTimingWheel(100*time.Millisecond, 16), // 自定义时间轮参数
)
defer reg.Close()

// 请求-响应一体化（注册 + 发送 + 等待）
resp, err := antsx.RequestReply(ctx, reg, requestID, func() error {
    return conn.Send(request)
})

// 或分步操作
promise, _ := reg.Register(requestID, 5*time.Second) // 可覆盖默认 TTL
conn.Send(request)
// ... 远端回调 ...
reg.Resolve(requestID, response)
resp, _ := promise.Await(ctx)

// 查询状态
reg.Has(requestID) // 是否有待处理请求
reg.Len()          // 待处理总数
```

### UnboundedChan（无界通道）

```go
ch := antsx.NewUnboundedChan[Task]()
defer ch.Close()

// 生产（已关闭时 panic，类似内置 channel 行为）
ch.Send(task)

// 安全生产（已关闭时返回 false）
ok := ch.TrySend(task)

// 阻塞消费（通道关闭且清空后返回 zero, false）
val, ok := ch.Receive()

// 带 ctx 超时消费
val, ok := ch.ReceiveContext(ctx)

// 队列长度
ch.Len()
```

## 设计细节

### 流的内部架构

`StreamReader` 使用类型标签 + 联合体模式（而非接口多态），通过 switch-case 分派到具体实现，
避免接口虚表调用开销：

- `readerStream`: 基于 channel 的基本流
- `readerArray`: 基于数组的同步流（零 goroutine 开销）
- `readerMulti`: 多路合并流
- `readerConvert`: 类型转换/过滤流
- `readerChild`: Copy 产生的子流

### 静态 select 优化

多路合并（`MergeStreamReaders`）在源流数量 ≤5 时使用编译时展开的 `select` 语句，
避免 `reflect.Select` 的反射开销。源流数量 >5 时降级为 `reflect.Select`。

### Copy 零拷贝广播

`Copy` 使用链表 + `sync.Once` 实现零拷贝的 fan-out：
- 第一个到达的子流触发实际的 `Recv()` 读取
- 其他子流直接读取已填充的数据
- `atomic.AddUint32` 引用计数确保所有子流关闭后自动关闭源流

### UnboundedChan 内存管理

底层使用切片作为环形缓冲区，当已消费偏移量超过容量一半时自动 compact：
- 重新分配切片，释放已消费元素
- 已消费位置显式清零，防止指针类型元素被底层数组持有导致 GC 无法回收

## 协程安全性

| 类型 | 安全性 |
|------|--------|
| `Promise` | 所有方法可并发调用。Resolve/Reject 通过 `sync.Once` 保证幂等 |
| `StreamWriter` | `Send` 和 `Close` 均可安全并发调用，`Close` 通过 atomic CAS 幂等 |
| `StreamReader` | `Recv` **非**并发安全（单消费者模型），`Close` 可从任意 goroutine 调用 |
| `EventEmitter` | 所有方法可并发调用。内部使用 `sync.RWMutex` 保护订阅者列表 |
| `PendingRegistry` | 所有方法可并发调用。内部使用 `sync.Mutex` 保护注册表 |
| `UnboundedChan` | 所有方法可并发调用。支持 MPMC（多生产者多消费者） |
| `Reactor` | `Submit/Post/Go` 可并发调用。`Release` 后调用其他方法返回错误 |

## 资源释放指南

**必须调用的 Close/Release**：

- `StreamWriter.Close()` — 不调用会导致消费者永远阻塞在 `Recv`
- `StreamReader.Close()` — 不调用会导致生产者 goroutine 泄漏（Send 阻塞）
- `EventEmitter.Close()` — 不调用会导致所有订阅者 goroutine 泄漏
- `PendingRegistry.Close()` — 不调用会导致内部 TimingWheel 协程泄漏
- `Reactor.Release()` — 不调用会导致协程池泄漏
- `UnboundedChan.Close()` — 不调用会导致等待中的消费者永远阻塞

**防御性机制**：

- `StreamReader.SetAutomaticClose()` 注册 GC Finalizer，在 reader 不可达时自动关闭
- 所有 Close 方法均为幂等操作，多次调用安全

## 错误类型参考

| 错误 | 来源 | 含义 |
|------|------|------|
| `io.EOF` | `StreamReader.Recv` | 流数据已全部读取完毕 |
| `ErrNoValue` | `StreamReaderWithConvert` | convert 函数跳过当前元素（过滤语义） |
| `ErrRecvAfterClosed` | `StreamReader.Recv` (Copy) | 在已关闭的子流上调用 Recv |
| `SourceEOF` | `MergeNamedStreamReaders` | 某条具名源流结束（其他源流可能仍在产出） |
| `ErrDuplicateID` | `PendingRegistry.Register` | 注册了重复的关联 ID |
| `ErrPendingExpired` | `PendingRegistry` | 条目超过 TTL 自动过期 |
| `ErrRegistryClosed` | `PendingRegistry` | 注册表已关闭 |
| `ErrChanClosed` | `UnboundedChan.TrySend` | 向已关闭的通道发送（TrySend 返回 false） |

## 最佳实践

1. **始终 `defer sr.Close()` 和 `defer sw.Close()`** — 即使已经读到 EOF
2. **检查 `Send` 返回值** — `closed == true` 表示消费者已关闭，应停止生产
3. **Copy 必须在 Recv 之前调用** — 原始 reader 在 Copy 后不可用
4. **单个 StreamReader 只由一个 goroutine 消费** — 需要多消费者时先 Copy
5. **使用 `errors.Is(err, io.EOF)` 判断流结束** — 而非 `err == io.EOF`
6. **ErrNoValue 仅在 convert 函数中使用** — 不要在其他场景返回该错误
7. **Invoke 结果按定义顺序排列** — 不是按完成时间排列
8. **Promise 的 Resolve/Reject 只有首次调用生效** — 后续调用被忽略

## 实战示例

### 流式管道：Copy + 并行处理 + Merge

将一个流 Copy 为多份，分别处理后 Merge 合并结果：

```go
func processStream(ctx context.Context) error {
    sr, sw := antsx.Pipe[string](10)
    go func() {
        defer sw.Close()
        for _, msg := range messages {
            sw.Send(msg, nil)
        }
    }()

    copies := sr.Copy(2)

    translated := antsx.StreamReaderWithConvert(copies[0], func(s string) (string, error) {
        return translate(s), nil
    })

    summarized := antsx.StreamReaderWithConvert(copies[1], func(s string) (string, error) {
        return summarize(s), nil
    })

    merged := antsx.MergeStreamReaders([]*antsx.StreamReader[string]{translated, summarized})
    defer merged.Close()

    for {
        chunk, err := merged.Recv()
        if errors.Is(err, io.EOF) { break }
        if err != nil { return err }
        fmt.Println(chunk)
    }
    return nil
}
```

### Promise 流水线

Go + Then + Map + FlatMap 链式调用：

```go
ctx := context.Background()

result := antsx.Go(ctx, func(ctx context.Context) (int, error) {
    return fetchUserID(ctx)
})

name := antsx.Then(ctx, result, func(id int) (string, error) {
    return fetchUserName(ctx, id)
})

greeting := antsx.Map(ctx, name, func(n string) string {
    return "Hello, " + n + "!"
})

val, err := greeting.Await(ctx)
```

### Invoke 错误处理

带错误处理和超时的 Invoke 示例：

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

results, err := antsx.Invoke(ctx,
    antsx.Task[string]{
        Name:    "user-service",
        Timeout: 2 * time.Second,
        Fn: func(ctx context.Context) (string, error) {
            return userClient.GetProfile(ctx, userID)
        },
    },
    antsx.Task[string]{
        Name:    "order-service",
        Timeout: 3 * time.Second,
        Fn: func(ctx context.Context) (string, error) {
            return orderClient.GetRecent(ctx, userID)
        },
    },
)
if err != nil {
    log.Printf("聚合查询失败: %v", err)
    return
}
profile, orders := results[0], results[1]
```

### EventEmitter 多 topic 并发

多 topic 生产消费示例：

```go
emitter := antsx.NewEventEmitter[Event]()
defer emitter.Close()

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

chatSr, chatCancel := emitter.Subscribe(ctx, "chat", 50)
defer chatCancel()
logSr, logCancel := emitter.Subscribe(ctx, "system-log", 100)
defer logCancel()

go func() {
    defer chatSr.Close()
    for {
        event, err := chatSr.Recv()
        if errors.Is(err, io.EOF) { break }
        handleChatEvent(event)
    }
}()

go func() {
    defer logSr.Close()
    for {
        event, err := logSr.Recv()
        if errors.Is(err, io.EOF) { break }
        writeLog(event)
    }
}()

emitter.Emit("chat", Event{Type: "message", Data: "hello"})
emitter.Emit("system-log", Event{Type: "info", Data: "user joined"})
```

### PendingRegistry WebSocket 请求-响应

WebSocket 场景的请求-响应关联：

```go
reg := antsx.NewPendingRegistry[[]byte](
    antsx.WithDefaultTTL(30 * time.Second),
)
defer reg.Close()

go func() {
    for {
        msg := conn.ReadMessage()
        reg.Resolve(msg.RequestID, msg.Payload)
    }
}()

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

resp, err := antsx.RequestReply(ctx, reg, "req-123", func() error {
    return conn.WriteMessage(Request{ID: "req-123", Action: "getUser"})
})
if errors.Is(err, antsx.ErrPendingExpired) {
    log.Println("请求超时")
}
```

### UnboundedChan 工作池

生产者-消费者池示例：

```go
ch := antsx.NewUnboundedChan[func()]()

var wg sync.WaitGroup
for i := 0; i < runtime.NumCPU(); i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        for {
            task, ok := ch.Receive()
            if !ok { return }
            task()
        }
    }()
}

for _, item := range items {
    item := item
    ch.Send(func() { process(item) })
}

ch.Close()
wg.Wait()
```

### Reactor + Invoke 组合使用

受限并发的批量任务执行：

```go
reactor, _ := antsx.NewReactor(10)
defer reactor.Release()

ctx := context.Background()

tasks := make([]antsx.Task[*Result], len(urls))
for i, url := range urls {
    url := url
    tasks[i] = antsx.Task[*Result]{
        Name:    url,
        Timeout: 5 * time.Second,
        Fn: func(ctx context.Context) (*Result, error) {
            return httpGet(ctx, url)
        },
    }
}

results, err := antsx.InvokeWithReactor(ctx, reactor, tasks...)
if err != nil {
    log.Fatal(err)
}
for i, r := range results {
    fmt.Printf("%s -> %s\n", urls[i], r.Status)
}
```
