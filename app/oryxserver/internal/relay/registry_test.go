package relay

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"zero-service/common/ffmpegx"
)

func relayTestCommand(t *testing.T, mode string, ffmpegArgs ...string) func(context.Context, string, string) *exec.Cmd {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	return func(ctx context.Context, _, _ string) *exec.Cmd {
		args := []string{"-test.run=^TestRelayHelperProcess$", "--", mode}
		args = append(args, ffmpegArgs...)
		return exec.CommandContext(ctx, executable, args...)
	}
}

func TestRelayHelperProcess(t *testing.T) {
	for i, arg := range os.Args {
		if arg != "--" || i+1 >= len(os.Args) {
			continue
		}
		switch os.Args[i+1] {
		case "wait":
			time.Sleep(time.Hour)
		case "exit-error":
			os.Exit(7)
		case "progress":
			_, _ = fmt.Fprint(os.Stdout, "frame=1\nprogress=continue\n")
		}
		return
	}
}

func TestRelayRegistryFastExitCleansMetadataBeforeHook(t *testing.T) {
	processes := ffmpegx.NewManager()
	registry := newRelayRegistry(processes, relayTestCommand(t, "exit-error"))
	exit := make(chan string, 1)
	registry.SetExitHandler(func(_ context.Context, target string, _ ffmpegx.ExitResult) {
		if registry.HasTarget(target) {
			t.Errorf("metadata still exists when exit hook runs for %q", target)
		}
		exit <- target
	})

	const target = "rtmp://localhost/live/fast"
	if err := registry.StartRelay(context.Background(), "rtmp://source/live/test", target, target); err != nil {
		t.Fatalf("StartRelay: %v", err)
	}
	select {
	case got := <-exit:
		if got != NormalizeTarget(target) {
			t.Fatalf("exit target = %q, want %q", got, NormalizeTarget(target))
		}
	case <-time.After(time.Second):
		t.Fatal("exit hook did not run")
	}
}

func TestRelayRegistryExitHookUsesDetachedContext(t *testing.T) {
	processes := ffmpegx.NewManager()
	registry := newRelayRegistry(processes, relayTestCommand(t, "exit-error"))
	exitCtx := make(chan context.Context, 1)
	registry.SetExitHandler(func(ctx context.Context, _ string, _ ffmpegx.ExitResult) { exitCtx <- ctx })

	requestCtx, cancelRequest := context.WithCancel(context.WithValue(context.Background(), testContextKey{}, "trace"))
	const target = "rtmp://localhost/live/context"
	if err := registry.StartRelay(requestCtx, "rtmp://source/live/test", target, target); err != nil {
		t.Fatalf("StartRelay: %v", err)
	}
	cancelRequest()
	select {
	case ctx := <-exitCtx:
		if err := ctx.Err(); err != nil {
			t.Fatalf("exit hook context is canceled: %v", err)
		}
		if got := ctx.Value(testContextKey{}); got != "trace" {
			t.Fatalf("exit hook context value = %v, want trace", got)
		}
	case <-time.After(time.Second):
		t.Fatal("exit hook did not run")
	}
}

type testContextKey struct{}

