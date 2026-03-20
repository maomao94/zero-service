# antsx - Go 响应式工具包

基于 ants goroutine 池的响应式编程工具包，参考 Java Project Reactor 理念，以 Go 惯用风格实现。

## 模块概览

| 文件 | 核心类型 | 职责 |
|------|---------|------|
| `promise.go` | `Promise[T]` | 泛型异步结果容器，支持链式调用 |
| `promise_ext.go` | `PromiseAll`, `PromiseRace`, `Map`, `FlatMap` | Promise 组合器 |
| `reactor.go` | `Reactor` | ants 池调度器，带 ID 去重 |
| `pending.go` | `PendingRegistry[T]` | 关联 ID 请求-响应匹配，自动过期 |
| `invoke.go` | `Task[T]`, `Invoke` | 并行流程编排，支持超时控制 |
| `emitter.go` | `EventEmitter[T]` | Topic 级别发布/订阅 |
| `errors.go` | `ErrPendingExpired`, `ErrDuplicateID`, `ErrRegistryClosed` | 哨兵错误 |

**并发安全**：所有类型均为并发安全设计，内置 panic recovery。
在 `Submit`、`Then`、`Map`、`FlatMap`、`Invoke` 等入口点中，用户函数 panic 会被捕获并转换为 error，不会导致 goroutine 泄漏或调用方永久阻塞。

---

## 1. Promise[T] - 异步结果容器

### 基本用法

```go
p := antsx.NewPromise[string]("req-1")

// 异步解决
go func() {
    result := doSomeWork()
    p.Resolve(result)
}()

// 等待结果（支持 context 超时）
val, err := p.Await(ctx)
```

### 链式调用

```go
// Then: T -> (U, error)
p2 := antsx.Then(ctx, p1, func(val int) (string, error) {
    return fmt.Sprintf("result: %d", val), nil
})

// Map: T -> U (纯映射，不返回 error)
p3 := antsx.Map(ctx, p1, func(val int) string {
    return strconv.Itoa(val)
})

// FlatMap: T -> *Promise[U]
p4 := antsx.FlatMap(ctx, p1, func(val int) *antsx.Promise[string] {
    return fetchFromRemote(val)
})
```

### 错误捕获

```go
p.Catch(func(err error) {
    log.Printf("任务失败: %v", err)
})
```

### 并发组合

```go
// 等待所有 Promise，任一失败立即返回
results, err := antsx.PromiseAll(ctx, p1, p2, p3)

// 竞争，返回最先完成的结果
val, err := antsx.PromiseRace(ctx, p1, p2)

// 带超时的 Await
val, err := p.AwaitWithTimeout(3 * time.Second)
```

---

## 2. Reactor - 池调度器

```go
// 创建 5 个 worker 的池
reactor, err := antsx.NewReactor(5)
if err != nil {
    log.Fatal(err)
}
defer reactor.Release()

// Submit: 带 ID 去重的任务提交，返回 Promise
promise, err := antsx.Submit(ctx, reactor, "task-1", func(ctx context.Context) (string, error) {
    return callRPC(ctx)
})
val, err := promise.Await(ctx)

// Post: fire-and-forget，只记录错误日志
antsx.Post(ctx, reactor, func(ctx context.Context) (any, error) {
    return nil, sendNotification(ctx)
})

// Go: 最轻量的池提交，无 Promise/ID 开销
reactor.Go(func() {
    cleanupTempFiles()
})
```

---

## 3. PendingRegistry[T] - 关联 ID 匹配

核心场景：发送异步请求后，通过关联 ID 匹配响应。

### 创建注册表

```go
// 默认 TTL 30 秒
reg := antsx.NewPendingRegistry[ResponseMsg]()

// 自定义默认 TTL
reg := antsx.NewPendingRegistry[ResponseMsg](
    antsx.WithDefaultTTL(10 * time.Second),
)
defer reg.Close()
```

### 场景一：MQ 消息匹配

```go
// --- 发送端 ---
correlationId := uuid.New().String()

// 注册等待，TTL 5 秒
promise, err := reg.Register(correlationId, 5*time.Second)
if err != nil {
    return err
}

// 发送消息
err = kafka.Publish("cmd-topic", Message{
    CorrelationId: correlationId,
    Payload:       cmdPayload,
})
if err != nil {
    reg.Reject(correlationId, err) // 发送失败，立即清理
    return err
}

// 等待响应
reply, err := promise.Await(ctx)
// err 可能是: nil(成功), ErrPendingExpired(超时), context.DeadlineExceeded(ctx超时)


// --- 消费端（另一个 goroutine）---
for msg := range consumer.Messages() {
    var resp ResponseMsg
    json.Unmarshal(msg.Value, &resp)
    reg.Resolve(resp.CorrelationId, resp) // 通过关联 ID 匹配
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

### 错误处理

```go
promise, err := reg.Register("id-1")
// err == ErrDuplicateID   -> ID 已被注册
// err == ErrRegistryClosed -> 注册表已关闭

