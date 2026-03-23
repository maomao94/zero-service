# antsx - Go 响应式工具包

基于 ants goroutine 池的响应式编程工具包，参考 Java Project Reactor 理念，以 Go 惯用风格实现。

**并发安全**：所有类型均为并发安全设计，内置 panic recovery。
用户函数 panic 会被捕获并转换为 error，不会导致 goroutine 泄漏或调用方永久阻塞。

---

## 场景选择指南

根据你的架构选择合适的工具：

| 场景 | 推荐工具 | 关键特征 |
|------|---------|---------|
| 阻塞调用加超时 | `Promise` | 数据直连流转，只需包装阻塞 I/O |
| 并行聚合多个调用 | `Invoke` | 多个独立调用并行执行，汇总结果 |
| 高并发任务+资源控制 | `Reactor` | goroutine 池化复用，ID 去重 |
| 异步请求-响应匹配 | `PendingRegistry` | 请求和响应走不同通道，用关联 ID 匹配 |
| 事件广播/多订阅者 | `EventEmitter` | 一对多推送，topic 级隔离 |
| MQ 异步流式推送 | `EventEmitter` + `PendingRegistry` | 解耦的生产者-消费者 + 完成信号 |

### 如何判断用哪个？

```
你的数据流是怎样的？

├── 直连同步流（调用方直接拿结果）
│   ├── 单个阻塞调用需要超时 ──────────> Promise
│   ├── 多个独立调用并行聚合 ──────────> Invoke
│   └── 高并发提交需要限流 ───────────> Reactor + Promise
│
└── 解耦异步流（请求和响应走不同通道）
    ├── 一次请求 - 一次响应 ──────────> PendingRegistry
    ├── 一次请求 - 多次推送 ──────────> EventEmitter
    └── 一次请求 - 多次推送 + 完成信号 ─> EventEmitter + PendingRegistry
```

---

## 1. Promise[T] - 异步结果容器

**适用场景**：将阻塞调用包装为可超时、可取消的异步操作。

### 典型场景：gRPC stream 读取加超时

```
上游 AI ──SSE──> reader.Recv() [阻塞] ──> gRPC stream.Send() ──> 客户端
                     │
                     └── 如果上游挂了，Recv() 永远阻塞
                         用 Promise 包装后可以设超时中断
```

```go
for {
    recv := antsx.NewPromise[*StreamChunk]("stream-recv")
    go func() {
        chunk, err := reader.Recv() // 阻塞 I/O
        if err != nil {
            recv.Reject(err)
        } else {
            recv.Resolve(chunk)
        }
    }()

    // 90s 内没收到 chunk 就超时返回
    idleCtx, cancel := context.WithTimeout(streamCtx, 90*time.Second)
    chunk, err := recv.Await(idleCtx)
    cancel()

    if err != nil {
        // context.DeadlineExceeded = 超时
        // io.EOF = 正常结束
        // 其他 = 上游错误
        return err
    }
    stream.Send(toProto(chunk))
}
```

### 基本用法

```go
p := antsx.NewPromise[string]("req-1")

go func() {
    result := doSomeWork()
    p.Resolve(result)
}()

val, err := p.Await(ctx)                    // 跟随 context 超时
val, err := p.AwaitWithTimeout(3 * time.Second) // 固定超时
```

### 链式调用

```go
// Then: T -> (U, error)
p2 := antsx.Then(ctx, p1, func(val int) (string, error) {
    return fmt.Sprintf("result: %d", val), nil
})

// Map: T -> U（纯映射）
p3 := antsx.Map(ctx, p1, func(val int) string {
    return strconv.Itoa(val)
})

// FlatMap: T -> *Promise[U]（链接多个异步操作）
p4 := antsx.FlatMap(ctx, p1, func(val int) *antsx.Promise[string] {
    return fetchFromRemote(val)
})
```

### 并发组合

```go
// 等待所有完成，任一失败立即取消其余
results, err := antsx.PromiseAll(ctx, p1, p2, p3)

// 竞争，返回最先完成的
val, err := antsx.PromiseRace(ctx, p1, p2)
```

### 错误捕获

```go
p.Catch(func(err error) {
    log.Printf("任务失败: %v", err)
})
```

---

## 2. Invoke - 并行流程编排

**适用场景**：多个独立的 RPC/IO 调用需要并行执行并汇总结果。

### 典型场景：聚合用户画像

```
        ┌── getBasic()  ──> 基础信息
请求 ───┼── getStats()  ──> 统计数据  ───> 合并为完整画像
        └── getPrefs()  ──> 偏好设置
```

