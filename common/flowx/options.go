package flowx

import (
	flow "github.com/Azure/go-workflow"
	"github.com/benbjohnson/clock"
)

// FlowOptions 创建 Workflow 的配置结构体。标量字段为指针类型以对齐 WorkflowOption 的
// nil=未设置 语义；零值即默认行为（无并发限制、不捕获 panic 等）。
type FlowOptions struct {
	MaxConcurrency     *int              // 最大并发步骤数，nil 或 0 表示无限制
	DontPanic          *bool             // 非 nil 且 true 时捕获 panic 并转为 error
	SkipAsError        *bool             // 非 nil 且 true 时将 Skipped 步骤视为工作流失败
	DontInherit        bool              // 作为子 Workflow 时不继承父 Option
	Clock              clock.Clock       // 时间源，nil 为真实时钟
	StepDefaults       *flow.StepOption  // 应用于所有 Step 的默认选项
	Mutators           []flow.Mutator    // 跨切面 Step 配置注入器
	StepInterceptors   []flow.StepInterceptor
	AttemptInterceptors []flow.AttemptInterceptor
}

// FlowOption 函数式选项。
type FlowOption func(*FlowOptions)

// ---- 工作流级别 ----

// WithMaxConcurrency 限制同时运行的 Step 数量。0 表示无限制。
func WithMaxConcurrency(n int) FlowOption {
	return func(o *FlowOptions) { o.MaxConcurrency = ptr(n) }
}

// WithDontPanic 启用 panic 恢复：Step 内部 panic 会被捕获并转为 error，不会导致程序崩溃。
func WithDontPanic() FlowOption {
	return func(o *FlowOptions) { o.DontPanic = ptr(true) }
}

// WithSkipAsError 将因条件不满足而被跳过的 Step 视为工作流失败。
func WithSkipAsError() FlowOption {
	return func(o *FlowOptions) { o.SkipAsError = ptr(true) }
}

// WithDontInherit 设置此 Workflow 作为子 Workflow 时不继承父 Workflow 的 Option。
func WithDontInherit() FlowOption {
	return func(o *FlowOptions) { o.DontInherit = true }
}

// WithClock 注入时间源。在单元测试中用 clock.NewMock() 实现确定性时间控制。
func WithClock(c clock.Clock) FlowOption {
	return func(o *FlowOptions) { o.Clock = c }
}

// WithStepDefaults 直接设置应用于所有 Step 的默认选项。
func WithStepDefaults(sd *flow.StepOption) FlowOption {
	return func(o *FlowOptions) { o.StepDefaults = sd }
}

// ---- 拦截器 & 切面 ----

// WithStepInterceptor 追加一个 StepInterceptor。按追加顺序执行，先追加的在外层。
func WithStepInterceptor(ic flow.StepInterceptor) FlowOption {
	return func(o *FlowOptions) {
		o.StepInterceptors = append(o.StepInterceptors, ic)
	}
}

// WithAttemptInterceptor 追加一个 AttemptInterceptor。
func WithAttemptInterceptor(ic flow.AttemptInterceptor) FlowOption {
	return func(o *FlowOptions) {
		o.AttemptInterceptors = append(o.AttemptInterceptors, ic)
	}
}

// WithMutator 追加一个跨切面配置注入器。Mutator 会为所有匹配类型的 Step 注入配置
//（默认值、重试策略、回调等），包括子 Workflow 中的 Step。使用 flow.Mutate[T]() 创建。
func WithMutator(m flow.Mutator) FlowOption {
	return func(o *FlowOptions) {
		o.Mutators = append(o.Mutators, m)
	}
}
