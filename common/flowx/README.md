# flowx — go-zero 集成的 workflow 封装

基于 [Azure/go-workflow](https://github.com/Azure/go-workflow) 的 go-zero 框架集成封装。

## 快速开始

```go
package main

import (
    "context"
    "zero-service/common/flowx"
    flow "github.com/Azure/go-workflow"
)

func main() {
    w := flowx.New(
        flowx.WithMaxConcurrency(5),
        flowx.WithDontPanic(),
        flowx.WithStepInterceptor(flowx.StepFields()),
        flowx.WithStepInterceptor(flowx.LoggingStepInterceptor{}),
    )

    fetch := &FetchUser{Name: "Alice"}
    build := &BuildProfile{}

    w.Add(
        flow.Step(build).
            DependsOn(fetch).
            Input(func(ctx context.Context, b *BuildProfile) error {
                b.Name = fetch.Name
                return nil
            }),
    )

    if err := w.Do(context.Background()); err != nil {
        panic(err)
    }
}

type FetchUser struct{ Name string }
func (f *FetchUser) Do(ctx context.Context) error { return nil }

type BuildProfile struct{ Name string }
func (b *BuildProfile) Do(ctx context.Context) error { return nil }
```

输出日志示例（`StepFields` 注入 `step` 字段到 ctx）：

```
{"content":"[flowx] step done","duration":"1.2ms","step":"*main.FetchUser"}
{"content":"[flowx] step done","duration":"0.5ms","step":"*main.BuildProfile"}
```

## 核心 API

### 构造函数

```go
w := flowx.New(opts ...FlowOption)
```

不注入任何默认拦截器，全部显式配置。

### 可用选项

| 选项 | 说明 |
|------|------|
| `WithMaxConcurrency(n)` | 最大并发步骤数 |
| `WithDontPanic()` | 捕获 panic 为 error |
| `WithSkipAsError()` | 将 Skipped 步骤视为失败 |
| `WithDontInherit()` | 子 Workflow 不继承父 Option |
| `WithClock(c)` | 注入时间源（测试用） |
| `WithStepDefaults(sd)` | 设置全局默认 StepOption |
| `WithStepInterceptor(ic)` | 追加 StepInterceptor |
| `WithAttemptInterceptor(ic)` | 追加 AttemptInterceptor |
| `WithMutator(m)` | 追加跨切面配置注入器 |

### 拦截器

| 拦截器 | 类型 | 说明 |
|--------|------|------|
| `LoggingStepInterceptor{}` | StepInterceptor | 记录 step done/failed（含耗时），日志走 `logx` |
| `StepFields(extra...)` | StepInterceptor → 函数 | 注入 `step=<名称>` + 自定义字段到 ctx |
| `AttemptFields(extra...)` | AttemptInterceptor → 函数 | 注入 `attempt=<序号>` + 自定义字段到 ctx |

**推荐顺序**：`StepFields()` 放外层，`LoggingStepInterceptor{}` 紧随其后，这样日志自动携带 `step` 字段：

```go
w := flowx.New(
    flowx.WithStepInterceptor(flowx.StepFields()),
    flowx.WithStepInterceptor(flowx.LoggingStepInterceptor{}),
)
```

### SubWorkflow 继承

父 Workflow 的 Option 会自动继承到子 Workflow：

```go
parent := flowx.New(
    flowx.WithMaxConcurrency(10),
    flowx.WithDontPanic(),
)
child := flowx.New()
child.Add(flow.Step(flow.Func("task", func(ctx context.Context) error {
    return nil
})))
parent.Add(flow.Step(child))
_ = parent.Do(ctx)
```

子 Workflow 独立运行用 `WithDontInherit()`。

## 更深入学习

- [go-workflow 官方示例](https://github.com/Azure/go-workflow/tree/main/example) — 从 quickstart 到 mutators 的完整学习路径
- [go-workflow README](https://github.com/Azure/go-workflow#readme) — API 参考
