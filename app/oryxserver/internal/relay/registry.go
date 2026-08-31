package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"zero-service/common/carbonx"
	"zero-service/common/ffmpegx"
	"zero-service/common/tool"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
)

type relayCommandBuilder func(ctx context.Context, source, relayURL string) *exec.Cmd

// RelayRegistry 管理 relay 进程生命周期 + 分布式协调（Redis state/lease + Asynq 补拉/补停）。
type RelayRegistry struct {
	// 本地进程管理
	processes *ffmpegx.Manager
	buildCmd  relayCommandBuilder

	lifecycleMu sync.Mutex
	metaMu      sync.RWMutex
	meta        map[string]*pullMeta

	// 分布式协调
	store  *Store
	nodeID string
	asynq  *asynq.Client

	// 内部回调（可测试覆盖）
	onProgressFunc    func(ctx context.Context, md *pullMeta)
	onProcessExitFunc func(ctx context.Context, md *pullMeta, result ffmpegx.ExitResult)
}

type pullMeta struct {
	UUID             string
	App, Stream      string
	Source, RelayURL string
}

// processID 生成本地进程标识: app:stream
func (m *pullMeta) processID() string {
	return m.App + ":" + m.Stream
}

// NewRelayRegistry 创建 relay registry。
func NewRelayRegistry(processes *ffmpegx.Manager, store *Store, nodeID string, asynqClient *asynq.Client) *RelayRegistry {
	r := &RelayRegistry{
		processes: processes,
		buildCmd:  ffmpegx.BuildRelayCmd,
		meta:      make(map[string]*pullMeta),
		store:     store,
		nodeID:    nodeID,
		asynq:     asynqClient,
	}
	r.onProgressFunc = r.defaultOnProgress
	r.onProcessExitFunc = r.defaultOnProcessExit
	return r
}

// startLocalRelay 启动本地 ffmpeg 进程（纯本地，不操作 Redis）。
// app, stream 为业务标识，uuid 为中继会话标识，relayURL 为完整推流地址（含鉴权参数）。
func (r *RelayRegistry) startLocalRelay(ctx context.Context, source, app, stream, uuid, relayURL string, maxDuration ...time.Duration) error {
	if source == "" || app == "" || stream == "" || relayURL == "" {
		return fmt.Errorf("source/app/stream/relayURL 不能为空")
	}

	pid := app + ":" + stream
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	if r.getMeta(pid) != nil && r.processes.Has(pid) {
		return nil
	}

	eventCtx := context.WithoutCancel(ctx)
	md := &pullMeta{UUID: uuid, App: app, Stream: stream, Source: source, RelayURL: relayURL}
	r.saveMeta(pid, md)
	build := func(processCtx context.Context) (*exec.Cmd, error) {
		return r.buildCmd(processCtx, source, relayURL), nil
	}
	processCtx := eventCtx
	var durationCancel context.CancelFunc
	if len(maxDuration) > 0 && maxDuration[0] > 0 {
		processCtx, durationCancel = context.WithTimeout(eventCtx, maxDuration[0])
	}
	if err := r.processes.Start(processCtx, pid, build,
		ffmpegx.WithStdoutHandler(func(_ string, line string) {
			r.handleOutput(eventCtx, md, line)
		}),
		ffmpegx.WithStderrHandler(func(id, line string) {
			r.handleStderr(eventCtx, id, line)
		}),
		ffmpegx.WithExitHandler(func(_ string, result ffmpegx.ExitResult) {
			if durationCancel != nil {
				defer durationCancel()
			}
			r.handleExit(eventCtx, md, result)
		}),
	); err != nil {
		if durationCancel != nil {
			durationCancel()
		}
		r.deleteMeta(pid)
		return err
	}
	return nil
}

