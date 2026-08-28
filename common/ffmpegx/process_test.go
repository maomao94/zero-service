package ffmpegx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"
)

func helperCommand(t *testing.T, mode string, args ...string) CommandBuilder {
	t.Helper()
	executable := testExecutable(t)
	return func(ctx context.Context) (*exec.Cmd, error) {
		commandArgs := []string{"-test.run=^TestFFmpegxHelperProcess$", "--", mode}
		commandArgs = append(commandArgs, args...)
		return exec.CommandContext(ctx, executable, commandArgs...), nil
	}
}

func testExecutable(t *testing.T) string {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	return path
}

func TestFFmpegxHelperProcess(t *testing.T) {
	for i, arg := range os.Args {
		if arg != "--" || i+1 >= len(os.Args) {
			continue
		}
		switch os.Args[i+1] {
		case "wait":
			time.Sleep(time.Hour)
		case "exit-error":
			os.Exit(7)
		case "stdout":
			_, _ = fmt.Fprint(os.Stdout, "ordinary-output")
		case "progress":
			_, _ = fmt.Fprint(os.Stdout, "frame=1\nprogress=continue\n")
		case "delayed-exit":
			time.Sleep(50 * time.Millisecond)
		}
		return
	}
}

func TestManagerStartHasCountAndStop(t *testing.T) {
	m := NewManager()
	if err := m.Start(context.Background(), "active", helperCommand(t, "wait")); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !m.Has("active") {
		t.Fatal("Has returned false for active process")
	}
	if got := m.Count(); got != 1 {
		t.Fatalf("Count = %d, want 1", got)
	}
	if !m.Stop("active") {
		t.Fatal("Stop returned false")
	}
	if m.Has("active") || m.Count() != 0 {
		t.Fatal("process remains registered after Stop")
	}
}

func TestManagerStartValidatesArguments(t *testing.T) {
	m := NewManager()
	tests := []struct {
		name  string
		ctx   context.Context
		id    string
		build CommandBuilder
	}{
		{name: "nil context", id: "id", build: helperCommand(t, "wait")},
		{name: "empty id", ctx: context.Background(), build: helperCommand(t, "wait")},
		{name: "nil builder", ctx: context.Background(), id: "id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := m.Start(tt.ctx, tt.id, tt.build); err == nil {
				t.Fatal("Start returned nil error")
			}
		})
	}
	if m.Count() != 0 {
		t.Fatalf("invalid starts registered %d processes", m.Count())
	}
}

func TestManagerStopMissingID(t *testing.T) {
	if NewManager().Stop("missing") {
		t.Fatal("Stop returned true for missing id")
	}
}

func TestManagerOrdinaryCommandDoesNotRequireStdoutPipe(t *testing.T) {
	m := NewManager()
	exit := make(chan ExitResult, 1)
	build := func(ctx context.Context) (*exec.Cmd, error) {
		cmd, err := helperCommand(t, "stdout")(ctx)
		if err != nil {
			return nil, err
		}
		cmd.Stdout = errWriter{}
		return cmd, nil
	}
	if err := m.Start(context.Background(), "ordinary", build, WithExitHandler(func(_ string, result ExitResult) {
		exit <- result
	})); err != nil {
		t.Fatalf("Start: %v", err)
	}
	select {
	case result := <-exit:
		if result.WaitErr == nil {
			t.Fatal("exit error is nil, want stdout write failure")
		}
	case <-time.After(StopWaitTimeout):
		t.Fatal("ordinary command exit handler did not run")
	}
}

var errWriterFailure = errors.New("writer failure")

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errWriterFailure }

func TestManagerStdoutHandlerExplicitlyConsumesStdout(t *testing.T) {
	m := NewManager()
	output := make(chan string, 2)
	exit := make(chan struct{}, 1)
	if err := m.Start(context.Background(), "progress", helperCommand(t, "progress"),
		WithStdoutHandler(func(_ string, line string) { output <- line }),
		WithExitHandler(func(string, ExitResult) { exit <- struct{}{} }),
	); err != nil {
		t.Fatalf("Start: %v", err)
	}
	select {
	case line := <-output:
		if line != "frame=1" {
			t.Fatalf("first stdout line = %q", line)
		}
	case <-time.After(StopWaitTimeout):
		t.Fatal("progress handler did not run")
	}
	select {
	case <-exit:
	case <-time.After(StopWaitTimeout):
		t.Fatal("process did not exit")
	}
}

func TestManagerImmediateUnexpectedExitCallsPerProcessHandler(t *testing.T) {
	m := NewManager()
	exit := make(chan ExitResult, 1)
	if err := m.Start(context.Background(), "quick", helperCommand(t, "exit-error"),
		WithExitHandler(func(_ string, result ExitResult) { exit <- result })); err != nil {
		t.Fatalf("Start: %v", err)
	}
	select {
	case result := <-exit:
		if result.WaitErr == nil || result.ContextErr != nil {
			t.Fatal("exit error is nil")
		}
	case <-time.After(StopWaitTimeout):
		t.Fatal("exit handler did not run")
	}
}

