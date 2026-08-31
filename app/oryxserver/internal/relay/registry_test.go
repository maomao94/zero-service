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

// newTestRegistry 创建用于本地测试的 registry（无分布式依赖）。
func newTestRegistry(processes *ffmpegx.Manager, buildCmd relayCommandBuilder) *RelayRegistry {
	r := &RelayRegistry{
		processes: processes,
		buildCmd:  buildCmd,
		meta:      make(map[string]*pullMeta),
	}
	r.onProgressFunc = r.defaultOnProgress
	r.onProcessExitFunc = r.defaultOnProcessExit
	return r
}

func TestRelayRegistryFastExitCleansMetadata(t *testing.T) {
	processes := ffmpegx.NewManager()
	registry := newTestRegistry(processes, relayTestCommand(t, "exit-error"))
	exit := make(chan string, 1)
	origOnExit := registry.onProcessExitFunc
	registry.onProcessExitFunc = func(ctx context.Context, md *pullMeta, result ffmpegx.ExitResult) {
		pid := md.processID()
		if registry.HasTarget(md.App, md.Stream) {
			t.Errorf("metadata still exists when exit hook runs for %q", pid)
		}
		exit <- pid
		origOnExit(ctx, md, result)
	}

	app, stream := "live", "fast"
	if err := registry.startLocalRelay(context.Background(), "rtmp://source/live/test", app, stream, "test-uuid", "rtmp://localhost/live/fast"); err != nil {
		t.Fatalf("startLocalRelay: %v", err)
	}
	pid := app + ":" + stream
	select {
	case got := <-exit:
		if got != pid {
			t.Fatalf("exit pid = %q, want %q", got, pid)
		}
	case <-time.After(time.Second):
		t.Fatal("exit hook did not run")
	}
}