// StartRelay 启动中继（分布式入口）：生成 UUID → 校验已存在 → 写状态 → 抢租约 → 本地启动。
func (r *RelayRegistry) StartRelay(ctx context.Context, source, target, relayURL string, maxDurationSeconds uint64) (string, error) {
	clog := logx.WithContext(ctx)

	app, stream, err := ParseTarget(NormalizeTarget(target))
	if err != nil {
		return "", fmt.Errorf("target 格式无效: %w", err)
	}

	lock, ok, err := r.store.Lock(ctx, app, stream)
	if err != nil {
		return "", err
	}
	if !ok {
		clog.Infof("分布式锁被其他节点持有，跳过启动: app=%s stream=%s", app, stream)
		return "", nil
	}
	defer lock.Release()

	// 校验是否已存在中继（state 存在 = 应该运行）
	existing, err := r.store.GetState(ctx, app, stream)
	if err != nil {
		return "", fmt.Errorf("读取中继状态失败: %w", err)
	}
	if existing != nil {
		return "", fmt.Errorf("该 app+stream 已存在中继，请先停止: app=%s stream=%s uuid=%s", app, stream, existing.UUID)
	}

	// 生成 UUID
	uuid, err := tool.SimpleUUID()
	if err != nil {
		return "", fmt.Errorf("生成 UUID 失败: %w", err)
	}

	// 计算 deadline
	dur := time.Duration(maxDurationSeconds) * time.Second
	if dur <= 0 {
		dur = defaultRelayDuration
	}
	deadline := time.Now().Add(dur).Unix()
	deadlineStr := carbonx.FormatDateTime(time.Unix(deadline, 0))

	// 解析 host:port（日志用）
	host, port := parseHostPortFromTarget(target)

	if err := r.store.SaveState(ctx, &RelayState{
		UUID: uuid, App: app, Stream: stream, Host: host, Port: port,
		Source: source, RelayURL: relayURL, DeadlineAtUnix: deadline, DeadlineAtStr: deadlineStr, CreatedAtStr: carbonx.NowDateTime(),
	}); err != nil {
		return "", fmt.Errorf("保存中继状态失败: %w", err)
	}

	claimed, err := r.store.TryClaim(ctx, app, stream, r.nodeID)
	if err != nil {
		return "", fmt.Errorf("抢占租约失败: %w", err)
	}
	if !claimed {
		clog.Infof("租约被其他节点持有，跳过本地启动: app=%s stream=%s", app, stream)
		return uuid, nil
	}

	if err := r.startLocalRelay(ctx, source, app, stream, uuid, relayURL, remainingDuration(deadline)); err != nil {
		r.releaseAndRetry(ctx, app, stream)
		return "", fmt.Errorf("启动 ffmpeg 失败: %w", err)
	}
	// 启动成功：加入 Sorted Set 索引
	_ = r.store.AddToRegistry(ctx, app, stream)
	return uuid, nil
}

// stopLocalRelay 停止本地进程（纯本地，不操作 Redis）。
func (r *RelayRegistry) stopLocalRelay(app, stream string) bool {
	pid := app + ":" + stream
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	if !r.deleteMeta(pid) {
		return false
	}
	return r.processes.Stop(pid)
}

// StopRelay 停止中继（分布式入口）：删 state → 删 lease → 停本地进程。
func (r *RelayRegistry) StopRelay(ctx context.Context, app, stream string) bool {
	lock, ok, err := r.store.Lock(ctx, app, stream)
	if err != nil {
		logx.WithContext(ctx).Errorf("[relay] 获取分布式锁失败（停止）: app=%s stream=%s err=%v", app, stream, err)
		return false
	}
	if !ok {
		logx.WithContext(ctx).Infof("[relay] 分布式锁被其他节点持有，跳过停止: app=%s stream=%s", app, stream)
		return false
	}
	defer lock.Release()

	r.cleanupTarget(ctx, app, stream)
	return r.stopLocalRelay(app, stream)
}

// HasTarget reports whether app:stream has registered relay metadata.
func (r *RelayRegistry) HasTarget(app, stream string) bool {
	pid := app + ":" + stream
	r.metaMu.RLock()
	defer r.metaMu.RUnlock()
	_, exists := r.meta[pid]
	return exists
}