func TestManagerActiveStopPathsReportContextCancellation(t *testing.T) {
	tests := []struct {
		name string
		stop func(*Manager, context.CancelFunc)
	}{
		{name: "Stop", stop: func(m *Manager, _ context.CancelFunc) { m.Stop("active") }},
		{name: "StopAll", stop: func(m *Manager, _ context.CancelFunc) { m.StopAll() }},
		{name: "context cancel", stop: func(_ *Manager, cancel context.CancelFunc) { cancel() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager()
			ctx, cancel := context.WithCancel(context.Background())
			exit := make(chan ExitResult, 1)
			if err := m.Start(ctx, "active", helperCommand(t, "wait"),
				WithExitHandler(func(_ string, result ExitResult) { exit <- result })); err != nil {
				t.Fatalf("Start: %v", err)
			}
			tt.stop(m, cancel)
			waitUntil(t, StopWaitTimeout, func() bool { return !m.Has("active") })
			select {
			case result := <-exit:
				if !errors.Is(result.ContextErr, context.Canceled) {
					t.Fatalf("ContextErr = %v, want context.Canceled", result.ContextErr)
				}
			case <-time.After(StopWaitTimeout):
				t.Fatal("exit handler did not run")
			}
		})
	}
}

func TestManagerContextTimeoutReportsDeadline(t *testing.T) {
	m := NewManager()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	exit := make(chan ExitResult, 1)
	if err := m.Start(ctx, "timeout", helperCommand(t, "wait"),
		WithExitHandler(func(_ string, result ExitResult) { exit <- result })); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitUntil(t, StopWaitTimeout, func() bool { return !m.Has("timeout") })
	select {
	case result := <-exit:
		if !errors.Is(result.ContextErr, context.DeadlineExceeded) {
			t.Fatalf("ContextErr = %v, want context.DeadlineExceeded", result.ContextErr)
		}
		if result.WaitErr == nil {
			t.Fatal("WaitErr = nil, want process exit error")
		}
	case <-time.After(StopWaitTimeout):
		t.Fatal("exit handler did not run")
	}
}

func TestManagerDuplicateIDLeavesExistingProcessRunning(t *testing.T) {
	m := NewManager()
	if err := m.Start(context.Background(), "same", helperCommand(t, "wait")); err != nil {
		t.Fatalf("Start old: %v", err)
	}
	built := false
	err := m.Start(context.Background(), "same", func(context.Context) (*exec.Cmd, error) {
		built = true
		return nil, nil
	})
	if !errors.Is(err, ErrProcessExists) {
		t.Fatalf("Start duplicate error = %v, want ErrProcessExists", err)
	}
	if built {
		t.Fatal("duplicate Start invoked builder")
	}
	t.Cleanup(m.StopAll)
	if !m.Has("same") || m.Count() != 1 {
		t.Fatal("old watcher removed replacement")
	}
}

