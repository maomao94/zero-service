# antsx

`antsx` 是项目内的 Go 泛型异步编排工具包，面向流式数据、并行任务、请求-响应关联和进程内发布订阅。

底层组合 Go channel、context、go-zero `TimingWheel`、`threading.GoSafe` 和 [ants](https://github.com/panjf2000/ants) 协程池。Stream API 参考字节跳动 [eino](https://github.com/cloudwego/eino) 的 `Pipe / StreamReader / StreamWriter` 范式。

## 模块速查

| 模块 | 文件 | 核心 API | 用途 |
| --- | --- | --- | --- |
| Stream | `stream.go` `select.go` | `Pipe`、`Recv`、`Send`、`Copy`、`MergeStreamReaders`、`StreamReaderWithConvert` | 流式输出、fan-in/fan-out、边读边转 |
| TeeWriter | `tee.go` | `NewTeeWriter`、`Reader`、`CloseWithError` | `io.Reader` 上传链路扇出、边上传边计算 hash |
| Promise | `promise.go` | `Await`、`Done`、`Get`、`Catch`、`Then`、`Map`、`FlatMap`、`PromiseAll/Race/Any` | 单个异步结果和 Promise 组合 |
| Invoke | `invoke.go` | `Task`、`Invoke`、`InvokeAllSettled`、Reactor 变体 | 一组同类型函数的并行编排 |
| Reactor | `reactor.go` | `NewReactor`、`Submit`、`Post`、`Go`、`Release` | 通过 ants 池限制并发 goroutine 数 |
| EventEmitter | `emitter.go` | `Subscribe`、`Emit`、`Close` | topic 发布订阅，订阅端是 `StreamReader` |
| ReplyPool | `replypool.go` | `Register`、`Resolve`、`Reject`、`RequestReply`、`WithName` | correlation ID 请求-响应匹配 |
| UnboundedChan | `unbounded.go` | `Send`、`TrySend`、`Receive`、`ReceiveContext` | MPMC 无界阻塞队列 |

## 先选型

### Promise 和 Invoke

| 需求 | 推荐 API |
| --- | --- |
| 等已有 Promise 全部成功，一个失败就返回 | `PromiseAll` |
| 等已有 Promise 全部结束，失败项也保留 | `PromiseAllSettled` |
| 已有 Promise 谁先完成就用谁，成功失败都算完成 | `PromiseRace` |
| 已有 Promise 谁先成功就用谁，全部失败才报错 | `PromiseAny` |
| 现场定义一组 `Task[T]` 并行执行 | `Invoke` |
| 现场定义一组 `Task[T]`，需要全部结果，失败互不影响 | `InvokeAllSettled` |
| 需要限制并发度 | `InvokeWithReactor` / `InvokeAllSettledWithReactor` |
| 单个后台任务有返回值 | `Go` 或 `Submit` |
| 单个后台任务无返回值 | `Post` 或 `Reactor.Go` |

`Promise*` 适合组合已经存在的异步结果；`Invoke*` 适合把一组函数临时编排成并行任务。`Invoke` 和 `PromiseAll` 的结果都按定义顺序返回，不按完成顺序返回。

### ReplyPool

| 需求 | 推荐 API |
| --- | --- |
| 注册 ID、发送请求、等待响应一体化 | `RequestReply` |
| 注册、发送、等待需要拆开 | `Register` + `Resolve` / `Reject` |
| 区分多个池子的统计日志 | `WithName` |
| 调整默认过期时间 | `WithDefaultTTL` |
| 单次请求覆盖 TTL | `Register(id, ttl)` 或 `RequestReply(..., ttl)` |

`ReplyPool` 已重构为自动化配置：时间轮参数根据 TTL 自动推导；统计循环构造后自动启动、`Close()` 时自动停止。不要再使用旧的 `WithTimingWheel`、`StartStatsLoop`、`WithStatsLoop` 或 `Stats()` 思路。

## Stream

### 基本管道

```go
sr, sw := antsx.Pipe[string](10)
defer sr.Close()

go func() {
    defer sw.Close()
    if closed := sw.Send("hello", nil); closed {
        return
    }
    sw.Send("world", nil)
}()

for {
    val, err := sr.Recv()
    if errors.Is(err, io.EOF) {
        break
    }
    if err != nil {
        return err
    }
    fmt.Println(val)
}
```

规则：

- 写端必须 `Close()`，否则读端不会收到 EOF。
- 读端退出时也要 `Close()`，否则写端可能阻塞在 `Send`。
- `Send` 返回 `closed=true` 表示读端已关闭，生产者应停止发送。
- 单个 `StreamReader` 是单消费者模型，不要多个 goroutine 并发 `Recv`。

### 转换、过滤、合并

```go
converted := antsx.StreamReaderWithConvert(sr, func(i int) (string, error) {
    if i == 0 {
        return "", antsx.ErrNoValue
    }
    return fmt.Sprintf("v%d", i), nil
})

merged := antsx.MergeStreamReaders([]*antsx.StreamReader[string]{a, b, c})
if merged != nil {
    defer merged.Close()
}
```

`ErrNoValue` 只用于 `StreamReaderWithConvert` 的过滤语义。上游错误可通过 `WithErrWrapper` 包装；包装函数返回 nil 时会跳过该错误并继续读取。

### Copy 和具名合并

```go
copies := sr.Copy(2)

named := antsx.MergeNamedStreamReaders(map[string]*antsx.StreamReader[string]{
    "left":  copies[0],
    "right": copies[1],
})
defer named.Close()

for {
    chunk, err := named.Recv()
    if errors.Is(err, io.EOF) {
        break
    }
    if name, ok := antsx.GetSourceName(err); ok {
        fmt.Printf("%s finished\n", name)
        continue
    }
    if err != nil {
        return err
    }
    process(chunk)
}
```

`Copy(n)` 必须在原始 reader 被消费前调用。`n < 2` 时直接返回原始 reader；数组流复制独立游标，普通流用链表和 `sync.Once` 做零拷贝广播。

## TeeWriter

```go
hash := md5.New()
tee := antsx.NewTeeWriter(hash)
defer tee.Close()

go func() {
    if err := uploadOSS(ctx, tee.Reader()); err != nil {
        tee.CloseWithError(err)
    }
}()

if _, err := io.Copy(tee, sourceReader); err != nil {
    return err
}
```

`TeeWriter` 的写入会同时进入内部 `io.PipeWriter` 和所有 additional writers。写入顺序是内部 pipe 优先，然后是附加 writer；任一 writer 返回错误时，本次 `Write` 立即返回错误。`Close()` 只关闭内部 pipe 写端，不会自动关闭传入的 hash、文件等附加 writer。

## Promise

```go
p := antsx.NewPromise[int]()
go func() { p.Resolve(42) }()

val, err := p.Await(ctx)
```

常用方法：

- `Resolve` / `Reject` 只有第一次调用生效。
- `Await(ctx)` 阻塞等待完成；ctx 先取消则返回 `ctx.Err()`。
- `Done()` 返回完成信号 channel，适合 `select`。
- `Get()` 非阻塞读取当前状态，返回 `(val, err, ok)`。
- `Catch(ctx, fn)` 在 Promise 失败时异步执行错误回调。
- `AwaitWithTimeout(timeout)` 是带超时的快捷等待。
- `FireAndForget(ctx)` 后台等待并忽略结果，适合确保关联资源被释放。

链式处理：

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

userID := antsx.Go(ctx, func(ctx context.Context) (int64, error) {
    return fetchUserID(ctx)
})

name := antsx.Then(ctx, userID, func(id int64) (string, error) {
    return fetchUserName(ctx, id)
})

greeting := antsx.Map(ctx, name, func(v string) string {
    return "hello " + v
})

val, err := greeting.Await(ctx)
```

`Then`、`Map`、`FlatMap`、`Go` 会启动 goroutine。不要给永不完成的 Promise 配 `context.Background()`，业务代码优先传入带超时或可取消的 ctx。

组合语义：

```go
results, err := antsx.PromiseAll(ctx, p1, p2, p3)

settled := antsx.PromiseAllSettled(ctx, p1, p2, p3)
for _, r := range settled {
    if r.Succeeded() {
        use(r.Val)
    } else {
        log.Printf("failed: %v", r.Err)
    }
}

fastest, err := antsx.PromiseRace(ctx, p1, p2)
firstOK, err := antsx.PromiseAny(ctx, p1, p2, p3)
```

空输入时，`PromiseAll` 返回空切片；`PromiseRace` 和 `PromiseAny` 返回 `ErrEmptyPromises`。

## Invoke

```go
tasks := []antsx.Task[*Profile]{
    {
        Name: "db",
        Fn: func(ctx context.Context) (*Profile, error) {
            return queryDB(ctx, userID)
        },
    },
    {
        Name:    "cache",
        Timeout: 500 * time.Millisecond,
        Fn: func(ctx context.Context) (*Profile, error) {
            return queryCache(ctx, userID)
        },
    },
}

profiles, err := antsx.Invoke(ctx, tasks...)
```

`Invoke` 任一任务失败会 cancel 其他任务，并用 `errors.Join` 聚合错误。`Task.Fn` 必须主动检查 `ctx.Done()`，否则取消只能发出信号，不能强制终止阻塞调用。

需要容错降级时用 `InvokeAllSettled`：

```go
for _, r := range antsx.InvokeAllSettled(ctx, tasks...) {
    if r.Succeeded() {
        use(r.Name, r.Val)
    } else {
        log.Printf("task=%s err=%v", r.Name, r.Err)
    }
}
```

`Task.Name` 应填写稳定名称，panic 错误和 `SettledResult` 都靠它定位任务。

## Reactor

```go
reactor, err := antsx.NewReactor(100)
if err != nil {
    return err
}
defer reactor.Release()

p, err := antsx.Submit(ctx, reactor, func(ctx context.Context) (int, error) {
    return compute(ctx)
})
if err != nil {
    return err
}
val, err := p.Await(ctx)

err = antsx.Post(ctx, reactor, func(ctx context.Context) {
    doWork(ctx)
})

err = reactor.Go(ctx, func(ctx context.Context) {
    doMore(ctx)
})
```

`Submit` 用于有返回值的任务；`Post` 和 `Reactor.Go` 用于 fire-and-forget。nil Reactor 会返回错误，不会 panic。`Release()` 后不要再提交任务。

## EventEmitter

```go
emitter := antsx.NewEventEmitter[string]()
defer emitter.Close()

ctx, cancelCtx := context.WithCancel(context.Background())
defer cancelCtx()

sr, cancelSub := emitter.Subscribe(ctx, "chat", 64)
defer cancelSub()
defer sr.Close()

emitter.Emit("chat", "hello")
val, err := sr.Recv()
```

`Subscribe` 返回的 cancel 函数会移除订阅并关闭 reader；ctx 取消时会自动调用 cancel。`Emit` 对无订阅者或已关闭 emitter 静默忽略。

## ReplyPool

### 一体化请求-响应

```go
pool := antsx.NewReplyPool[Response](
    antsx.WithName("drc-reply"),
    antsx.WithDefaultTTL(30*time.Second),
)
defer pool.Close()

resp, err := antsx.RequestReply(ctx, pool, requestID, func() error {
    return conn.Send(request)
}, 10*time.Second)
```

`RequestReply` 先注册 ID，再执行 `sendFn`，最后等待 Promise。如果 `sendFn` 失败，会自动 `Reject` 并清理 entry。

### 分步注册和回调响应

```go
promise, err := pool.Register(requestID, 5*time.Second)
if err != nil {
    return err
}

if err := conn.Send(request); err != nil {
    pool.Reject(requestID, err)
    return err
}

// 异步回包 goroutine 中：
pool.Resolve(requestID, response)

resp, err := promise.Await(ctx)
```

重要语义：caller 的 `ctx` 只控制等待，不控制 entry 生命周期。`Await(ctx)` 超时后不要主动 `Reject` 清理 entry；异步任务可能稍后 `Resolve`。如果远端始终不响应，时间轮会自动触发 `ErrReplyExpired`。

内置行为：

- TTL 自动推导时间轮精度，无需手动配置 interval/slots。
- 构造后自动启动每分钟统计日志，格式为 `<name>[1m] - registered: ...`。
- `WithName` 只影响日志前缀，默认 `reply-pool`。
- `Close()` 会拒绝所有待处理 entry，并停止统计循环和时间轮。

## UnboundedChan

```go
ch := antsx.NewUnboundedChan[func()]()
defer ch.Close()

ch.Send(func() { doWork() })

if ok := ch.TrySend(func() { doMore() }); !ok {
    return antsx.ErrChanClosed
}

task, ok := ch.ReceiveContext(ctx)
if ok {
    task()
}
```

`Send` 在通道关闭后 panic，语义对齐 Go 原生 channel；需要安全发送时用 `TrySend`。`Receive` 会一直阻塞到有数据或关闭；需要超时和取消时用 `ReceiveContext`。

## 错误参考

| 错误 | 来源 | 语义 |
| --- | --- | --- |
| `io.EOF` | `StreamReader.Recv` | 流已结束 |
| `ErrNoValue` | `StreamReaderWithConvert` | 跳过当前元素 |
| `ErrRecvAfterClosed` | Copy 子流 | 子流关闭后继续 `Recv` |
| `ErrEmptyPromises` | `PromiseRace` / `PromiseAny` | 空 Promise 输入 |
| `SourceEOF` | `MergeNamedStreamReaders` | 某条具名源流结束 |
| `ErrDuplicateID` | `ReplyPool.Register` | 重复 correlation ID |
| `ErrReplyExpired` | `ReplyPool` | entry TTL 过期 |
| `ErrReplyClosed` | `ReplyPool` | pool 已关闭 |
| `ErrChanClosed` | `UnboundedChan` | 向已关闭通道发送 |

错误判断优先使用 `errors.Is` / `errors.As`。`Invoke` 会使用 `errors.Join` 聚合多个错误，不要依赖完整字符串相等。

## 资源释放清单

| 资源 | 必做 |
| --- | --- |
| `StreamWriter` | 生产结束后 `Close()`，让读端收到 EOF |
| `StreamReader` | 消费结束或提前退出时 `Close()`，通知写端停止 |
| `EventEmitter` | 服务退出时 `Close()`，关闭全部订阅者 |
| `ReplyPool` | 服务退出时 `Close()`，停止时间轮和统计循环 |
| `Reactor` | 服务退出时 `Release()`，释放 ants pool |
| `UnboundedChan` | 不再生产时 `Close()`，唤醒阻塞消费者 |

## 实现注意点

- `MergeStreamReaders` 在 <=5 路时使用静态 `select`，超过 5 路走 `reflect.Select`。
- `StreamReaderFromArray` 是零 goroutine 的同步流，适合把已有切片接入 Stream API。
- `Copy` 的普通流使用链表 + `sync.Once` 广播，同一节点只从源流读取一次。
- `Promise`、`ReplyPool`、`EventEmitter`、`UnboundedChan` 和 `Reactor` 的公开方法可并发调用。
- 单个 `StreamReader.Recv` 不并发安全；多消费者必须先 `Copy`。