func (r *RelayRegistry) saveMeta(pid string, md *pullMeta) {
	r.metaMu.Lock()
	r.meta[pid] = md
	r.metaMu.Unlock()
}

func (r *RelayRegistry) deleteMeta(pid string) bool {
	r.metaMu.Lock()
	_, exists := r.meta[pid]
	delete(r.meta, pid)
	r.metaMu.Unlock()
	return exists
}

// StopRelayByAppStream stops local relays matching app and stream (仅本地，广播用)。
func (r *RelayRegistry) StopRelayByAppStream(app, stream string) bool {
	if app == "" || stream == "" {
		return false
	}
	pid := app + ":" + stream
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	r.metaMu.Lock()
	if r.meta[pid] == nil {
		r.metaMu.Unlock()
		return false
	}
	delete(r.meta, pid)
	r.metaMu.Unlock()
	return r.processes.Stop(pid)
}

// StopAll removes and stops only processes owned by this relay registry (仅本地，退场用)。
func (r *RelayRegistry) StopAll() {
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	r.metaMu.Lock()
	pids := make([]string, 0, len(r.meta))
	for pid := range r.meta {
		pids = append(pids, pid)
	}
	clear(r.meta)
	r.metaMu.Unlock()
	for _, pid := range pids {
		r.processes.Stop(pid)
	}
}

// cleanupTarget 清理分布式状态（state + lease + registry index）。
func (r *RelayRegistry) cleanupTarget(ctx context.Context, app, stream string) {
	_ = r.store.RemoveFromRegistry(ctx, app, stream)
	_ = r.store.DeleteState(ctx, app, stream)
	_ = r.store.DeleteLease(ctx, app, stream)
}

// Reconcile 补拉入口（Asynq 任务消费）：读状态 → 校验 UUID → 检查重试次数 → 抢租约 → 复查状态 → 启动。
func (r *RelayRegistry) Reconcile(ctx context.Context, app, stream, uuid string, retryCount int) error {
	lock, ok, err := r.store.Lock(ctx, app, stream)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	defer lock.Release()

	st, err := r.store.GetState(ctx, app, stream)
	if err != nil {
		return err
	}
	if st == nil {
		return nil
	}
	// UUID 校验：不匹配说明已被新中继覆盖，跳过
	if st.UUID != uuid {
		logx.WithContext(ctx).Infof("[relay] 补拉 UUID 不匹配，跳过: app=%s stream=%s taskUUID=%s stateUUID=%s", app, stream, uuid, st.UUID)
		return nil
	}
	if deadlineExpired(st.DeadlineAtUnix) {
		logx.WithContext(ctx).Infof("[relay] 补拉 deadline 过期，跳过: app=%s stream=%s", app, stream)
		r.cleanupTarget(ctx, app, stream)
		return nil
	}

	claimed, err := r.store.TryClaim(ctx, app, stream, r.nodeID)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}

	st2, err := r.store.GetState(ctx, app, stream)
	if err != nil {
		_, _ = r.store.Release(ctx, app, stream, r.nodeID)
		return err
	}
	if st2 == nil || deadlineExpired(st2.DeadlineAtUnix) {
		_, _ = r.store.Release(ctx, app, stream, r.nodeID)
		r.cleanupTarget(ctx, app, stream)
		return nil
	}
	// 二次 UUID 校验
	if st2.UUID != uuid {
		logx.WithContext(ctx).Infof("[relay] 补拉二次 UUID 不匹配，跳过: app=%s stream=%s taskUUID=%s stateUUID=%s", app, stream, uuid, st2.UUID)
		_, _ = r.store.Release(ctx, app, stream, r.nodeID)
		return nil
	}

	if err := r.startLocalRelay(ctx, st2.Source, app, stream, uuid, st2.RelayURL, remainingDuration(st2.DeadlineAtUnix)); err != nil {
		_, _ = r.store.Release(ctx, app, stream, r.nodeID)
		return fmt.Errorf("启动 ffmpeg 失败: %w", err)
	}
	// 启动成功：清 pending 标记 + 加入 registry 索引
	st2.PendingReconcile = false
	_ = r.store.SaveState(ctx, st2)
	_ = r.store.AddToRegistry(ctx, app, stream)
	return nil
}

