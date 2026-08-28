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
	registry.onProcessExitFunc = func(ctx context.Context, uid string, result ffmpegx.ExitResult) {
		if registry.HasTarget(uid) {
			t.Errorf("metadata still exists when exit hook runs for %q", uid)
		}
		exit <- uid
		origOnExit(ctx, uid, result)
	}

	const target = "rtmp://localhost/live/fast"
	uid, _ := CanonicalUID(target)
	if err := registry.startLocalRelay(context.Background(), "rtmp://source/live/test", uid, target); err != nil {
		t.Fatalf("startLocalRelay: %v", err)
	}
	select {
	case got := <-exit:
		if got != uid {
			t.Fatalf("exit uid = %q, want %q", got, uid)
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
	registry.onProcessExitFunc = func(ctx context.Context, uid string, result ffmpegx.ExitResult) {
		exitCtx <- ctx
		origOnExit(ctx, uid, result)
	}

	requestCtx, cancelRequest := context.WithCancel(context.WithValue(context.Background(), testContextKey{}, "trace"))
	const target = "rtmp://localhost/live/context"
	uid, _ := CanonicalUID(target)
	if err := registry.startLocalRelay(requestCtx, "rtmp://source/live/test", uid, target); err != nil {
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
	registry.onProcessExitFunc = func(ctx context.Context, uid string, result ffmpegx.ExitResult) {
		exit <- uid
		origOnExit(ctx, uid, result)
	}

	const target = "rtmp://localhost/live/stopped"
	uid, _ := CanonicalUID(target)
	if err := registry.startLocalRelay(context.Background(), "rtmp://source/live/test", uid, target); err != nil {
		t.Fatalf("startLocalRelay: %v", err)
	}
	if !registry.stopLocalRelay(uid) {
		t.Fatal("stopLocalRelay returned false")
	}
	if registry.HasTarget(uid) {
		t.Fatal("metadata remains after stopLocalRelay")
	}
	select {
	case got := <-exit:
		t.Fatalf("exit hook called for actively stopped uid %q", got)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestRelayRegistryStopPullSelectsAppAndStream(t *testing.T) {
	processes := ffmpegx.NewManager()
	registry := newTestRegistry(processes, relayTestCommand(t, "wait"))
	t.Cleanup(registry.StopAll)

	targets := []string{
		"rtmp://localhost/live/one",
		"rtmp://localhost/live/two",
	}
	uids := make([]string, len(targets))
	for i, target := range targets {
		uid, _ := CanonicalUID(target)
		uids[i] = uid
		if err := registry.startLocalRelay(context.Background(), "rtmp://source/live/test", uid, target); err != nil {
			t.Fatalf("startLocalRelay(%q): %v", target, err)
		}
	}
	if !registry.StopRelayByAppStream("live", "one") {
		t.Fatal("StopPull returned false")
	}
	if registry.HasTarget(uids[0]) {
		t.Fatal("matching uid remains after StopPull")
	}
	if !registry.HasTarget(uids[1]) {
		t.Fatal("non-matching uid was stopped")
	}
}

func TestRelayRegistryOldExitDoesNotDeleteReplacement(t *testing.T) {
	registry := newTestRegistry(ffmpegx.NewManager(), nil)
	const target = "rtmp://localhost/live/replaced"
	uid, _ := CanonicalUID(target)
	oldMeta := &pullMeta{UID: uid}
	newMeta := &pullMeta{UID: uid}
	registry.meta[uid] = newMeta
	exit := make(chan string, 1)
	origOnExit := registry.onProcessExitFunc
	registry.onProcessExitFunc = func(ctx context.Context, uid string, result ffmpegx.ExitResult) {
		exit <- uid
		origOnExit(ctx, uid, result)
	}

	registry.handleExit(context.Background(), oldMeta, ffmpegx.ExitResult{})

	if !registry.HasTarget(uid) {
		t.Fatal("old exit deleted replacement metadata")
	}
	select {
	case got := <-exit:
		t.Fatalf("old exit invoked business hook for replacement uid %q", got)
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

	const target = "rtmp://localhost/live/owned"
	uid, _ := CanonicalUID(target)
	if err := registry.startLocalRelay(context.Background(), "rtmp://source/live/test", uid, target); err != nil {
		t.Fatalf("startLocalRelay: %v", err)
	}
	registry.StopAll()

	if registry.HasTarget(uid) || processes.Has(uid) {
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
	registry.onProgressFunc = func(ctx context.Context, uid string) {
		progress <- uid
		origOnProgress(ctx, uid)
	}

	const target = "rtmp://localhost/live/progress"
	uid, _ := CanonicalUID(target)
	if err := registry.startLocalRelay(context.Background(), "rtmp://source/live/test", uid, target); err != nil {
		t.Fatalf("startLocalRelay: %v", err)
	}
	select {
	case got := <-progress:
		if got != uid {
			t.Fatalf("progress uid = %q, want %q", got, uid)
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

func TestCanonicalUID(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		wantErr bool
	}{
		{"full url", "rtmp://127.0.0.1:1935/live/stream?secret=abc", "127.0.0.1_1935/live/stream", false},
		{"normalized endpoint", "127.0.0.1:1935/live/stream", "127.0.0.1_1935/live/stream", false},
		{"uid", "127.0.0.1_1935/live/stream", "127.0.0.1_1935/live/stream", false},
		{"invalid path", "rtmp://127.0.0.1:1935/live", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CanonicalUID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CanonicalUID(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("CanonicalUID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
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

	const target = "rtmp://localhost/live/test"
	uid, _ := CanonicalUID(target)
	if err := registry.startLocalRelay(context.Background(), "rtmp://source/live/test", uid, target); err != nil {
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
	if !registry.HasTarget(uid) {
		t.Fatal("uid should still exist after invalid StopPull calls")
	}
}

func TestStartLocalRelayRejectsInvalidTarget(t *testing.T) {
	processes := ffmpegx.NewManager()
	registry := newTestRegistry(processes, relayTestCommand(t, "wait"))
	invalidTargets := []string{
		"rtmp://localhost",
		"rtmp://localhost/live",
		"rtmp://localhost/live/",
		"rtmp://localhost/a/b/c",
		"rtmp://localhost/live/a/b/c",
	}
	for _, target := range invalidTargets {
		uid, err := CanonicalUID(target)
		if err != nil {
			continue // invalid target, CanonicalUID rejects
		}
		err = registry.startLocalRelay(context.Background(), "rtmp://source/live/test", uid, target)
		if err == nil {
			t.Fatalf("startLocalRelay(%q) expected error, got nil", target)
		}
	}
}
