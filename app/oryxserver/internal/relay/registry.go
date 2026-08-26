package relay

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/common/ffmpegx"
)

type relayCommandBuilder func(ctx context.Context, source, relayURL string) *exec.Cmd

// RelayRegistry owns relay metadata and maps relay operations to ffmpeg processes.
type RelayRegistry struct {
	processes *ffmpegx.Manager
	buildCmd  relayCommandBuilder

	lifecycleMu sync.Mutex
	metaMu      sync.RWMutex
	meta        map[string]*pullMeta
	onExit      func(ctx context.Context, target string, result ffmpegx.ExitResult)
	onProgress  func(ctx context.Context, target string)
}

type pullMeta struct {
	Source, Target, RelayURL, App, Stream string
}

// NewRelayRegistry creates a relay registry backed by the process manager.
func NewRelayRegistry(processes *ffmpegx.Manager) *RelayRegistry {
	return newRelayRegistry(processes, func(ctx context.Context, source, relayURL string) *exec.Cmd {
		return ffmpegx.BuildRelayCmd(ctx, source, relayURL)
	})
}

func newRelayRegistry(processes *ffmpegx.Manager, buildCmd relayCommandBuilder) *RelayRegistry {
	return &RelayRegistry{
		processes: processes,
		buildCmd:  buildCmd,
		meta:      make(map[string]*pullMeta),
	}
}

// SetExitHandler sets the relay hook invoked after unexpected-exit metadata cleanup.
func (r *RelayRegistry) SetExitHandler(fn func(ctx context.Context, target string, result ffmpegx.ExitResult)) {
	r.metaMu.Lock()
	r.onExit = fn
	r.metaMu.Unlock()
}

// SetProgressHandler sets the relay hook invoked for process progress frames.
func (r *RelayRegistry) SetProgressHandler(fn func(ctx context.Context, target string)) {
	r.metaMu.Lock()
	r.onProgress = fn
	r.metaMu.Unlock()
}

// StartRelay detaches the relay lifetime from ctx cancellation, registers
// metadata, and starts an ffmpeg command that writes progress to stdout.
func (r *RelayRegistry) StartRelay(ctx context.Context, source, target, relayURL string, maxDuration ...time.Duration) error {
	if source == "" || target == "" || relayURL == "" {
		return fmt.Errorf("source/target/relayURL must not be empty")
	}

	// 标准化 target：去掉 scheme，统一唯一键格式
	target = NormalizeTarget(target)

	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	r.metaMu.RLock()
	if _, exists := r.meta[target]; exists && r.processes.Has(target) {
		r.metaMu.RUnlock()
		return nil
	}
	r.metaMu.RUnlock()

	eventCtx := context.WithoutCancel(ctx)
	app, stream, err := ParseTarget(target)
	if err != nil {
		return fmt.Errorf("invalid target: %w", err)
	}
	md := &pullMeta{Source: source, Target: target, RelayURL: relayURL, App: app, Stream: stream}
	r.metaMu.Lock()
	r.meta[target] = md
	r.metaMu.Unlock()
	build := func(processCtx context.Context) (*exec.Cmd, error) {
		cmd := r.buildCmd(processCtx, source, relayURL)
		return cmd, nil
	}
	processCtx := eventCtx
	var durationCancel context.CancelFunc
	if len(maxDuration) > 0 && maxDuration[0] > 0 {
		processCtx, durationCancel = context.WithTimeout(eventCtx, maxDuration[0])
	}
	if err := r.processes.Start(processCtx, target, build,
		ffmpegx.WithStdoutHandler(func(_ string, line string) {
			r.handleOutput(eventCtx, md, line)
		}),
		ffmpegx.WithStderrHandler(func(id, line string) {
			r.handleStderr(eventCtx, id, line)
		}),
		ffmpegx.WithExitHandler(func(result ffmpegx.ExitResult) {
			if durationCancel != nil {
				defer durationCancel()
			}
			r.handleExit(eventCtx, md, result)
		}),
	); err != nil {
		if durationCancel != nil {
			durationCancel()
		}
		r.metaMu.Lock()
		if r.meta[target] == md {
			delete(r.meta, target)
		}
		r.metaMu.Unlock()
		return err
	}
	return nil
}

// HasTarget reports whether target has registered relay metadata.
func (r *RelayRegistry) HasTarget(target string) bool {
	target = NormalizeTarget(target)
	r.metaMu.RLock()
	defer r.metaMu.RUnlock()
	_, exists := r.meta[target]
	return exists
}

// StopRelay removes metadata and actively stops the target process.
func (r *RelayRegistry) StopRelay(target string) bool {
	target = NormalizeTarget(target)
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	r.metaMu.Lock()
	if _, exists := r.meta[target]; !exists {
		r.metaMu.Unlock()
		return false
	}
	delete(r.meta, target)
	r.metaMu.Unlock()
	return r.processes.Stop(target)
}

// StopRelayByAppStream stops local relays matching app and stream.
func (r *RelayRegistry) StopRelayByAppStream(app, stream string) bool {
	if app == "" || stream == "" {
		return false
	}
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	r.metaMu.Lock()
	var targets []string
	for target, md := range r.meta {
		if md.App == app && md.Stream == stream {
			targets = append(targets, target)
		}
	}
	for _, target := range targets {
		delete(r.meta, target)
	}
	r.metaMu.Unlock()
	found := false
	for _, target := range targets {
		if r.processes.Stop(target) {
			found = true
		}
	}
	return found
}

// StopAll removes and stops only processes owned by this relay registry.
func (r *RelayRegistry) StopAll() {
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	r.metaMu.Lock()
	targets := make([]string, 0, len(r.meta))
	for target := range r.meta {
		targets = append(targets, target)
	}
	clear(r.meta)
	r.metaMu.Unlock()
	for _, target := range targets {
		r.processes.Stop(target)
	}
}

func (r *RelayRegistry) handleOutput(ctx context.Context, md *pullMeta, line string) {
	key, value, ok := strings.Cut(line, "=")
	if !ok || key == "" {
		return
	}
	r.metaMu.RLock()
	if r.meta[md.Target] != md {
		r.metaMu.RUnlock()
		return
	}
	fn := r.onProgress
	r.metaMu.RUnlock()
	logx.WithContext(ctx).Debugf("[relay] stdout: target=%s key=%s value=%s", md.Target, key, value)
	if key == "progress" && value == "continue" && fn != nil {
		fn(ctx, md.Target)
	}
}

func (r *RelayRegistry) handleExit(ctx context.Context, md *pullMeta, result ffmpegx.ExitResult) {
	r.metaMu.Lock()
	if r.meta[md.Target] != md {
		r.metaMu.Unlock()
		logx.WithContext(ctx).Debugf("[relay] handleExit skipped (stale meta): target=%s", md.Target)
		return
	}
	delete(r.meta, md.Target)
	fn := r.onExit
	r.metaMu.Unlock()
	logx.WithContext(ctx).Debugf("[relay] handleExit: target=%s waitErr=%v contextErr=%v", md.Target, result.WaitErr, result.ContextErr)
	if fn != nil {
		fn(ctx, md.Target, result)
	}
}

func (r *RelayRegistry) handleStderr(ctx context.Context, id, line string) {
	r.metaMu.RLock()
	md := r.meta[id]
	r.metaMu.RUnlock()
	if md == nil {
		return
	}
	logx.WithContext(ctx).Debugf("[ffmpegx] stderr: id=%s line=%s", id, line)
}