```go
results, err := antsx.Invoke(ctx,
    antsx.Task[*UserInfo]{
        Name:    "basic-info",
        Timeout: 3 * time.Second,
        Fn:      func(ctx context.Context) (*UserInfo, error) { return userRpc.GetBasic(ctx) },
    },
    antsx.Task[*UserInfo]{
        Name:    "stats",
        Timeout: 5 * time.Second,
        Fn:      func(ctx context.Context) (*UserInfo, error) { return userRpc.GetStats(ctx) },
    },
)
// results 按 index 排列（不是完成顺序）
// 任一失败 -> 立即取消其余，返回第一个错误
```

### InvokeCallback - 执行后聚合变换

```go
profile, err := antsx.InvokeCallback(ctx,
    []antsx.Task[*UserInfo]{
        {Name: "basic", Fn: getBasic, Timeout: 3 * time.Second},
        {Name: "stats", Fn: getStats, Timeout: 5 * time.Second},
    },
    func(infos []*UserInfo) (*AggregatedProfile, error) {
        return mergeProfiles(infos[0], infos[1]), nil
    },
)
```

### InvokeWithReactor - 池化执行

当并发任务量大时，通过 Reactor 池限制 goroutine 数量：

```go
reactor, _ := antsx.NewReactor(10)
defer reactor.Release()

results, err := antsx.InvokeWithReactor(ctx, reactor,
    antsx.Task[int]{Name: "t1", Fn: task1},
    antsx.Task[int]{Name: "t2", Fn: task2},
)
```

---

## 3. Reactor - 池调度器

**适用场景**：高并发场景下需要控制 goroutine 数量，同时需要 ID 去重防止重复提交。

### 典型场景：消息处理限流

```
高并发请求 ──> Reactor (pool size=100) ──> 有序处理
                   │
                   └── ID 去重：同一请求不会重复执行
```

```go
reactor, err := antsx.NewReactor(100)
defer reactor.Release()

// Submit: 带 ID 去重，返回 Promise
promise, err := antsx.Submit(ctx, reactor, "order-123", func(ctx context.Context) (string, error) {
    return processOrder(ctx, "order-123")
})
// err == ErrDuplicateID 表示同 ID 任务正在执行
val, err := promise.Await(ctx)

// Post: fire-and-forget，错误仅日志记录
antsx.Post(ctx, reactor, func(ctx context.Context) (any, error) {
    return nil, sendNotification(ctx)
})

// Go: 最轻量的提交，无 Promise/ID 开销
reactor.Go(func() {
    cleanupTempFiles()
})
```

---

## 4. PendingRegistry[T] - 关联 ID 请求-响应匹配

**适用场景**：请求和响应走不同通道（MQ、TCP 等），需要通过关联 ID 将响应路由回对应的请求方。

### 典型流程

```
发送端                              消费端（另一个 goroutine）
  │                                    │
  ├── reg.Register(id) -> Promise      │
  ├── mq.Publish(请求)                 │
  ├── promise.Await(ctx) [等待]        │
  │                                    ├── 收到响应
  │                                    ├── reg.Resolve(id, 响应)
  │<── Promise 返回结果 ───────────────┘
```

### 场景一：MQ 消息匹配

```go
reg := antsx.NewPendingRegistry[ResponseMsg](
    antsx.WithDefaultTTL(10 * time.Second),
)
defer reg.Close()

// --- 发送端 ---
correlationId := uuid.New().String()
promise, err := reg.Register(correlationId, 5*time.Second)
if err != nil {
    return err // ErrDuplicateID 或 ErrRegistryClosed
}

err = kafka.Publish("cmd-topic", Message{
    CorrelationId: correlationId,
    Payload:       cmdPayload,
})
if err != nil {
    reg.Reject(correlationId, err) // 发送失败，立即清理
    return err
}

reply, err := promise.Await(ctx)
// err: nil(成功) / ErrPendingExpired(TTL超时) / context.DeadlineExceeded(ctx超时)


// --- 消费端（另一个 goroutine）---
for msg := range consumer.Messages() {
    var resp ResponseMsg
    json.Unmarshal(msg.Value, &resp)
    reg.Resolve(resp.CorrelationId, resp)
}
```

### 场景二：TCP sendNo 匹配

```go
// 发送端
sendNo := atomic.AddUint32(&seq, 1)
id := strconv.FormatUint(uint64(sendNo), 10)

promise, _ := reg.Register(id, 3*time.Second)
conn.WriteFrame(Frame{SendNo: sendNo, Data: payload})
reply, err := promise.Await(ctx)

// 接收端（read loop）
for {
    frame := conn.ReadFrame()
    id := strconv.FormatUint(uint64(frame.SendNo), 10)
    reg.Resolve(id, frame)
}
```

### RequestReply 便捷封装

一行代码完成 注册 -> 发送 -> 等待：

```go
reply, err := antsx.RequestReply(ctx, reg, correlationId, func() error {
    return kafka.Publish("cmd-topic", Message{
        CorrelationId: correlationId,
        Payload:       cmdPayload,
    })
})
// sendFn 失败会自动清理已注册的 entry
```

