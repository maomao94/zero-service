package ffmpegx

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// StopWaitTimeout limits how long Stop waits for a process to exit.
	StopWaitTimeout = 3 * time.Second
)

// CommandBuilder creates an unstarted command bound to ctx. Builders must
// return quickly and must not start the command themselves.
type CommandBuilder func(ctx context.Context) (*exec.Cmd, error)

// StartOption configures one process start.
type StartOption func(*startOptions)

type startOptions struct {
	onExit   func(id string, result ExitResult)
	onStdout func(id, line string)
	onStderr func(id, line string)
}

// ExitResult reports both the command's wait result and process context state.
type ExitResult struct {
	WaitErr    error
	ContextErr error
}

// WithExitHandler returns an option that synchronously invokes fn after
// process cleanup. The process is no longer registered when fn runs.
func WithExitHandler(fn func(id string, result ExitResult)) StartOption {
	return func(options *startOptions) { options.onExit = fn }
}

// WithStdoutHandler returns an option that synchronously invokes fn for each
// stdout line. The callback must not synchronously Stop its own ID because the
// reader must finish before that process can Wait.
func WithStdoutHandler(fn func(id, line string)) StartOption {
	return func(options *startOptions) { options.onStdout = fn }
}

// WithStderrHandler returns an option that synchronously invokes fn for each
// stderr line. Useful for long-running processes where buffering stderr would
// cause unbounded memory growth. The callback must not synchronously Stop its
// own ID.
func WithStderrHandler(fn func(id, line string)) StartOption {
	return func(options *startOptions) { options.onStderr = fn }
}

type process struct {
	id       string
	ctx      context.Context
	cmd      *exec.Cmd
	cancel   context.CancelFunc
	handlers startOptions
	done     chan struct{}
}

// Manager manages commands by ID. Starting an existing ID returns
// ErrProcessExists; callers explicitly Stop before replacement. Different IDs
// are independent.
// Construct a Manager with NewManager; its zero value is not ready for use.
type Manager struct {
	mu        sync.RWMutex
	processes map[string]*process
}

// NewManager creates an empty process manager.
func NewManager() *Manager {
	return &Manager{processes: make(map[string]*process)}
}

// Start builds and starts a command. Manager derives the process context from ctx.
// Callers must serialize Start/Stop for the same ID; different IDs are independent.
func (m *Manager) Start(ctx context.Context, id string, build CommandBuilder, opts ...StartOption) error {
	if ctx == nil {
		return fmt.Errorf("context must not be nil")
	}
	if id == "" {
		return fmt.Errorf("process id must not be empty")
	}
	if build == nil {
		return fmt.Errorf("command builder must not be nil")
	}

	var options startOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	m.mu.Lock()
	_, exists := m.processes[id]
	m.mu.Unlock()
	if exists {
		return fmt.Errorf("process %q: %w", id, ErrProcessExists)
	}

	processCtx, cancel := context.WithCancel(ctx)
	cmd, err := build(processCtx)
	if err != nil {
		cancel()
		return fmt.Errorf("build process: %w", err)
	}
	if cmd == nil {
		cancel()
		return fmt.Errorf("build process: nil command")
	}

	var stdout io.ReadCloser
	if options.onStdout != nil {
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			cancel()
			return fmt.Errorf("attach stdout pipe: %w", err)
		}
	}
	var stderr io.ReadCloser
	if options.onStderr != nil {
		stderr, err = cmd.StderrPipe()
		if err != nil {
			cancel()
			if stdout != nil {
				_ = stdout.Close()
			}
			return fmt.Errorf("attach stderr pipe: %w", err)
		}
	}
	proc := &process{
		id:       id,
		ctx:      processCtx,
		cmd:      cmd,
		cancel:   cancel,
		handlers: options,
		done:     make(chan struct{}),
	}
	m.mu.Lock()
	m.processes[id] = proc
	m.mu.Unlock()
	if err := cmd.Start(); err != nil {
		m.removeProcess(proc)
		if stdout != nil {
			_ = stdout.Close()
		}
		if stderr != nil {
			_ = stderr.Close()
		}
		cancel()
		return fmt.Errorf("start process: %w", err)
	}
	logger := logx.WithContext(processCtx)
	logger.Infof("[ffmpegx] process started: id=%s cmd=%s", id, strings.Join(cmd.Args, " "))
	go m.watchProcess(processCtx, proc, stdout, stderr)
	return nil
}