func TestRelayRegistryStopRelayDoesNotCallExitHook(t *testing.T) {
	processes := ffmpegx.NewManager()
	registry := newRelayRegistry(processes, relayTestCommand(t, "wait"))
	exit := make(chan string, 1)
	registry.SetExitHandler(func(_ context.Context, target string, _ ffmpegx.ExitResult) { exit <- target })

	const target = "rtmp://localhost/live/stopped"
	if err := registry.StartRelay(context.Background(), "rtmp://source/live/test", target, target); err != nil {
		t.Fatalf("StartRelay: %v", err)
	}
	if !registry.StopRelay(target) {
		t.Fatal("StopRelay returned false")
	}
	if registry.HasTarget(target) {
		t.Fatal("metadata remains after StopRelay")
	}
	select {
	case got := <-exit:
		t.Fatalf("exit hook called for actively stopped target %q", got)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestRelayRegistryStopPullSelectsAppAndStream(t *testing.T) {
	processes := ffmpegx.NewManager()
	registry := newRelayRegistry(processes, relayTestCommand(t, "wait"))
	t.Cleanup(registry.StopAll)

	targets := []string{
		"rtmp://localhost/live/one",
		"rtmp://localhost/live/two",
	}
	for _, target := range targets {
		if err := registry.StartRelay(context.Background(), "rtmp://source/live/test", target, target); err != nil {
			t.Fatalf("StartRelay(%q): %v", target, err)
		}
	}
	if !registry.StopRelayByAppStream("live", "one") {
		t.Fatal("StopPull returned false")
	}
	if registry.HasTarget(targets[0]) {
		t.Fatal("matching target remains after StopPull")
	}
	if !registry.HasTarget(targets[1]) {
		t.Fatal("non-matching target was stopped")
	}
}

func TestRelayRegistryOldExitDoesNotDeleteReplacement(t *testing.T) {
	registry := newRelayRegistry(ffmpegx.NewManager(), nil)
	const target = "rtmp://localhost/live/replaced"
	normTarget := NormalizeTarget(target)
	oldMeta := &pullMeta{Target: normTarget}
	newMeta := &pullMeta{Target: normTarget}
	registry.meta[normTarget] = newMeta
	exit := make(chan string, 1)
	registry.SetExitHandler(func(_ context.Context, target string, _ ffmpegx.ExitResult) { exit <- target })

	registry.handleExit(context.Background(), oldMeta, ffmpegx.ExitResult{ID: normTarget})

	if !registry.HasTarget(target) {
		t.Fatal("old exit deleted replacement metadata")
	}
	select {
	case got := <-exit:
		t.Fatalf("old exit invoked business hook for replacement target %q", got)
	default:
	}
}

func TestRelayRegistryStopAllStopsOnlyRelayTargets(t *testing.T) {
	processes := ffmpegx.NewManager()
	registry := newRelayRegistry(processes, relayTestCommand(t, "wait"))
	if err := processes.Start(context.Background(), "unrelated", func(ctx context.Context) (*exec.Cmd, error) {
		return relayTestCommand(t, "wait")(ctx, "", ""), nil
	}); err != nil {
		t.Fatalf("start unrelated process: %v", err)
	}
	t.Cleanup(processes.StopAll)

	const target = "rtmp://localhost/live/owned"
	if err := registry.StartRelay(context.Background(), "rtmp://source/live/test", target, target); err != nil {
		t.Fatalf("StartRelay: %v", err)
	}
	registry.StopAll()

	if registry.HasTarget(target) || processes.Has(target) {
		t.Fatal("relay target remains after registry StopAll")
	}
	if !processes.Has("unrelated") {
		t.Fatal("registry StopAll stopped unrelated ffmpeg process")
	}
}

func TestRelayRegistryProgressInvokesHookForRelayCommand(t *testing.T) {
	processes := ffmpegx.NewManager()
	registry := newRelayRegistry(processes, relayTestCommand(t, "progress", "-progress", "pipe:1", "-stats_period", "5"))
	progress := make(chan string, 1)
	registry.SetProgressHandler(func(_ context.Context, target string) { progress <- target })

	const target = "rtmp://localhost/live/progress"
	if err := registry.StartRelay(context.Background(), "rtmp://source/live/test", target, target); err != nil {
		t.Fatalf("StartRelay: %v", err)
	}
	select {
	case got := <-progress:
		if got != NormalizeTarget(target) {
			t.Fatalf("progress target = %q, want %q", got, NormalizeTarget(target))
		}
	case <-time.After(time.Second):
		t.Fatal("relay progress hook did not run")
	}
}

func TestRemainingDurationUsesPersistedDeadline(t *testing.T) {
	deadline := time.Now().Add(2 * time.Second).Unix()
	got := remainingDuration(deadline)
	if got <= 0 || got > 2*time.Second {
		t.Fatalf("remaining duration = %v, want positive duration no greater than two seconds", got)
	}
}

func TestExpiredDeadlineIsNotRunnable(t *testing.T) {
	if !deadlineExpired(time.Now().Add(-time.Second).Unix()) {
		t.Fatal("past deadline was considered runnable")
	}
	if deadlineExpired(0) {
		t.Fatal("unlimited deadline was considered expired")
	}
}

func TestNormalizeTarget(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"rtmp://host:1935/live/stream", "host:1935/live/stream"},
		{"http://host:1935/live/stream", "host:1935/live/stream"},
		{"rtmp://host:1935/live/stream?secret=abc", "host:1935/live/stream"},
		{"rtmp://host/live/test", "host/live/test"},
		{"host:1935/live/stream", "host:1935/live/stream"},
	}
	for _, tt := range tests {
		if got := NormalizeTarget(tt.input); got != tt.want {
			t.Errorf("NormalizeTarget(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseTargetValid(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		wantApp    string
		wantStream string
	}{
		{"with port", "host:1935/live/stream1", "live", "stream1"},
		{"no port", "host/live/test", "live", "test"},
		{"with query", "host:1935/live/stream1?secret=abc", "live", "stream1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, stream, err := ParseTarget(tt.target)
			if err != nil {
				t.Fatalf("ParseTarget(%q) unexpected error: %v", tt.target, err)
			}
			if app != tt.wantApp || stream != tt.wantStream {
				t.Fatalf("ParseTarget(%q) = (%q, %q), want (%q, %q)", tt.target, app, stream, tt.wantApp, tt.wantStream)
			}
		})
	}
}

func TestParseTargetInvalid(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{"no path", "host:1935"},
		{"only app", "host:1935/live"},
		{"trailing slash", "host:1935/live/"},
		{"empty app", "host:1935//stream"},
		{"empty stream", "host:1935/live/"},
		{"three segments", "host:1935/a/b/c"},
		{"four segments", "host:1935/a/b/c/d"},
		{"app with slash", "host:1935/a/b/stream"},
		{"stream with slash", "host:1935/app/b/c"},
		{"empty host", "/live/stream"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseTarget(tt.target)
			if err == nil {
				t.Fatalf("ParseTarget(%q) expected error, got nil", tt.target)
			}
		})
	}
}