---

## 5. EventEmitter[T] - 发布/订阅

**适用场景**：一对多事件广播，按 topic 隔离。适合 SSE 推送、实时通知等场景。

### 典型流程

```
生产者                         消费者 A（SSE 连接 1）
  │                              │
  ├── emitter.Emit(topic, event) ├── ch, cancel := emitter.Subscribe(topic)
  │         │                    ├── for event := range ch { sendSSE(event) }
  │         ├──────────────────> │
  │         │                    消费者 B（SSE 连接 2）
  │         │                    │
  │         └──────────────────> ├── for event := range ch { sendSSE(event) }
```

```go
emitter := antsx.NewEventEmitter[SSEEvent]()
defer emitter.Close()

// 订阅（bufSize 防止慢消费者阻塞生产者）
ch, cancel := emitter.Subscribe("user-123", 32)
defer cancel()

go func() {
    for event := range ch {
        sendSSE(event)
    }
}()

// 发布（非阻塞，慢消费者消息丢弃）
emitter.Emit("user-123", SSEEvent{Event: "message", Data: "hello"})

// 查询
emitter.TopicCount()                  // 活跃 topic 数
emitter.SubscriberCount("user-123")   // 指定 topic 订阅者数
```

---

## 6. 组合模式：EventEmitter + PendingRegistry

**适用场景**：通过 MQ 解耦的异步流式推送。请求通过 MQ 发给后端 Worker，Worker 逐步产生事件推回前端，
最后用一个完成信号收尾。

### 与 Promise 直连模式的区别

```
Promise 直连模式（如 gRPC stream + 上游 SSE）：
  客户端 ──请求──> 网关 ──gRPC stream──> 服务 ──HTTP SSE──> 上游 AI
                   │<── chunk ──────────<── chunk ──────<── chunk
                   │
                   └── 数据在同一条链路直接流转，Promise 只是给阻塞的 Recv() 加超时

EventEmitter + PendingRegistry 模式（如 MQ 异步流式）：
  客户端 ──请求──> HTTP Handler ──MQ──> AI Worker（独立进程）
                   │                        │
                   │<── EventEmitter ────────┤ 逐 token 推送
                   │                        │
                   │<── PendingRegistry ────┘ 完成信号
                   │
                   └── 请求和响应走不同通道，需要 EventEmitter 桥接事件 + PendingRegistry 匹配完成
```

### 完整示例

```go
// ========== 服务初始化 ==========
emitter := antsx.NewEventEmitter[SSEEvent]()
pendingReg := antsx.NewPendingRegistry[string](antsx.WithDefaultTTL(60 * time.Second))

// ========== HTTP Handler: 客户端发起对话 ==========
func handleChat(ctx context.Context, req ChatRequest) {
    sessionId := uuid.New().String()

    // 1. 注册完成信号（60s TTL 兜底，防止 Worker 挂掉后连接永不关闭）
    done, _ := pendingReg.Register(sessionId, 60*time.Second)

    // 2. 订阅该 session 的流式事件
    ch, cancel := emitter.Subscribe(sessionId)
    defer cancel()

    // 3. 通过 MQ 发送给后端 AI Worker
    mq.Publish("ai-request", AIRequest{SessionId: sessionId, Prompt: req.Prompt})

    // 4. 等待完成信号，触发 cancel 关闭 ch
    go func() {
        done.Await(ctx)
        cancel()
    }()

    // 5. 持续推送事件给客户端，直到 ch 关闭
    for event := range ch {
        sendSSE(event)
    }
}

// ========== AI Worker: 处理推理结果（独立进程/goroutine） ==========
func handleAIResult(result AIResult) {
    for _, token := range result.Tokens {
        emitter.Emit(result.SessionId, SSEEvent{Event: "token", Data: token})
    }
    emitter.Emit(result.SessionId, SSEEvent{Event: "done", Data: ""})
    pendingReg.Resolve(result.SessionId, "completed") // 触发完成信号
}
```

---

## 模块速查

| 文件 | 核心类型 | 职责 |
|------|---------|------|
| `promise.go` | `Promise[T]` | 泛型异步结果容器，支持链式调用 |
| `promise_ext.go` | `PromiseAll`, `PromiseRace`, `Then`, `Map`, `FlatMap` | Promise 组合器和转换器 |
| `reactor.go` | `Reactor` | ants 池调度器，带 ID 去重 |
| `pending.go` | `PendingRegistry[T]` | 关联 ID 请求-响应匹配，自动过期 |
| `invoke.go` | `Task[T]`, `Invoke` | 并行流程编排，支持超时控制 |
| `emitter.go` | `EventEmitter[T]` | Topic 级别发布/订阅 |
| `errors.go` | `ErrPendingExpired`, `ErrDuplicateID`, `ErrRegistryClosed` | 哨兵错误 |
