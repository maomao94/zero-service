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
	onProgressFunc    func(ctx context.Context, uid string)
	onProcessExitFunc func(ctx context.Context, uid string, result ffmpegx.ExitResult)
}

type pullMeta struct {
	UID, Source, RelayURL, App, Stream string
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
// uid 为唯一标识（host_port/app/stream），relayURL 为完整推流地址（含鉴权参数）。
func (r *RelayRegistry) startLocalRelay(ctx context.Context, source, uid, relayURL string, maxDuration ...time.Duration) error {
	if source == "" || uid == "" || relayURL == "" {
		return fmt.Errorf("source/uid/relayURL 不能为空")
	}

	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	if r.getMeta(uid) != nil && r.processes.Has(uid) {
		return nil
	}

	eventCtx := context.WithoutCancel(ctx)
	app, stream, err := ParseTarget(uid)
	if err != nil {
		return fmt.Errorf("UID 格式无效: %w", err)
	}
	md := &pullMeta{UID: uid, Source: source, RelayURL: relayURL, App: app, Stream: stream}
	r.saveMeta(uid, md)
	build := func(processCtx context.Context) (*exec.Cmd, error) {
		return r.buildCmd(processCtx, source, relayURL), nil
	}
	processCtx := eventCtx
	var durationCancel context.CancelFunc
	if len(maxDuration) > 0 && maxDuration[0] > 0 {
		processCtx, durationCancel = context.WithTimeout(eventCtx, maxDuration[0])
	}
	if err := r.processes.Start(processCtx, uid, build,
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
		r.deleteMeta(uid)
		return err
	}
	return nil
}

// StartRelay 启动中继（分布式入口）：写状态 → 抢租约 → 本地启动。
func (r *RelayRegistry) StartRelay(ctx context.Context, source, target, relayURL string, maxDurationSeconds uint64) (string, error) {
	clog := logx.WithContext(ctx)

	uid, err := CanonicalUID(target)
	if err != nil {
		return "", fmt.Errorf("target 格式无效: %w", err)
	}

	lock, ok, err := r.store.Lock(ctx, uid)
	if err != nil {
		return "", err
	}
	if !ok {
		clog.Infof("分布式锁被其他节点持有，跳过启动: uid=%s", uid)
		return uid, nil
	}
	defer lock.Release()

	has, err := r.store.HasLease(ctx, uid)
	if err != nil {
		return "", fmt.Errorf("检查租约失败: %w", err)
	}
	if has {
		return "", fmt.Errorf("该 target 已有活跃租约: %s", uid)
	}

	deadline := int64(0)
	previous, err := r.store.GetState(ctx, uid)
	if err != nil {
		return "", fmt.Errorf("读取中继状态失败: %w", err)
	}
	if previous != nil {
		if previous.Source != source {
			return "", fmt.Errorf("该 target 已存在但 source 不同: 已有=%s 请求=%s", previous.Source, source)
		}
		deadline = previous.DeadlineAtUnix
	} else {
		// maxDurationSeconds=0 时默认 1 天（短中继场景，不设无限）
		dur := time.Duration(maxDurationSeconds) * time.Second
		if dur <= 0 {
			dur = defaultRelayDuration
		}
		deadline = time.Now().Add(dur).Unix()
	}
	deadlineStr := ""
	if deadline > 0 {
		deadlineStr = carbonx.FormatDateTime(time.Unix(deadline, 0))
	}

	app, stream, _ := ParseTarget(uid)
	host, port := parseHostPort(uid)

	if err := r.store.SaveState(ctx, &RelayState{
		UID: uid, Host: host, Port: port, App: app, Stream: stream,
		Source: source, RelayURL: relayURL, DeadlineAtUnix: deadline, DeadlineAtStr: deadlineStr, CreatedAtStr: carbonx.NowDateTime(),
	}); err != nil {
		return "", fmt.Errorf("保存中继状态失败: %w", err)
	}

	claimed, err := r.store.TryClaim(ctx, uid, r.nodeID)
	if err != nil {
		return "", fmt.Errorf("抢占租约失败: %w", err)
	}
	if !claimed {
		clog.Infof("租约被其他节点持有，跳过本地启动: uid=%s", uid)
		return uid, nil
	}

	if err := r.startLocalRelay(ctx, source, uid, relayURL, remainingDuration(deadline)); err != nil {
		r.releaseAndRetry(ctx, uid)
		return "", fmt.Errorf("启动 ffmpeg 失败: %w", err)
	}
	// 启动成功：加入 Sorted Set 索引
	_ = r.store.AddToRegistry(ctx, uid)
	return uid, nil
}

// stopLocalRelay 停止本地进程（纯本地，不操作 Redis）。
func (r *RelayRegistry) stopLocalRelay(uid string) bool {
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	if !r.deleteMeta(uid) {
		return false
	}
	return r.processes.Stop(uid)
}

// StopRelay 停止中继（分布式入口）：删 state → 删 lease → 停本地进程。
func (r *RelayRegistry) StopRelay(ctx context.Context, uid string) bool {
	lock, ok, err := r.store.Lock(ctx, uid)
	if err != nil {
		logx.WithContext(ctx).Errorf("[relay] 获取分布式锁失败（停止）: uid=%s err=%v", uid, err)
		return false
	}
	if !ok {
		logx.WithContext(ctx).Infof("[relay] 分布式锁被其他节点持有，跳过停止: uid=%s", uid)
		return false
	}
	defer lock.Release()

	r.cleanupTarget(ctx, uid)
	return r.stopLocalRelay(uid)
}

// HasTarget reports whether uid has registered relay metadata.
func (r *RelayRegistry) HasTarget(uid string) bool {
	r.metaMu.RLock()
	defer r.metaMu.RUnlock()
	_, exists := r.meta[uid]
	return exists
}

func (r *RelayRegistry) saveMeta(uid string, md *pullMeta) {
	r.metaMu.Lock()
	r.meta[uid] = md
	r.metaMu.Unlock()
}

func (r *RelayRegistry) deleteMeta(uid string) bool {
	r.metaMu.Lock()
	_, exists := r.meta[uid]
	delete(r.meta, uid)
	r.metaMu.Unlock()
	return exists
}

// StopRelayByAppStream stops local relays matching app and stream (仅本地，广播用)。
func (r *RelayRegistry) StopRelayByAppStream(app, stream string) bool {
	if app == "" || stream == "" {
		return false
	}
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	r.metaMu.Lock()
	var uids []string
	for uid, md := range r.meta {
		if md.App == app && md.Stream == stream {
			uids = append(uids, uid)
		}
	}
	for _, uid := range uids {
		delete(r.meta, uid)
	}
	r.metaMu.Unlock()
	stopped := false
	for _, uid := range uids {
		if r.processes.Stop(uid) {
			stopped = true
		}
	}
	return stopped
}

// StopAll removes and stops only processes owned by this relay registry (仅本地，退场用)。
func (r *RelayRegistry) StopAll() {
	r.lifecycleMu.Lock()
	defer r.lifecycleMu.Unlock()
	r.metaMu.Lock()
	uids := make([]string, 0, len(r.meta))
	for uid := range r.meta {
		uids = append(uids, uid)
	}
	clear(r.meta)
	r.metaMu.Unlock()
	for _, uid := range uids {
		r.processes.Stop(uid)
	}
}

// cleanupTarget 清理分布式状态（state + lease + registry index）。
func (r *RelayRegistry) cleanupTarget(ctx context.Context, uid string) {
	_ = r.store.RemoveFromRegistry(ctx, uid)
	_ = r.store.DeleteState(ctx, uid)
	_ = r.store.DeleteLease(ctx, uid)
}

// Reconcile 补拉入口（Asynq 任务消费）：读状态 → 检查重试次数 → 抢租约 → 复查状态 → 启动。
func (r *RelayRegistry) Reconcile(ctx context.Context, uid, source string, retryCount int) error {
	lock, ok, err := r.store.Lock(ctx, uid)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	defer lock.Release()

	st, err := r.store.GetState(ctx, uid)
	if err != nil {
		return err
	}
	if st == nil {
		return nil
	}
	if deadlineExpired(st.DeadlineAtUnix) {
		r.cleanupTarget(ctx, uid)
		return nil
	}

	claimed, err := r.store.TryClaim(ctx, uid, r.nodeID)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}

	st2, err := r.store.GetState(ctx, uid)
	if err != nil {
		_, _ = r.store.Release(ctx, uid, r.nodeID)
		return err
	}
	if st2 == nil || deadlineExpired(st2.DeadlineAtUnix) {
		_, _ = r.store.Release(ctx, uid, r.nodeID)
		r.cleanupTarget(ctx, uid)
		return nil
	}

	// source 校验：task 带的 source 和 state 不一致说明已被新请求覆盖，跳过（不改 state）
	if source != st2.Source {
		logx.WithContext(ctx).Infof("[relay] 补拉 source 不匹配，跳过: uid=%s taskSource=%s stateSource=%s", uid, source, st2.Source)
		_, _ = r.store.Release(ctx, uid, r.nodeID)
		return nil
	}
	if err := r.startLocalRelay(ctx, st2.Source, uid, st2.RelayURL, remainingDuration(st2.DeadlineAtUnix)); err != nil {
		_, _ = r.store.Release(ctx, uid, r.nodeID)
		return fmt.Errorf("启动 ffmpeg 失败: %w", err)
	}
	// 启动成功：清 pending 标记 + 加入 registry 索引
	st2.PendingReconcile = false
	_ = r.store.SaveState(ctx, st2)
	_ = r.store.AddToRegistry(ctx, uid)
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
func (r *RelayRegistry) EnqueueReconcile(ctx context.Context, uid, source string, retryCount int) error {
	delay := reconcileDelay(retryCount)
	if err := r.enqueueTask(ctx, RelayReconcileTask, ReconcilePayload{UID: uid, Source: source, RetryCount: retryCount}, delay, 1*time.Hour); err != nil {
		logx.WithContext(ctx).Errorf("[asynq-task] 入队补拉失败: uid=%s err=%v", uid, err)
		return err
	}
	logx.WithContext(ctx).Infof("[asynq-task] 入队补拉: uid=%s retryCount=%d delay=%s", uid, retryCount, delay)
	return nil
}

// EnqueueStop 入队补停任务（延迟 5s 执行，给广播恢复窗口）
func (r *RelayRegistry) EnqueueStop(ctx context.Context, uid, app, stream string) error {
	if err := r.enqueueTask(ctx, RelayStopTask, StopPayload{UID: uid, App: app, Stream: stream}, 5*time.Second, 7*24*time.Hour); err != nil {
		logx.WithContext(ctx).Errorf("[asynq-task] 入队补停失败: uid=%s app=%s stream=%s err=%v", uid, app, stream, err)
		return err
	}
	logx.WithContext(ctx).Infof("[asynq-task] 入队补停: uid=%s app=%s stream=%s delay=5s", uid, app, stream)
	return nil
}

// defaultOnProgress 默认 progress 回调：续租 + 续 registry 索引；续租失败 → 停止本地进程。
func (r *RelayRegistry) defaultOnProgress(ctx context.Context, uid string) {
	if r.store == nil {
		return
	}
	logx.WithContext(ctx).Debugf("[relay] 进度续租: uid=%s", uid)
	ok, err := r.store.Renew(ctx, uid, r.nodeID)
	if err != nil {
		logx.WithContext(ctx).Errorf("[relay] 续租失败: uid=%s err=%v", uid, err)
		return
	}
	if !ok {
		logx.WithContext(ctx).Infof("[relay] 租约丢失，停止本地进程: uid=%s", uid)
		r.stopLocalRelay(uid)
		return
	}
	// 续 registry 索引（与续租同步，扫描器以此判断 relay 是否存活）
	_ = r.store.RenewRegistry(ctx, uid)
}

// defaultOnProcessExit 默认进程退出回调：deadline 超时 → 清理；其他 → releaseAndRetry。
func (r *RelayRegistry) defaultOnProcessExit(ctx context.Context, uid string, result ffmpegx.ExitResult) {
	if r.store == nil {
		return
	}
	logx.WithContext(ctx).Infof("[relay] 进程退出: uid=%s waitErr=%v contextErr=%v", uid, result.WaitErr, result.ContextErr)
	if result.ContextErr == context.DeadlineExceeded {
		logx.WithContext(ctx).Infof("[relay] deadline 超时，清理状态: uid=%s", uid)
		r.cleanupTarget(ctx, uid)
		return
	}
	r.releaseAndRetry(ctx, uid)
}

// releaseAndRetry 启动失败路径：释放租约 + retry+1 + pending=true → 入队补拉
func (r *RelayRegistry) releaseAndRetry(ctx context.Context, uid string) {
	_, _ = r.store.Release(ctx, uid, r.nodeID)
	st, _ := r.store.GetState(ctx, uid)
	retryCount := 0
	source := ""
	if st != nil {
		source = st.Source
		retryCount = st.RetryCount + 1
		st.RetryCount = retryCount
		st.PendingReconcile = true
		_ = r.store.SaveState(ctx, st)
	}
	if err := r.EnqueueReconcile(ctx, uid, source, retryCount); err != nil {
		logx.WithContext(ctx).Errorf("[relay] 入队补拉失败: uid=%s err=%v", uid, err)
	}
}

// handleOutput 解析 ffmpeg 进度输出，progress=continue 时触发续租回调。
func (r *RelayRegistry) handleOutput(ctx context.Context, md *pullMeta, line string) {
	key, value, ok := strings.Cut(line, "=")
	if !ok || key == "" {
		return
	}
	if r.getMeta(md.UID) != md {
		return
	}
	logx.WithContext(ctx).Debugf("[relay] stdout: uid=%s key=%s value=%s", md.UID, key, value)
	if key == "progress" && value == "continue" {
		r.onProgressFunc(ctx, md.UID)
	}
}

func (r *RelayRegistry) handleExit(ctx context.Context, md *pullMeta, result ffmpegx.ExitResult) {
	r.metaMu.Lock()
	if r.meta[md.UID] != md {
		r.metaMu.Unlock()
		logx.WithContext(ctx).Debugf("[relay] handleExit 跳过（过期 metadata）: uid=%s", md.UID)
		return
	}
	delete(r.meta, md.UID)
	r.metaMu.Unlock()
	logx.WithContext(ctx).Debugf("[relay] handleExit: uid=%s waitErr=%v contextErr=%v", md.UID, result.WaitErr, result.ContextErr)
	r.onProcessExitFunc(ctx, md.UID, result)
}

func (r *RelayRegistry) getMeta(uid string) *pullMeta {
	r.metaMu.RLock()
	md := r.meta[uid]
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

// parseHostPort 从 UID（host_port/app/stream）解析 host 和 port。
func parseHostPort(uid string) (string, int) {
	i := strings.Index(uid, "/")
	if i < 0 {
		return strings.ReplaceAll(uid, "_", ":"), 0
	}
	hostPort := uid[:i]
	parts := strings.SplitN(hostPort, "_", 2)
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
