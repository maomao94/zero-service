# Flow 工作流编排规范

## 适用范围

修改 `common/flowx` 或使用 Azure go-workflow 编排的代码时读取。

## flowx — Azure go-workflow 包装

### 核心依赖

`common/flowx` 封装 `github.com/Azure/go-workflow`，提供 go-zero 日志集成和功能选项模式的配置。

依据：`common/flowx/flowx.go`、`common/flowx/options.go`。

### 关键类型

| 类型 | 用途 |
|------|------|
| `FlowOptions` | 工作流配置选项，字段为指针（nil = 未设置，不覆盖默认） |
| `FlowOption` | 功能选项函数: `WithMaxConcurrency(n)`、`WithDontPanic()`、`WithSkipAsError()`、`WithDontInherit()`、`WithClock(c)`、`WithStepDefaults(sd)`、`WithStepInterceptor(ic)`、`WithAttemptInterceptor(ic)`、`WithMutator(m)` |
| `LoggingStepInterceptor` | `flow.StepInterceptor` 实现，通过 go-zero logx 记录每个 Step 的开始/耗时/错误 |

### 创建 Workflow

```go
wf := flowx.New(
    flowx.WithMaxConcurrency(5),
    flowx.WithStepInterceptor(flowx.StepFields()),
    flowx.WithStepInterceptor(flowx.LoggingStepInterceptor{}),
    flowx.WithAttemptInterceptor(flowx.AttemptFields()),
)
```

- `flowx.New()` 接收可变 `FlowOption`，映射到 `flow.WorkflowOption`。
- 未设置的字段保持 go-workflow 默认值。

依据：`common/flowx/flowx.go`。

### 日志拦截器

#### LoggingStepInterceptor

- 实现 `flow.StepInterceptor` 接口。
- 每次 Step 执行时记录开始、耗时（`WithDuration`）、成功/失败。
- 成功日志: `Info("[flowx] step done")`；失败日志: `Errorf("[flowx] step failed: %+v", err)`

#### StepFields

- 将 `step=<步骤名>` 注入 logx context。
- 支持 extra 函数追加字段。
- **必须放在 `LoggingStepInterceptor` 外层**，确保日志拦截器能读取到 step 字段：

```go
// 正确顺序：
flowx.WithStepInterceptor(flowx.StepFields()),           // 外层: 注入 step 字段
flowx.WithStepInterceptor(flowx.LoggingStepInterceptor{}), // 内层: 使用 step 字段记录日志
```

#### AttemptFields

- 将 `attempt=<尝试序号>` 注入 logx context。
- 支持 extra 函数追加字段。

依据：`common/flowx/flowx.go`。

### Mutator

- `WithMutator(m)` 支持跨步骤的配置注入。
- Mutator 在步骤创建时调用，可修改步骤的 `StepOption`。
- 典型用途: 统一设置超时、重试策略、日志级别等。

依据：`common/flowx/flowx.go`（Workflow 实例化时传入 `Mutators`）。

## 反模式

- 拦截器顺序错误：`LoggingStepInterceptor` 放在 `StepFields` 外层（导致日志丢失 step 字段）。
- 在 `FlowOptions` 中设置零值以为会覆盖默认（应使用指针字段，nil = 不设置）。
- 用 `go-workflow` 的 `panic` 机制处理业务错误（应使用 `DontPanic` + error 返回）。
- 在 go-workflow step 中直接使用全局 logger 而非 context logger。

## 验证

- 验证 `FlowOptions` 各字段映射到 `WorkflowOption` 的正确性。
- 验证拦截器顺序对日志输出的影响。
- 验证 `LoggingStepInterceptor` 在成功/失败路径下的日志格式。
- 验证 `StepFields` 和 `AttemptFields` 的 extra 函数行为。
- 验证 `WithDontPanic` 和 `WithSkipAsError` 的错误处理语义。