func (r *RelayRegistry) enqueueTask(ctx context.Context, taskType string, payload any, delay time.Duration, retention time.Duration) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	task := asynq.NewTask(taskType, data)
	_, err = r.asynq.EnqueueContext(ctx, task,
		asynq.Queue(RelayQueue),
		asynq.Retention(retention),
		asynq.ProcessIn(delay),
	)
	return err
}

// EnqueueReconcile 入队补拉任务（随机 jitter 延迟执行）
func (r *RelayRegistry) EnqueueReconcile(ctx context.Context, payload ReconcilePayload) error {
	delay := reconcileDelay(payload.RetryCount)
	if err := r.enqueueTask(ctx, RelayReconcileTask, payload, delay, 1*time.Hour); err != nil {
		logx.WithContext(ctx).Errorf("[asynq-task] 入队补拉失败: app=%s stream=%s err=%v", payload.App, payload.Stream, err)
		return err
	}
	logx.WithContext(ctx).Infof("[asynq-task] 入队补拉: app=%s stream=%s retryCount=%d delay=%s", payload.App, payload.Stream, payload.RetryCount, delay)
	return nil
}

// EnqueueStop 入队补停任务（延迟 5s 执行，给广播恢复窗口）
func (r *RelayRegistry) EnqueueStop(ctx context.Context, payload StopPayload) error {
	if err := r.enqueueTask(ctx, RelayStopTask, payload, 5*time.Second, 7*24*time.Hour); err != nil {
		logx.WithContext(ctx).Errorf("[asynq-task] 入队补停失败: app=%s stream=%s err=%v", payload.App, payload.Stream, err)
		return err
	}
	logx.WithContext(ctx).Infof("[asynq-task] 入队补停: app=%s stream=%s delay=5s", payload.App, payload.Stream)
	return nil
}

// defaultOnProgress 默认 progress 回调：续租 + 续 registry 索引；续租失败 → 停止本地进程。
func (r *RelayRegistry) defaultOnProgress(ctx context.Context, md *pullMeta) {
	if r.store == nil {
		return
	}
	logx.WithContext(ctx).Debugf("[relay] 进度续租: app=%s stream=%s", md.App, md.Stream)
	ok, err := r.store.Renew(ctx, md.App, md.Stream, r.nodeID)
	if err != nil {
		logx.WithContext(ctx).Errorf("[relay] 续租失败: app=%s stream=%s err=%v", md.App, md.Stream, err)
		return
	}
	if !ok {
		logx.WithContext(ctx).Infof("[relay] 租约丢失，停止本地进程: app=%s stream=%s", md.App, md.Stream)
		r.stopLocalRelay(md.App, md.Stream)
		return
	}
	// 续 registry 索引（与续租同步，扫描器以此判断 relay 是否存活）
	_ = r.store.RenewRegistry(ctx, md.App, md.Stream)
}

// defaultOnProcessExit 默认进程退出回调：deadline 超时 → 清理；其他 → releaseAndRetry。
func (r *RelayRegistry) defaultOnProcessExit(ctx context.Context, md *pullMeta, result ffmpegx.ExitResult) {
	if r.store == nil {
		return
	}
	logx.WithContext(ctx).Infof("[relay] 进程退出: app=%s stream=%s waitErr=%v contextErr=%v", md.App, md.Stream, result.WaitErr, result.ContextErr)
	if result.ContextErr == context.DeadlineExceeded {
		logx.WithContext(ctx).Infof("[relay] deadline 超时，清理状态: app=%s stream=%s", md.App, md.Stream)
		r.cleanupTarget(ctx, md.App, md.Stream)
		return
	}
	r.releaseAndRetry(ctx, md.App, md.Stream)
}

