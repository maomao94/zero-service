# antsx

Go 泛型异步编排工具包。底层基于 Go channel、context、go-zero TimingWheel / threading.GoSafe 和 [ants](https://github.com/panjf2000/ants) 协程池。Stream API 参考 [eino](https://github.com/cloudwego/eino) 的 Pipe / StreamReader / StreamWriter 范式。

## 模块总览

| 模块 | 一句话 | 文件 |
| --- | --- | --- |
| **Promise** | 异步结果容器（底层原语，其他模块内部都用它） | `promise.go` |
| **Reactor** | 协程池并发限制器（控制同时跑多少 goroutine） | `reactor.go` |
| **Invoke** | 一组函数的批量并行（发起 + 等待一体） | `invoke.go` |
| **ReplyPool** | correlation ID 请求-响应匹配（跨 goroutine/连接/进程） | `replypool.go` |
| **Stream** | 流式管道（单生产者-单消费者，支持转换/合并/Copy） | `stream.go` `select.go` |
| **TeeWriter** | io.Reader 链路扇出（边上传边计算 hash） | `tee.go` |
| **EventEmitter** | 进程内 topic 发布订阅（订阅端是 StreamReader） | `emitter.go` |
| **UnboundedChan** | MPMC 无界阻塞队列 | `unbounded.go` |

依赖关系：`Promise` ← `Reactor`(Submit) ← `Invoke`(WithReactor)；`Promise` ← `ReplyPool`(内部存储)。

---

## 我该用哪个？

从你的**场景**出发选 API：

| 场景 | 用什么 |
| --- | --- |
| 单个后台计算，要拿返回值 | `Go(ctx, fn)` → 返回 `*Promise[T]` |
| 单个后台计算，要拿返回值，要限并发 | `Submit(ctx, reactor, fn)` → 返回 `*Promise[T]` |
| 单个后台计算，不关心返回值 | `Post(ctx, reactor, fn)` 或 `reactor.Go(ctx, fn)` |
| N 个函数并行，全成功才算成功 | `Invoke(ctx, tasks...)` |
| N 个函数并行，各自独立互不影响 | `InvokeAllSettled(ctx, tasks...)` |
| 上面两个想限并发 | `InvokeWithReactor` / `InvokeAllSettledWithReactor` |
| 手里已有多个 Promise，等全部完成 | `PromiseAll` / `PromiseAllSettled` |
| 手里已有多个 Promise，要最快的那个 | `PromiseRace`（不管成败）/ `PromiseAny`（要成功的） |
| 发请求出去，等对方按 ID 回包 | `ReplyPool` + `RequestReply` |
| 流式数据逐条传递 | `Pipe` + `StreamReader` / `StreamWriter` |
| 一份数据广播给多个消费者 | `EventEmitter` 或 `StreamReader.Copy` |
| 生产者消费者解耦，无界缓冲 | `UnboundedChan` |

### FAQ：Submit 和 ReplyPool 看起来很像？

它们的等待端一模一样——都是 `promise.Await(ctx)` 拿结果。**区别在于谁 Resolve 这个 Promise：**

- **Submit**：你传入的 `fn` 跑完后**自动** Resolve。整个过程在进程内，一个 goroutine 搞定。
- **ReplyPool**：你发出请求后，**另一个 goroutine**（网络回包处理器）调 `pool.Resolve(id, val)` 来完成。请求和响应通过 correlation ID 关联。

一句话：Submit = "帮我跑函数"，ReplyPool = "帮我等回包"。

---

## Promise

异步结果容器。`Resolve` / `Reject` 仅第一次调用生效。

```go
p := antsx.NewPromise[int]()
go func() { p.Resolve(42) }()
val, err := p.Await(ctx)
```

**方法：**

| 方法 | 说明 |
| --- | --- |
| `Resolve(val)` / `Reject(err)` | 完成 Promise，仅首次生效 |
| `Await(ctx)` | 阻塞等待；ctx 取消返回 `ctx.Err()` |
| `AwaitWithTimeout(d)` | 快捷超时等待 |
| `Done()` | 完成信号 channel，用于 select |
| `Get()` | 非阻塞检查 `(val, err, ok)` |
| `Catch(ctx, fn)` | 失败时异步执行回调 |
| `FireAndForget(ctx)` | 后台等待并忽略，确保资源释放 |

**链式处理：**

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

> `Then`、`Map`、`FlatMap`、`Go` 都启 goroutine。务必传带超时的 ctx，避免 goroutine 泄漏。

**组合：**

```go
results, err := antsx.PromiseAll(ctx, p1, p2, p3)        // 全成功或 fast-fail
settled := antsx.PromiseAllSettled(ctx, p1, p2, p3)       // 全等完，各自保留
fastest, err := antsx.PromiseRace(ctx, p1, p2)            // 谁先完成用谁
firstOK, err := antsx.PromiseAny(ctx, p1, p2, p3)        // 谁先成功用谁
```

---

## Invoke

一组同类型函数的批量并行。与 PromiseAll 的区别：Invoke 是"定义 + 发起 + 等待"一体的；PromiseAll 只等待已有 Promise。

```go
tasks := []antsx.Task[*Profile]{
    {Name: "db", Fn: func(ctx context.Context) (*Profile, error) { return queryDB(ctx, id) }},
    {Name: "cache", Timeout: 500 * time.Millisecond, Fn: func(ctx context.Context) (*Profile, error) { return queryCache(ctx, id) }},
}

// fast-fail：一个失败全部取消
profiles, err := antsx.Invoke(ctx, tasks...)

// 容错：各自独立，互不影响
for _, r := range antsx.InvokeAllSettled(ctx, tasks...) {
    if r.Succeeded() { use(r.Val) }
}

// 限并发
results, err := antsx.InvokeWithReactor(ctx, reactor, tasks...)
```

> `Task.Fn` 必须检查 `ctx.Done()`，cancel 只发信号不强制终止。结果按输入顺序返回。

---

## Reactor

并发限制器，基于 ants 协程池。

```go
reactor, err := antsx.NewReactor(100)
if err != nil { return err }
defer reactor.Release()
```

| API | 返回值 | 场景 |
| --- | --- | --- |
| `Submit(ctx, reactor, fn)` | `*Promise[T], error` | 有返回值 |
| `Post(ctx, reactor, fn)` | `error` | fire-and-forget |
| `reactor.Go(ctx, fn)` | `error` | fire-and-forget |

```go
p, _ := antsx.Submit(ctx, reactor, func(ctx context.Context) (int, error) {
    return compute(ctx)
})
val, err := p.Await(ctx)
```

> nil Reactor 返回错误不 panic。`Release()` 后不要再提交。

---

## ReplyPool

跨边界的请求-响应 ID 匹配。内部用 Promise 承载等待，用 TimingWheel 管超时。

**典型流程：** goroutine A 注册 ID 并发请求 → goroutine B 收到回包后按 ID Resolve → goroutine A 的 Await 返回。

**一体化（推荐）：**

```go
pool := antsx.NewReplyPool[Response](
    antsx.WithName("drc-reply"),
    antsx.WithDefaultTTL(30*time.Second),
)
defer pool.Close()

// 注册 ID → 发送 → 等待，一步到位
resp, err := antsx.RequestReply(ctx, pool, requestID, func() error {
    return conn.Send(request)
}, 10*time.Second)
```

**分步（注册/发送/响应在不同代码路径）：**

```go
// 发送端
promise, _ := pool.Register(requestID, 5*time.Second)
conn.Send(request)

// 接收端（另一个 goroutine）
pool.Resolve(requestID, response)

// 发送端取结果
resp, err := promise.Await(ctx)
```

**注意：**

- ctx 只控制等待，不控制 entry 生命周期。Await 超时后**不要手动 Reject**，时间轮 TTL 到期会自动触发 `ErrReplyExpired`。
- 构造后自动启动每分钟统计日志，`Close()` 自动停止。
- 重复 ID 返回 `ErrDuplicateID`。

---

## Stream

流式数据管道，单生产者-单消费者。

```go
sr, sw := antsx.Pipe[string](10)
defer sr.Close()

go func() {
    defer sw.Close()
    if closed := sw.Send("hello", nil); closed { return }
    sw.Send("world", nil)
}()

for {
    val, err := sr.Recv()
    if errors.Is(err, io.EOF) { break }
    if err != nil { return err }
    fmt.Println(val)
}
```

**规则：** 写端必须 Close（否则读端无 EOF）；读端也要 Close（否则写端阻塞）；单 Reader 不并发安全。

**转换过滤：**

```go
converted := antsx.StreamReaderWithConvert(sr, func(i int) (string, error) {
    if i == 0 { return "", antsx.ErrNoValue } // 跳过
    return fmt.Sprintf("v%d", i), nil
})
```

**合并：**

```go
merged := antsx.MergeStreamReaders([]*antsx.StreamReader[string]{a, b, c})
```

**Copy 广播：**

```go
copies := sr.Copy(2)  // 必须在消费前调用
```

---

## TeeWriter

io.Writer 扇出：写入同时进入内部 pipe 和所有附加 writer。

```go
hash := md5.New()
tee := antsx.NewTeeWriter(hash)
defer tee.Close()

go func() {
    if err := uploadOSS(ctx, tee.Reader()); err != nil { tee.CloseWithError(err) }
}()
io.Copy(tee, sourceReader)
```

---

## EventEmitter

进程内 topic 发布订阅。

```go
emitter := antsx.NewEventEmitter[string]()
defer emitter.Close()

sr, cancel := emitter.Subscribe(ctx, "chat", 64)
defer cancel()
defer sr.Close()

emitter.Emit("chat", "hello")
val, _ := sr.Recv()
```

> ctx 取消自动退订。Emit 对无订阅者静默忽略。

---

## UnboundedChan

无界阻塞队列（MPMC）。

```go
ch := antsx.NewUnboundedChan[func()]()
defer ch.Close()

ch.Send(task)              // 关闭后 panic（同原生 channel）
ch.TrySend(task)           // 关闭后返回 false
fn, ok := ch.Receive()     // 阻塞到有数据或关闭
fn, ok := ch.ReceiveContext(ctx)  // 支持 ctx 取消
```

---

## 附录

### 错误表

| 错误 | 来源 | 含义 |
| --- | --- | --- |
| `io.EOF` | StreamReader.Recv | 流结束 |
| `ErrNoValue` | StreamReaderWithConvert | 过滤跳过 |
| `ErrRecvAfterClosed` | Copy 子流 | 关闭后 Recv |
| `ErrEmptyPromises` | PromiseRace / PromiseAny | 空输入 |
| `SourceEOF` | MergeNamedStreamReaders | 某条源流结束 |
| `ErrDuplicateID` | ReplyPool.Register | ID 重复 |
| `ErrReplyExpired` | ReplyPool | TTL 过期 |
| `ErrReplyClosed` | ReplyPool | 池已关闭 |
| `ErrChanClosed` | UnboundedChan | 通道已关闭 |

用 `errors.Is` / `errors.As` 判断。Invoke 用 `errors.Join` 聚合，不要做字符串比较。

### 资源释放

| 资源 | 方式 | 不释放后果 |
| --- | --- | --- |
| StreamWriter | `Close()` | 读端无 EOF |
| StreamReader | `Close()` | 写端阻塞 |
| EventEmitter | `Close()` | 订阅者不关闭 |
| ReplyPool | `Close()` | 时间轮泄漏 |
| Reactor | `Release()` | 池不回收 |
| UnboundedChan | `Close()` | 消费者永远阻塞 |

### 实现细节

- MergeStreamReaders ≤5 路静态 select，>5 路 reflect.Select。
- StreamReaderFromArray 零 goroutine 同步流。
- Copy 普通流用链表 + sync.Once 广播。
- Promise / ReplyPool / EventEmitter / UnboundedChan / Reactor 公开方法并发安全。
- 单个 StreamReader.Recv 不并发安全，多消费者先 Copy。