func TestStopPullRejectsEmptyAppOrStream(t *testing.T) {
	processes := ffmpegx.NewManager()
	registry := newRelayRegistry(processes, relayTestCommand(t, "wait"))
	t.Cleanup(registry.StopAll)

	const target = "rtmp://localhost/live/test"
	if err := registry.StartRelay(context.Background(), "rtmp://source/live/test", target, target); err != nil {
		t.Fatalf("StartRelay: %v", err)
	}
	if registry.StopRelayByAppStream("", "test") {
		t.Fatal("StopPull with empty app should return false")
	}
	if registry.StopRelayByAppStream("live", "") {
		t.Fatal("StopPull with empty stream should return false")
	}
	if registry.StopRelayByAppStream("", "") {
		t.Fatal("StopPull with both empty should return false")
	}
	if !registry.HasTarget(target) {
		t.Fatal("target should still exist after invalid StopPull calls")
	}
}

func TestStartRelayRejectsInvalidTarget(t *testing.T) {
	processes := ffmpegx.NewManager()
	registry := newRelayRegistry(processes, relayTestCommand(t, "wait"))
	invalidTargets := []string{
		"rtmp://localhost",
		"rtmp://localhost/live",
		"rtmp://localhost/live/",
		"rtmp://localhost/a/b/c",
		"rtmp://localhost/live/a/b/c",
	}
	for _, target := range invalidTargets {
		err := registry.StartRelay(context.Background(), "rtmp://source/live/test", target, target)
		if err == nil {
			t.Fatalf("StartRelay(%q) expected error, got nil", target)
		}
	}
}