val, err := promise.Await(ctx)
// err == ErrPendingExpired      -> TTL 超时自动过期
// err == ErrRegistryClosed      -> Close() 被调用
// err == context.DeadlineExceeded -> ctx 超时

ok := reg.Resolve("id-1", val)
// ok == false -> ID 不存在（已解决/已过期/已关闭）
```

---

## 4. Invoke - 并行流程编排

### 基本并行执行

```go
results, err := antsx.Invoke(ctx,
    antsx.Task[*UserInfo]{
        Name: "basic-info",
        Fn:   func(ctx context.Context) (*UserInfo, error) { return userRpc.GetBasic(ctx) },
    },
    antsx.Task[*UserInfo]{
        Name: "stats",
        Fn:   func(ctx context.Context) (*UserInfo, error) { return userRpc.GetStats(ctx) },
    },
    antsx.Task[*UserInfo]{
        Name: "prefs",
        Fn:   func(ctx context.Context) (*UserInfo, error) { return userRpc.GetPrefs(ctx) },
    },
)
// results[0] = basic-info, results[1] = stats, results[2] = prefs
// 结果按 index 排列，不是按完成顺序
// 任一任务失败 -> 立即取消其余任务，返回第一个错误
```

### 单任务超时 + 整体超时

```go
// 整体 10 秒超时
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

results, err := antsx.Invoke(ctx,
    antsx.Task[string]{
        Name:    "fast-api",
        Timeout: 2 * time.Second,  // 单任务 2 秒超时
        Fn:      callFastAPI,
    },
    antsx.Task[string]{
        Name:    "slow-api",
        Timeout: 5 * time.Second,  // 单任务 5 秒超时
        Fn:      callSlowAPI,
    },
)
```

### InvokeCallback - 执行后聚合变换

```go
profile, err := antsx.InvokeCallback(ctx,
    []antsx.Task[*UserInfo]{
        {Name: "basic", Fn: getBasic, Timeout: 3 * time.Second},
        {Name: "stats", Fn: getStats, Timeout: 5 * time.Second},
        {Name: "prefs", Fn: getPrefs, Timeout: 2 * time.Second},
    },
    func(infos []*UserInfo) (*AggregatedProfile, error) {
        return mergeProfiles(infos[0], infos[1], infos[2]), nil
    },
)
// 如果 Invoke 阶段失败，callback 不会被调用
```

### InvokeWithReactor - 池化执行

```go
reactor, _ := antsx.NewReactor(10)
defer reactor.Release()

// 任务通过 Reactor 池执行，而非裸 goroutine
results, err := antsx.InvokeWithReactor(ctx, reactor,
    antsx.Task[int]{Name: "t1", Fn: task1},
    antsx.Task[int]{Name: "t2", Fn: task2},
)
```

---

## 5. EventEmitter[T] - 发布/订阅

```go
emitter := antsx.NewEventEmitter[SSEEvent]()
defer emitter.Close()

// 订阅
ch, cancel := emitter.Subscribe("user-123", 32) // bufSize=32
defer cancel()

// 消费
go func() {
    for event := range ch {
        sendSSE(event)
    }
}()

// 发布（非阻塞，慢消费者丢弃）
emitter.Emit("user-123", SSEEvent{Event: "message", Data: "hello"})

// 查询
emitter.TopicCount()              // 活跃 topic 数
emitter.SubscriberCount("user-123") // 指定 topic 订阅者数
```

---

## 组合使用示例

### EventEmitter + PendingRegistry 实现 SSE 流式 AI 对话

```go
// 服务初始化
emitter := antsx.NewEventEmitter[SSEEvent]()
pendingReg := antsx.NewPendingRegistry[string](antsx.WithDefaultTTL(60 * time.Second))

// HTTP Handler: 客户端发起对话
func handleChat(ctx context.Context, req ChatRequest) {
    sessionId := uuid.New().String()

    // 1. 注册完成信号
    done, _ := pendingReg.Register(sessionId, 60*time.Second)

    // 2. 订阅流式事件
    ch, cancel := emitter.Subscribe(sessionId)
    defer cancel()

    // 3. 发送到后端（MQ/RPC）
    mq.Publish("ai-request", AIRequest{SessionId: sessionId, Prompt: req.Prompt})

    // 4. 流式推送给客户端
    // 用独立 goroutine 等待完成信号，触发 cancel 关闭 ch
    go func() {
        done.Await(ctx)
        cancel() // AI 生成完毕，关闭订阅 channel
    }()

    for event := range ch {
        sendSSE(event) // 推送中间 token
    }
}

// AI Worker: 处理推理结果
func handleAIResult(result AIResult) {
    for _, token := range result.Tokens {
        emitter.Emit(result.SessionId, SSEEvent{Event: "token", Data: token})
    }
    emitter.Emit(result.SessionId, SSEEvent{Event: "done", Data: ""})
    pendingReg.Resolve(result.SessionId, "completed")
}
```
