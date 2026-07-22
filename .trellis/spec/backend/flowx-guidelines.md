# flowx Workflow 封装

> `common/flowx` 基于 go-workflow 的 go-zero 集成封装。提供 StepInterceptor/AttemptInterceptor、ctx 字段注入和 FlowOptions 配置。

## When to read

- 需要在 go-zero 服务中编排 DAG 工作流（并发步骤、依赖编排、重试超时）
- 需要为所有 Step 统一日志输出
- 需要在 Step 内部自动携带上下文字段（step、attempt）

## 包结构

| 文件 | 职责 |
|------|------|
| `flowx.go` | `New()` 构造函数 + `LoggingStepInterceptor` + `StepFields` + `AttemptFields` |
| `options.go` | `FlowOptions` 配置结构体 + 8 个 `FlowOption` 函数 |
| `flowx_test.go` | 17 个单测（配置 + 拦截器 + 运行态） |
| `README.md` | 快速开始和学习文档 |

## 构造函数

### New

```go
flowx.New(opts ...FlowOption) *flow.Workflow
```

不注入任何默认拦截器，全部显式配置。`FlowOptions` 与 `WorkflowOption` 1:1 对齐。

### 推荐用法

```go
w := flowx.New(
    flowx.WithMaxConcurrency(5),
    flowx.WithDontPanic(),

    // StepFields 放外层，注入 step 字段到 ctx
    flowx.WithStepInterceptor(flowx.StepFields()),

    // LoggingStepInterceptor 紧随其后，日志自动携带 step 字段
    flowx.WithStepInterceptor(flowx.LoggingStepInterceptor{}),

    // 可选：注入 attempt 字段（重试时自动递增）
    flowx.WithAttemptInterceptor(flowx.AttemptFields()),
)
```

## FlowOptions

与 `WorkflowOption` 字段完全对齐：

| FlowOptions 字段 | WorkflowOption 字段 | Option 函数 |
|---|---|---|
| `MaxConcurrency *int` | `MaxConcurrency *int` | `WithMaxConcurrency(n)` |
| `DontPanic *bool` | `DontPanic *bool` | `WithDontPanic()` |
| `SkipAsError *bool` | `SkipAsError *bool` | `WithSkipAsError()` |
| `DontInherit bool` | `DontInherit bool` | `WithDontInherit()` |
| `Clock clock.Clock` | `Clock clock.Clock` | `WithClock(c)` |
| `StepDefaults *StepOption` | `StepDefaults *StepOption` | `WithStepDefaults(sd)` |
| `StepInterceptors []` | `StepInterceptors []` | `WithStepInterceptor(ic)` |
| `AttemptInterceptors []` | `AttemptInterceptors []` | `WithAttemptInterceptor(ic)` |
| `Mutators []Mutator` | `Mutators []Mutator` | `WithMutator(m)` |

## 拦截器

### LoggingStepInterceptor

Step 级拦截器（跨重试），记录 step done/failed，耗时通过 `logx.WithDuration` 输出：

```
{"content":"[flowx] step done","duration":"1.2ms","step":"*pkg.MyStep"}
{"content":"[flowx] step failed","duration":"3.0s","step":"*pkg.MyStep","error":"connection refused"}
```

- 失败时使用 `%+v` 格式输出错误（符合日志规范中拦截器级标准）
- 不打印 `step start`（由 StepFields 注入的 step 字段区分）

### StepFields(extra ...func) StepInterceptor

注入 `step=<步骤名>` 及自定义字段到 ctx。使用 `flow.StepInterceptorFunc` 创建，支持 extra 函数动态追加字段：

```go
// 仅 step 名
flowx.StepFields()

// step + 自定义字段
flowx.StepFields(func(ctx context.Context, _ flow.Steper) []logx.LogField {
    return []logx.LogField{logx.Field("tenant", tenantFrom(ctx))}
})
```

**必须放在 LoggingStepInterceptor 外层**（索引 0），否则 LoggingStepInterceptor 无法从 ctx 读取 step 字段。

### AttemptFields(extra ...func) AttemptInterceptor

注入 `attempt=<尝试序号>` 及自定义字段到 ctx。每次重试时 `attempt` 自动递增。

## 拦截器链执行顺序

```
StepFields (index 0, 外层)
  └→ LoggingStepInterceptor (index 1)
      └→ Step.Do()
          └→ 内部 logx.WithContext(ctx) 自动携带 step + attempt 字段
```

## 日志规范

- 所有 `common/flowx` 日志使用 `[flowx]` 前缀
- 错误使用 `Errorf("%+v", err)`（拦截器级）
- 耗时使用 `logx.WithDuration` 结构化输出
- step 名通过 `StepFields` 注入 ctx，不嵌入消息文本

## Step 内部写日志

```go
func (s *MyStep) Do(ctx context.Context) error {
    logx.WithContext(ctx).Infof("processing item %d", s.ID)
    // 输出自动携带 step=*pkg.MyStep attempt=0
    return nil
}
```

## SubWorkflow

嵌入 `flow.Workflow` 即可作为 Step：

```go
type MySubFlow struct{ flow.Workflow }

func (m *MySubFlow) Do(ctx context.Context) error {
    return m.Workflow.Do(ctx)
}
```

父 Workflow 的 Option 自动继承。用 `WithDontInherit()` 可独立运行。

## 依赖

- `github.com/Azure/go-workflow`（固定到包含拦截器和 Mutator API 的官方版本，禁止本地路径 replace）
- `github.com/zeromicro/go-zero/core/logx`
- `github.com/benbjohnson/clock`（仅用于 `WithClock` 测试注入）