// releaseAndRetry 启动失败路径：释放租约 + retry+1 + pending=true → 入队补拉
func (r *RelayRegistry) releaseAndRetry(ctx context.Context, app, stream string) {
	_, _ = r.store.Release(ctx, app, stream, r.nodeID)
	st, _ := r.store.GetState(ctx, app, stream)
	retryCount := 0
	uuid := ""
	if st != nil {
		uuid = st.UUID
		retryCount = st.RetryCount + 1
		st.RetryCount = retryCount
		st.PendingReconcile = true
		_ = r.store.SaveState(ctx, st)
	}
	if err := r.EnqueueReconcile(ctx, ReconcilePayload{App: app, Stream: stream, UUID: uuid, RetryCount: retryCount}); err != nil {
		logx.WithContext(ctx).Errorf("[relay] 入队补拉失败: app=%s stream=%s err=%v", app, stream, err)
	}
}

// handleOutput 解析 ffmpeg 进度输出，progress=continue 时触发续租回调。
func (r *RelayRegistry) handleOutput(ctx context.Context, md *pullMeta, line string) {
	key, value, ok := strings.Cut(line, "=")
	if !ok || key == "" {
		return
	}
	pid := md.processID()
	if r.getMeta(pid) != md {
		return
	}
	logx.WithContext(ctx).Debugf("[relay] stdout: app=%s stream=%s key=%s value=%s", md.App, md.Stream, key, value)
	if key == "progress" && value == "continue" {
		r.onProgressFunc(ctx, md)
	}
}

func (r *RelayRegistry) handleExit(ctx context.Context, md *pullMeta, result ffmpegx.ExitResult) {
	pid := md.processID()
	r.metaMu.Lock()
	if r.meta[pid] != md {
		r.metaMu.Unlock()
		logx.WithContext(ctx).Debugf("[relay] handleExit 跳过（过期 metadata）: app=%s stream=%s", md.App, md.Stream)
		return
	}
	delete(r.meta, pid)
	r.metaMu.Unlock()
	logx.WithContext(ctx).Debugf("[relay] handleExit: app=%s stream=%s waitErr=%v contextErr=%v", md.App, md.Stream, result.WaitErr, result.ContextErr)
	r.onProcessExitFunc(ctx, md, result)
}

func (r *RelayRegistry) getMeta(pid string) *pullMeta {
	r.metaMu.RLock()
	md := r.meta[pid]
	r.metaMu.RUnlock()
	return md
}

func (r *RelayRegistry) handleStderr(_ context.Context, id, line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}
	// 只保留错误/警告行，过滤 ffmpeg 所有正常输出（banner、metadata、stream info、progress）
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "error") ||
		strings.Contains(lower, "warning") ||
		strings.Contains(lower, "cannot") ||
		strings.Contains(lower, "failed") ||
		strings.Contains(lower, "fatal") ||
		strings.Contains(lower, "denied") ||
		strings.Contains(lower, "timeout") {
		logx.Errorf("[ffmpegx] stderr: id=%s line=%s", id, line)
	}
}

// parseHostPortFromTarget 从 target URL 解析 host 和 port（日志用）。
func parseHostPortFromTarget(target string) (string, int) {
	normalized := NormalizeTarget(target)
	i := strings.Index(normalized, "/")
	if i < 0 {
		return normalized, 0
	}
	hostPort := normalized[:i]
	parts := strings.SplitN(hostPort, ":", 2)
	if len(parts) == 2 {
		port := 0
		fmt.Sscanf(parts[1], "%d", &port)
		return parts[0], port
	}
	return hostPort, 0
}

// reconcileDelay 计算补拉随机延迟：10s~60s jitter，避免同时重试风暴。
func reconcileDelay(_ int) time.Duration {
	return tool.JitterDelay(10*time.Second, 60*time.Second)
}

// deadlineExpired 判断截止时间是否已过。
func deadlineExpired(deadline int64) bool { return deadline > 0 && time.Now().Unix() >= deadline }

// remainingDuration 计算截止时间的剩余时长，deadline=0 返回 0（无限制）。
func remainingDuration(deadline int64) time.Duration {
	if deadline == 0 {
		return 0
	}
	return time.Until(time.Unix(deadline, 0))
}
