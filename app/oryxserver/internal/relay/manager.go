package relay

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sync"

	"zero-service/common/tool"
)

// Task 转推任务（内存态，进程退出即清理）
type Task struct {
	TaskID string
	Source string          // 源流地址
	Target string          // 目标推流地址（rtmp://<srs>/<app>/<stream>）
	Ctx    context.Context // 进程绑定的可取消上下文（cancel() 即 kill 进程）
	Cancel context.CancelFunc
	Cmd    *exec.Cmd
	Stderr *bytes.Buffer // 捕获的 ffmpeg stderr，退出时打印
}

// Manager 转推任务管理器：sync.Map 管理 FFmpeg 进程，对业务侧屏蔽进程细节
type Manager struct {
	tasks sync.Map // taskID(string) -> *Task
	wg    sync.WaitGroup // 追踪所有 FFmpeg 进程，StopAll 时等待全部退出
}

// NewManager 创建转推任务管理器
func NewManager() *Manager {
	return &Manager{}
}

// Start 启动转推任务：从 source 拉流，copy 转推到 target，返回 task_id（非阻塞）
func (m *Manager) Start(source, target string) (string, error) {
	if source == "" {
		return "", errors.New("source is empty")
	}
	if target == "" {
		return "", errors.New("target is empty")
	}
	taskID, err := tool.SimpleUUID()
	if err != nil {
		return "", fmt.Errorf("generate task id failed: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	stderr := &bytes.Buffer{}
	cmd := buildFfmpegCmd(ctx, source, target, stderr)
	if err := cmd.Start(); err != nil {
		cancel()
		return "", fmt.Errorf("start ffmpeg failed: %w", err)
	}
	task := &Task{
		TaskID: taskID,
		Source: source,
		Target: target,
		Ctx:    ctx,
		Cancel: cancel,
		Cmd:    cmd,
		Stderr: stderr,
	}
	m.tasks.Store(taskID, task)
	m.wg.Add(1)
	m.watch(task)
	return taskID, nil
}

// Stop 停止指定转推任务；返回是否命中本地任务（未命中返回 false, nil）
func (m *Manager) Stop(taskID string) (bool, error) {
	if taskID == "" {
		return false, nil
	}
	v, ok := m.tasks.Load(taskID)
	if !ok {
		return false, nil
	}
	task, ok := v.(*Task)
	if !ok {
		return false, fmt.Errorf("invalid task type: %s", taskID)
	}
	m.tasks.Delete(taskID)
	task.Cancel()
	return true, nil
}

// TaskIDs 返回当前全部转推任务 ID
func (m *Manager) TaskIDs() []string {
	ids := make([]string, 0)
	m.tasks.Range(func(key, _ any) bool {
		ids = append(ids, key.(string))
		return true
	})
	return ids
}

// StopAll 停止全部转推任务并清理所有 FFmpeg 子进程（服务优雅退出时调用）。
// 逐个 Cancel（SIGKILL 终止进程）后，等待所有 FFmpeg 进程真正退出再返回，
// 避免服务退出时仍有残留推流进程或僵尸进程。
func (m *Manager) StopAll() {
	m.tasks.Range(func(key, value any) bool {
		task, ok := value.(*Task)
		if !ok {
			return true
		}
		m.tasks.Delete(key)
		task.Cancel()
		return true
	})
	m.wg.Wait()
}