// Stop actively stops the process with id and waits up to StopWaitTimeout for
// its Wait call and synchronous callbacks to finish. It reports whether id existed.
func (m *Manager) Stop(id string) bool {
	proc := m.remove(id)
	if proc == nil {
		return false
	}
	logx.WithContext(proc.ctx).Infof("[ffmpegx] stopping process: id=%s", id)
	proc.cancel()
	m.waitProcess(proc)
	return true
}

// StopAll actively stops all managed processes and waits up to StopWaitTimeout
// per process for its Wait call and synchronous callbacks to finish.
func (m *Manager) StopAll() {
	m.mu.Lock()
	processes := make([]*process, 0, len(m.processes))
	for id, proc := range m.processes {
		delete(m.processes, id)
		processes = append(processes, proc)
	}
	m.mu.Unlock()
	for _, proc := range processes {
		proc.cancel()
	}
	for _, proc := range processes {
		m.waitProcess(proc)
	}
}

// Has reports whether id is registered as active.
func (m *Manager) Has(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.processes[id]
	return exists
}

// Count returns the number of active processes.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.processes)
}

func (m *Manager) remove(id string) *process {
	m.mu.Lock()
	proc := m.processes[id]
	if proc != nil {
		delete(m.processes, id)
	}
	m.mu.Unlock()
	return proc
}

func (m *Manager) removeProcess(proc *process) {
	m.mu.Lock()
	if m.processes[proc.id] == proc {
		delete(m.processes, proc.id)
	}
	m.mu.Unlock()
}

func (m *Manager) waitProcess(proc *process) {
	select {
	case <-proc.done:
	case <-time.After(StopWaitTimeout):
		logx.Errorf("[ffmpegx] process stop wait timeout: id=%s", proc.id)
	}
}

func (m *Manager) watchProcess(processCtx context.Context, proc *process, stdout, stderr io.ReadCloser) {
	logger := logx.WithContext(processCtx)

	// Scan stdout and stderr concurrently — one pipe blocking must not starve the other.
	var wg sync.WaitGroup
	if stdout != nil {
		wg.Go(func() {
			if err := WatchOutput(stdout, func(line string) {
				proc.handlers.onStdout(proc.id, line)
			}); err != nil && processCtx.Err() == nil {
				logger.Errorf("[ffmpegx] read process stdout: id=%s err=%v", proc.id, err)
			}
		})
	}
	if stderr != nil {
		wg.Go(func() {
			if err := WatchOutput(stderr, func(line string) {
				proc.handlers.onStderr(proc.id, line)
			}); err != nil && processCtx.Err() == nil {
				logger.Errorf("[ffmpegx] read process stderr: id=%s err=%v", proc.id, err)
			}
		})
	}
	wg.Wait()

	// All pipe readers done — safe to call Wait().
	waitErr := proc.cmd.Wait()
	contextErr := processCtx.Err()
	m.removeProcess(proc)
	if contextErr != nil {
		logger.Infof("[ffmpegx] process stopped: id=%s reason=%v", proc.id, contextErr)
	} else {
		if waitErr == nil {
			logger.Infof("[ffmpegx] process ended: id=%s", proc.id)
		} else {
			logger.Errorf("[ffmpegx] process exited with error: id=%s err=%v", proc.id, waitErr)
		}
	}
	if proc.handlers.onExit != nil {
		proc.handlers.onExit(proc.id, ExitResult{WaitErr: waitErr, ContextErr: contextErr})
	}
	proc.cancel()
	close(proc.done)
}
