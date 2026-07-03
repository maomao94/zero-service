package flowx

import (
	"bytes"
	"context"
	"strings"
	"testing"

	flow "github.com/Azure/go-workflow"
	"github.com/benbjohnson/clock"
	"github.com/zeromicro/go-zero/core/logx"
)

func TestNew_NoInterceptorsByDefault(t *testing.T) {
	w := New()
	if len(w.Option.StepInterceptors) != 0 {
		t.Fatalf("expected 0 StepInterceptor, got %d", len(w.Option.StepInterceptors))
	}
}

func TestNew_InterceptorsOrder(t *testing.T) {
	custom := &spyInterceptor{}
	w := New(
		WithStepInterceptor(StepFields()),
		WithStepInterceptor(LoggingStepInterceptor{}),
		WithStepInterceptor(custom),
	)
	if len(w.Option.StepInterceptors) != 3 {
		t.Fatalf("expected 3 StepInterceptors, got %d", len(w.Option.StepInterceptors))
	}
	if _, ok := w.Option.StepInterceptors[1].(LoggingStepInterceptor); !ok {
		t.Fatalf("expected LoggingStepInterceptor at index 1, got %T", w.Option.StepInterceptors[1])
	}
}

func TestNew_WithMaxConcurrency(t *testing.T) {
	w := New(WithMaxConcurrency(3))
	if w.Option.MaxConcurrency == nil || *w.Option.MaxConcurrency != 3 {
		t.Fatalf("expected MaxConcurrency=3, got %v", w.Option.MaxConcurrency)
	}
}

func TestNew_WithDontPanic(t *testing.T) {
	w := New(WithDontPanic())
	if w.Option.DontPanic == nil || !*w.Option.DontPanic {
		t.Fatal("expected DontPanic=true")
	}
}

func TestNew_WithSkipAsError(t *testing.T) {
	w := New(WithSkipAsError())
	if w.Option.SkipAsError == nil || !*w.Option.SkipAsError {
		t.Fatal("expected SkipAsError=true")
	}
}

func TestNew_WithDontInherit(t *testing.T) {
	w := New(WithDontInherit())
	if !w.Option.DontInherit {
		t.Fatal("expected DontInherit=true")
	}
}

func TestNew_WithAttemptInterceptor(t *testing.T) {
	ic := &spyAttemptInterceptor{}
	w := New(WithAttemptInterceptor(ic))
	if len(w.Option.AttemptInterceptors) != 1 {
		t.Fatalf("expected 1 AttemptInterceptor, got %d", len(w.Option.AttemptInterceptors))
	}
}

func TestNew_ComposeOptions(t *testing.T) {
	w := New(
		WithMaxConcurrency(5),
		WithDontPanic(),
		WithSkipAsError(),
		WithDontInherit(),
	)
	if w.Option.MaxConcurrency == nil || *w.Option.MaxConcurrency != 5 {
		t.Fatal("expected MaxConcurrency=5")
	}
	if w.Option.DontPanic == nil || !*w.Option.DontPanic {
		t.Fatal("expected DontPanic=true")
	}
	if w.Option.SkipAsError == nil || !*w.Option.SkipAsError {
		t.Fatal("expected SkipAsError=true")
	}
	if !w.Option.DontInherit {
		t.Fatal("expected DontInherit=true")
	}
}

func TestNew_WithClock(t *testing.T) {
	mock := clock.NewMock()
	w := New(WithClock(mock))
	if w.Option.Clock != mock {
		t.Fatal("expected Clock=mock")
	}
}

func TestNew_WithMutator(t *testing.T) {
	m := flow.Mutate[*countStep](func(ctx context.Context, step *countStep) flow.Builder {
		return nil
	})
	w := New(WithMutator(m))
	if len(w.Option.Mutators) != 1 {
		t.Fatalf("expected 1 Mutator, got %d", len(w.Option.Mutators))
	}
}

func TestNew_WithStepFields(t *testing.T) {
	w := New(
		WithStepInterceptor(StepFields()),
		WithAttemptInterceptor(AttemptFields()),
	)
	if len(w.Option.StepInterceptors) != 1 {
		t.Fatalf("expected 1 StepInterceptor, got %d", len(w.Option.StepInterceptors))
	}
	if len(w.Option.AttemptInterceptors) != 1 {
		t.Fatalf("expected 1 AttemptInterceptor, got %d", len(w.Option.AttemptInterceptors))
	}
}

