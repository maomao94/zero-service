package flowx

import (
	"context"
	"fmt"
	"time"

	flow "github.com/Azure/go-workflow"
	"github.com/zeromicro/go-zero/core/logx"
)

// LoggingStepInterceptor 日志拦截器：通过 go-zero logx 记录每个 Step 的全生命周期（跨重试）。
type LoggingStepInterceptor struct{}

// InterceptStep 实现 flow.StepInterceptor 接口。
func (LoggingStepInterceptor) InterceptStep(ctx context.Context, step flow.Steper, next func(context.Context) error) error {
	start := time.Now()
	err := next(ctx)
	duration := time.Since(start)
	if err != nil {
		logx.WithContext(ctx).WithDuration(duration).Errorf("[flowx] step failed: %+v", err)
	} else {
		logx.WithContext(ctx).WithDuration(duration).Info("[flowx] step done")
	}
	return err
}

// StepFields 返回一个 StepInterceptor，将 step=<步骤名> 及 extra 函数产出的字段注入 ctx。
// 配合 LoggingStepInterceptor 时，StepFields 应放在外层：
//
//	flowx.New(
//	    flowx.WithStepInterceptor(flowx.StepFields()),
//	    flowx.WithStepInterceptor(flowx.LoggingStepInterceptor{}),
//	)
func StepFields(extra ...func(context.Context, flow.Steper) []logx.LogField) flow.StepInterceptor {
	return flow.StepInterceptorFunc(func(ctx context.Context, step flow.Steper, next func(context.Context) error) error {
		fields := []logx.LogField{logx.Field("step", flow.String(step))}
		for _, fn := range extra {
			fields = append(fields, fn(ctx, step)...)
		}
		ctx = logx.ContextWithFields(ctx, fields...)
		return next(ctx)
	})
}

// AttemptFields 返回一个 AttemptInterceptor，将 attempt=<尝试序号> 及 extra 函数产出的字段注入 ctx。
func AttemptFields(extra ...func(context.Context, flow.Steper, uint64) []logx.LogField) flow.AttemptInterceptor {
	return flow.AttemptInterceptorFunc(func(ctx context.Context, step flow.Steper, attempt uint64, next func(context.Context) error) error {
		fields := []logx.LogField{logx.Field("attempt", attempt)}
		for _, fn := range extra {
			fields = append(fields, fn(ctx, step, attempt)...)
		}
		ctx = logx.ContextWithFields(ctx, fields...)
		return next(ctx)
	})
}

func stepName(step flow.Steper) string {
	if s, ok := step.(fmt.Stringer); ok {
		return s.String()
	}
	return fmt.Sprintf("%T", step)
}

// New 创建一个 *flow.Workflow。
func New(opts ...FlowOption) *flow.Workflow {
	o := &FlowOptions{}
	for _, opt := range opts {
		opt(o)
	}

	w := &flow.Workflow{
		Option: flow.WorkflowOption{
			MaxConcurrency:      o.MaxConcurrency,
			DontPanic:           o.DontPanic,
			SkipAsError:         o.SkipAsError,
			DontInherit:         o.DontInherit,
			Clock:               o.Clock,
			StepDefaults:        o.StepDefaults,
			Mutators:            o.Mutators,
			StepInterceptors:    o.StepInterceptors,
			AttemptInterceptors: o.AttemptInterceptors,
		},
	}

	return w
}

func ptr[T any](v T) *T { return &v }