func TestManagerBuildsBeforeRegisteringProcess(t *testing.T) {
	m := NewManager()
	err := m.Start(context.Background(), "building", func(ctx context.Context) (*exec.Cmd, error) {
		if m.Has("building") {
			return nil, errors.New("process registered before builder returned")
		}
		return helperCommand(t, "wait")(ctx)
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	m.StopAll()
}

func TestManagerBuilderFailureLeavesNoProcess(t *testing.T) {
	m := NewManager()
	want := errors.New("build failed")
	err := m.Start(context.Background(), "failed", func(context.Context) (*exec.Cmd, error) {
		return nil, want
	})
	if !errors.Is(err, want) {
		t.Fatalf("Start error = %v, want %v", err, want)
	}
	if m.Has("failed") || m.Count() != 0 {
		t.Fatal("builder failure left registered process")
	}
}

func TestManagerBuilderReceivesParentDeadline(t *testing.T) {
	m := NewManager()
	deadline := time.Now().Add(time.Hour)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	err := m.Start(ctx, "deadline", func(processCtx context.Context) (*exec.Cmd, error) {
		got, ok := processCtx.Deadline()
		if !ok || !got.Equal(deadline) {
			return nil, errors.New("process context did not preserve parent deadline")
		}
		return helperCommand(t, "wait")(processCtx)
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	m.StopAll()
}

func TestManagerPipeAndStartFailuresLeaveNoProcess(t *testing.T) {
	tests := []struct {
		name  string
		build CommandBuilder
	}{
		{
			name: "stdout pipe",
			build: func(ctx context.Context) (*exec.Cmd, error) {
				cmd, err := helperCommand(t, "stdout", "-progress", "pipe:1")(ctx)
				if err != nil {
					return nil, err
				}
				cmd.Stdout = errWriter{}
				return cmd, nil
			},
		},
		{
			name: "command start",
			build: func(ctx context.Context) (*exec.Cmd, error) {
				return exec.CommandContext(ctx, "/path/that/does/not/exist"), nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager()
			err := m.Start(context.Background(), "failed", tt.build,
				WithStdoutHandler(func(string, string) {}))
			if err == nil {
				t.Fatal("Start returned nil error")
			}
			if m.Has("failed") || m.Count() != 0 {
				t.Fatal("start failure left registered process")
			}
		})
	}
}

func TestManagerExitHandlerCanReenterStopAll(t *testing.T) {
	m := NewManager()
	done := make(chan struct{})
	if err := m.Start(context.Background(), "reentrant", helperCommand(t, "stdout"),
		WithExitHandler(func(_ string, _ ExitResult) {
			if m.Has("reentrant") {
				t.Error("process remains registered during exit handler")
			}
			m.StopAll()
			close(done)
		})); err != nil {
		t.Fatalf("Start: %v", err)
	}
	select {
	case <-done:
	case <-time.After(StopWaitTimeout):
		t.Fatal("exit handler deadlocked while reentering StopAll")
	}
}

func TestManagerExitHandlerCompletesBeforeDoneCloses(t *testing.T) {
	m := NewManager()
	entered := make(chan struct{})
	release := make(chan struct{})
	if err := m.Start(context.Background(), "sync-exit", helperCommand(t, "delayed-exit"),
		WithExitHandler(func(_ string, _ ExitResult) {
			close(entered)
			<-release
		})); err != nil {
		t.Fatalf("Start: %v", err)
	}
	m.mu.RLock()
	proc := m.processes["sync-exit"]
	m.mu.RUnlock()
	select {
	case <-entered:
	case <-time.After(StopWaitTimeout):
		t.Fatal("exit handler did not run")
	}
	if m.Has("sync-exit") {
		t.Fatal("process remains registered during exit handler")
	}
	select {
	case <-proc.done:
		t.Fatal("done closed before exit handler completed")
	default:
	}
	close(release)
	select {
	case <-proc.done:
	case <-time.After(StopWaitTimeout):
		t.Fatal("done did not close after exit handler completed")
	}
}

func TestManagerStdoutHandlerCompletesBeforeProcessCleanup(t *testing.T) {
	m := NewManager()
	enteredCh := make(chan struct{})
	release := make(chan struct{})
	exit := make(chan struct{})
	var entered sync.Once
	if err := m.Start(context.Background(), "sync-progress", helperCommand(t, "progress"),
		WithStdoutHandler(func(string, string) {
			entered.Do(func() { close(enteredCh) })
			<-release
		}),
		WithExitHandler(func(string, ExitResult) { close(exit) })); err != nil {
		t.Fatalf("Start: %v", err)
	}
	select {
	case <-enteredCh:
	case <-time.After(StopWaitTimeout):
		t.Fatal("progress handler did not run")
	}
	if !m.Has("sync-progress") {
		t.Fatal("process cleaned up before progress handler completed")
	}
	select {
	case <-exit:
		t.Fatal("exit handler ran before progress handler completed")
	default:
	}
	close(release)
	select {
	case <-exit:
	case <-time.After(StopWaitTimeout):
		t.Fatal("process did not exit after progress handler completed")
	}
}

func TestManagerBlockedExitHandlerDoesNotBlockDifferentID(t *testing.T) {
	m := NewManager()
	entered := make(chan struct{})
	release := make(chan struct{})
	if err := m.Start(context.Background(), "slow-callback", helperCommand(t, "delayed-exit"),
		WithExitHandler(func(_ string, _ ExitResult) {
			close(entered)
			<-release
		})); err != nil {
		t.Fatalf("Start slow callback: %v", err)
	}
	select {
	case <-entered:
	case <-time.After(StopWaitTimeout):
		t.Fatal("exit handler did not run")
	}
	if err := m.Start(context.Background(), "independent", helperCommand(t, "wait")); err != nil {
		t.Fatalf("Start independent: %v", err)
	}
	if !m.Stop("independent") {
		t.Fatal("different ID was not independently stoppable")
	}
	close(release)
}

func TestManagerStopAllWaitsForWaitAndDoesNotDeadlock(t *testing.T) {
	m := NewManager()
	if err := m.Start(context.Background(), "wait", helperCommand(t, "wait")); err != nil {
		t.Fatalf("Start: %v", err)
	}
	done := make(chan struct{})
	go func() {
		m.StopAll()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(StopWaitTimeout + time.Second):
		t.Fatal("StopAll deadlocked")
	}
	if m.Count() != 0 {
		t.Fatal("StopAll returned before process cleanup")
	}
}

func waitUntil(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !fn() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met before timeout")
		}
		time.Sleep(time.Millisecond)
	}
}