func TestNew_NoOptionsDefault(t *testing.T) {
	w := New()
	if len(w.Option.StepInterceptors) != 0 {
		t.Fatalf("expected 0 StepInterceptor, got %d", len(w.Option.StepInterceptors))
	}
	if w.Option.MaxConcurrency != nil {
		t.Fatal("expected MaxConcurrency=nil")
	}
	if w.Option.DontPanic != nil {
		t.Fatal("expected DontPanic=nil")
	}
	if w.Option.StepDefaults != nil {
		t.Fatal("expected StepDefaults=nil")
	}
	if w.Option.DontInherit {
		t.Fatal("expected DontInherit=false")
	}
}

func TestLoggingStepInterceptor_LogsStepLifecycle(t *testing.T) {
	var buf bytes.Buffer
	logx.SetWriter(logx.NewWriter(&buf))
	defer logx.Reset()

	interceptor := LoggingStepInterceptor{}
	err := interceptor.InterceptStep(context.Background(),
		flow.Func("test-step", func(ctx context.Context) error { return nil }),
		func(ctx context.Context) error { return nil },
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[flowx] step done") {
		t.Fatalf("expected step done log, got: %s", output)
	}
}

func TestLoggingStepInterceptor_LogsError(t *testing.T) {
	var buf bytes.Buffer
	logx.SetWriter(logx.NewWriter(&buf))
	defer logx.Reset()

	interceptor := LoggingStepInterceptor{}
	err := interceptor.InterceptStep(context.Background(),
		flow.Func("bad-step", func(ctx context.Context) error { return nil }),
		func(ctx context.Context) error { return flow.Skip(nil) },
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	output := buf.String()
	if !strings.Contains(output, "[flowx] step failed") {
		t.Fatalf("expected step failed log, got: %s", output)
	}
}

func TestNew_WorkflowRunsSuccessfully(t *testing.T) {
	w := New(
		WithStepInterceptor(StepFields()),
		WithStepInterceptor(LoggingStepInterceptor{}),
	)
	counter := &countStep{}
	w.Add(flow.Step(counter))

	if err := w.Do(context.Background()); err != nil {
		t.Fatalf("workflow failed: %v", err)
	}
	if counter.count != 1 {
		t.Fatalf("expected step to run once, got %d", counter.count)
	}
}

func TestNew_WorkflowParallelSteps(t *testing.T) {
	w := New(
		WithStepInterceptor(StepFields()),
		WithStepInterceptor(LoggingStepInterceptor{}),
	)
	a := &countStep{}
	b := &countStep{}
	w.Add(flow.Steps(a, b))

	if err := w.Do(context.Background()); err != nil {
		t.Fatalf("workflow failed: %v", err)
	}
	if a.count != 1 || b.count != 1 {
		t.Fatalf("expected both steps to run, a=%d b=%d", a.count, b.count)
	}
}

func TestNew_WorkflowWithDependencies(t *testing.T) {
	w := New(
		WithStepInterceptor(StepFields()),
		WithStepInterceptor(LoggingStepInterceptor{}),
	)
	a := &countStep{}
	b := &countStep{}
	w.Add(flow.Step(b).DependsOn(a))

	if err := w.Do(context.Background()); err != nil {
		t.Fatalf("workflow failed: %v", err)
	}
	if a.count != 1 || b.count != 1 {
		t.Fatalf("expected both steps to run, a=%d b=%d", a.count, b.count)
	}
}

type countStep struct{ count int }

func (c *countStep) Do(ctx context.Context) error {
	c.count++
	return nil
}

type spyInterceptor struct{}

func (s *spyInterceptor) InterceptStep(ctx context.Context, step flow.Steper, next func(context.Context) error) error {
	return next(ctx)
}

type spyAttemptInterceptor struct{}

func (s *spyAttemptInterceptor) InterceptAttempt(ctx context.Context, step flow.Steper, attempt uint64, next func(context.Context) error) error {
	return next(ctx)
}