func TestRelayRegistryExitHookUsesDetachedContext(t *testing.T) {
	processes := ffmpegx.NewManager()
	registry := newTestRegistry(processes, relayTestCommand(t, "exit-error"))
	exitCtx := make(chan context.Context, 1)
	origOnExit := registry.onProcessExitFunc
	registry.onProcessExitFunc = func(ctx context.Context, md *pullMeta, result ffmpegx.ExitResult) {
		exitCtx <- ctx
		origOnExit(ctx, md, result)
	}

	requestCtx, cancelRequest := context.WithCancel(context.WithValue(context.Background(), testContextKey{}, "trace"))
	app, stream := "live", "context"
	if err := registry.startLocalRelay(requestCtx, "rtmp://source/live/test", app, stream, "test-uuid", "rtmp://localhost/live/context"); err != nil {
		t.Fatalf("startLocalRelay: %v", err)
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

func TestRelayRegistryStopLocalDoesNotCallExitHook(t *testing.T) {
	processes := ffmpegx.NewManager()
	registry := newTestRegistry(processes, relayTestCommand(t, "wait"))
	exit := make(chan string, 1)
	origOnExit := registry.onProcessExitFunc
	registry.onProcessExitFunc = func(ctx context.Context, md *pullMeta, result ffmpegx.ExitResult) {
		exit <- md.processID()
		origOnExit(ctx, md, result)
	}

	app, stream := "live", "stopped"
	if err := registry.startLocalRelay(context.Background(), "rtmp://source/live/test", app, stream, "test-uuid", "rtmp://localhost/live/stopped"); err != nil {
		t.Fatalf("startLocalRelay: %v", err)
	}
	if !registry.stopLocalRelay(app, stream) {
		t.Fatal("stopLocalRelay returned false")
	}
	if registry.HasTarget(app, stream) {
		t.Fatal("metadata remains after stopLocalRelay")
	}
	select {
	case got := <-exit:
		t.Fatalf("exit hook called for actively stopped pid %q", got)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestRelayRegistryStopPullSelectsAppAndStream(t *testing.T) {
	processes := ffmpegx.NewManager()
	registry := newTestRegistry(processes, relayTestCommand(t, "wait"))
	t.Cleanup(registry.StopAll)

	type target struct {
		app, stream string
	}
	targets := []target{
		{"live", "one"},
		{"live", "two"},
	}
	for _, tt := range targets {
		if err := registry.startLocalRelay(context.Background(), "rtmp://source/live/test", tt.app, tt.stream, "test-uuid", "rtmp://localhost/"+tt.app+"/"+tt.stream); err != nil {
			t.Fatalf("startLocalRelay(%s/%s): %v", tt.app, tt.stream, err)
		}
	}
	if !registry.StopRelayByAppStream("live", "one") {
		t.Fatal("StopPull returned false")
	}
	if registry.HasTarget("live", "one") {
		t.Fatal("matching target remains after StopPull")
	}
	if !registry.HasTarget("live", "two") {
		t.Fatal("non-matching target was stopped")
	}
}

func TestRelayRegistryOldExitDoesNotDeleteReplacement(t *testing.T) {
	registry := newTestRegistry(ffmpegx.NewManager(), nil)
	app, stream := "live", "replaced"
	pid := app + ":" + stream
	oldMeta := &pullMeta{App: app, Stream: stream}
	newMeta := &pullMeta{App: app, Stream: stream}
	registry.meta[pid] = newMeta
	exit := make(chan string, 1)
	origOnExit := registry.onProcessExitFunc
	registry.onProcessExitFunc = func(ctx context.Context, md *pullMeta, result ffmpegx.ExitResult) {
		exit <- md.processID()
		origOnExit(ctx, md, result)
	}

	registry.handleExit(context.Background(), oldMeta, ffmpegx.ExitResult{})

	if !registry.HasTarget(app, stream) {
		t.Fatal("old exit deleted replacement metadata")
	}
	select {
	case got := <-exit:
		t.Fatalf("old exit invoked business hook for replacement pid %q", got)
	default:
	}
}

func TestRelayRegistryStopAllStopsOnlyRelayTargets(t *testing.T) {
	processes := ffmpegx.NewManager()
	registry := newTestRegistry(processes, relayTestCommand(t, "wait"))
	if err := processes.Start(context.Background(), "unrelated", func(ctx context.Context) (*exec.Cmd, error) {
		return relayTestCommand(t, "wait")(ctx, "", ""), nil
	}); err != nil {
		t.Fatalf("start unrelated process: %v", err)
	}
	t.Cleanup(processes.StopAll)

	app, stream := "live", "owned"
	if err := registry.startLocalRelay(context.Background(), "rtmp://source/live/test", app, stream, "test-uuid", "rtmp://localhost/live/owned"); err != nil {
		t.Fatalf("startLocalRelay: %v", err)
	}
	registry.StopAll()

	pid := app + ":" + stream
	if registry.HasTarget(app, stream) || processes.Has(pid) {
		t.Fatal("relay target remains after registry StopAll")
	}
	if !processes.Has("unrelated") {
		t.Fatal("registry StopAll stopped unrelated ffmpeg process")
	}
}

func TestRelayRegistryProgressInvokesOnProgress(t *testing.T) {
	processes := ffmpegx.NewManager()
	registry := newTestRegistry(processes, relayTestCommand(t, "progress", "-progress", "pipe:1", "-stats_period", "5"))
	progress := make(chan string, 1)
	origOnProgress := registry.onProgressFunc
	registry.onProgressFunc = func(ctx context.Context, md *pullMeta) {
		progress <- md.processID()
		origOnProgress(ctx, md)
	}

	app, stream := "live", "progress"
	if err := registry.startLocalRelay(context.Background(), "rtmp://source/live/test", app, stream, "test-uuid", "rtmp://localhost/live/progress"); err != nil {
		t.Fatalf("startLocalRelay: %v", err)
	}
	pid := app + ":" + stream
	select {
	case got := <-progress:
		if got != pid {
			t.Fatalf("progress pid = %q, want %q", got, pid)
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
	registry := newTestRegistry(processes, relayTestCommand(t, "wait"))
	t.Cleanup(registry.StopAll)

	app, stream := "live", "test"
	if err := registry.startLocalRelay(context.Background(), "rtmp://source/live/test", app, stream, "test-uuid", "rtmp://localhost/live/test"); err != nil {
		t.Fatalf("startLocalRelay: %v", err)
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
	if !registry.HasTarget(app, stream) {
		t.Fatal("target should still exist after invalid StopPull calls")
	}
}

func TestStartLocalRelayRejectsEmptyParams(t *testing.T) {
	processes := ffmpegx.NewManager()
	registry := newTestRegistry(processes, relayTestCommand(t, "wait"))
	// Test with empty source
	err := registry.startLocalRelay(context.Background(), "", "live", "test", "uuid", "rtmp://localhost/live/test")
	if err == nil {
		t.Fatal("startLocalRelay with empty source should return error")
	}
	// Test with empty app
	err = registry.startLocalRelay(context.Background(), "rtmp://source/live/test", "", "test", "uuid", "rtmp://localhost/live/test")
	if err == nil {
		t.Fatal("startLocalRelay with empty app should return error")
	}
	// Test with empty stream
	err = registry.startLocalRelay(context.Background(), "rtmp://source/live/test", "live", "", "uuid", "rtmp://localhost/live/test")
	if err == nil {
		t.Fatal("startLocalRelay with empty stream should return error")
	}
	// Test with empty relayURL
	err = registry.startLocalRelay(context.Background(), "rtmp://source/live/test", "live", "test", "uuid", "")
	if err == nil {
		t.Fatal("startLocalRelay with empty relayURL should return error")
	}
}
