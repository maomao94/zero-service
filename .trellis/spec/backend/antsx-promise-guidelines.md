# antsx Promise 使用规范

> `antsx.Promise` 异步结果容器和并行组合的 canonical source，覆盖签名、选型、泄漏防护和测试断言。

## When to read

- 使用或修改 `common/antsx` 的 Promise 系列函数。
- 需要异步结果容器、并行组合（All/AllSettled/Race/Any）或链式变换。
- 排查 goroutine 泄漏、Promise 永不完成或错误语义不精确。

## 核心签名

```go
// 结果容器
func NewPromise[T any]() *Promise[T]
func (p *Promise[T]) Resolve(val T)
func (p *Promise[T]) Reject(err error)
func (p *Promise[T]) Await(ctx context.Context) (T, error)

// 并行组合
func PromiseAll[T any](ctx context.Context, promises ...*Promise[T]) ([]T, error)
func PromiseAllSettled[T any](ctx context.Context, promises ...*Promise[T]) []PromiseResult[T]
func PromiseRace[T any](ctx context.Context, promises ...*Promise[T]) (T, error)
func PromiseAny[T any](ctx context.Context, promises ...*Promise[T]) (T, error)

// 结果类型
type PromiseResult[T any] struct {
    Val T
    Err error
}
func (r PromiseResult[T]) Succeeded() bool
```

## 并行组合四象限

| | fast-fail | 容错 |
|---|---|---|
| **等全部** | `PromiseAll` | `PromiseAllSettled` |
| **取最快** | `PromiseRace` | `PromiseAny` |

选型：

| 场景 | 用哪个 |
| --- | --- |
| 需要全部成功，一个失败就停 | `PromiseAll` |
| 需要全部结果，部分失败做降级 | `PromiseAllSettled` |
| 取第一个完成的（含错误） | `PromiseRace` |
| 取第一个成功的，跳过失败 | `PromiseAny` |

Contracts:

- `PromiseAll` 任一失败立即 cancel 返回第一个错误
- `PromiseAllSettled` 每个独立，不互相影响，返回 `[]PromiseResult`
- `PromiseRace` 第一个完成的无论成败都返回
- `PromiseAny` 第一个成功才返回，全部失败才报错
- 空输入：`PromiseAll` 返回空切片，`PromiseRace`/`PromiseAny` 返回 `ErrEmptyPromises`

## 错误语义

| 错误 | 来源 | 含义 |
| --- | --- | --- |
| `ErrEmptyPromises` | `PromiseRace` / `PromiseAny` | 传入空 promises 切片 |
| `io.EOF` | `StreamReader.Recv` | 流结束 |

ReplyPool 相关错误（`ErrReplyExpired`、`ErrReplyClosed`、`ErrDuplicateID`）见 [antsx-replypool-guidelines.md](./antsx-replypool-guidelines.md)。

Good:

```go
results := antsx.PromiseAllSettled(ctx, p1, p2, p3)
for _, r := range results {
    if r.Succeeded() {
        process(r.Val)
    } else {
        log.Printf("failed: %v", r.Err)
    }
}
```

Base:

```go
val, err := antsx.PromiseAny(ctx, cdn1, cdn2, cdn3)
if err != nil {
    if errors.Is(err, antsx.ErrEmptyPromises) {
        // no promises provided
    }
}
```

Bad:

```go
// 不要用 context.Canceled 判断空输入
if errors.Is(err, context.Canceled) { }  // 语义不精确
```

## Goroutine 泄漏防护

`Then`、`Map`、`FlatMap`、`Go` 内部启动 goroutine，依赖 ctx 取消或 Promise 完成来退出。

**如果 ctx 为 `context.Background()` 且 Promise 永远不完成，goroutine 将泄漏。**

Wrong:

```go
// 没有超时的链式操作，可能泄漏
result := antsx.Then(context.Background(), p, transform)
```

Correct:

```go
// 带超时的链式操作
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
result := antsx.Then(ctx, p, transform)
```

## Reactor nil 防护

`Submit`、`Post`、`Go` 对 nil Reactor 返回 `errNilReactor`，不会 panic。

```go
_, err := antsx.Submit(ctx, nil, fn)
if errors.Is(err, errNilReactor) {
    // Reactor not initialized
}
```

## Post 签名

`Post` 是 fire-and-forget，签名不接受 error 返回值：

```go
func Post(ctx context.Context, r *Reactor, fn func(ctx context.Context)) error
```

如果需要 error 返回值，用 `Submit`。

## Validation & Error Matrix

| 条件 | 正确行为 |
| --- | --- |
| 空 promises 切片 | `PromiseAll` 返回空切片，`PromiseRace`/`PromiseAny` 返回 `ErrEmptyPromises` |
| 单个 Promise | 所有函数正常工作 |
| 全部成功 | 按定义顺序返回结果 |
| 全部失败 | `PromiseAll` 返回第一个错误，`PromiseAllSettled` 返回所有错误，`PromiseAny` 返回聚合错误 |
| ctx 取消 | 通过 ctx.Err() 返回，goroutine 不泄漏 |
| nil Reactor | 返回 `errNilReactor` |

## Tests Required

- 空输入：所有四个组合函数
- 单个 Promise：成功和失败
- 全部成功：验证结果顺序
- 部分失败：`PromiseAll` fast-fail，`PromiseAllSettled` 独立返回
- 全部失败：错误聚合
- ctx 超时/取消：验证 goroutine 不泄漏
- `PromiseResult.Succeeded()`：成功和失败两种路径

## Related

- [antsx-replypool-guidelines.md](./antsx-replypool-guidelines.md) — ReplyPool 异步请求/应答池规范（设计决策、时间轮、统计循环、生命周期）
