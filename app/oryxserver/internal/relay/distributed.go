package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"zero-service/common/carbonx"
	"zero-service/common/ffmpegx"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
)

// DistributedRelay 分布式中继协调器：Redis 期望状态 + nodeID 租约 + 本地进程 + Asynq 补拉。
// state key = target（明文目标地址）；lease key = target（目标地址）。
type DistributedRelay struct {
	store    *Store
	registry *RelayRegistry
	nodeID   string
	asynq    *asynq.Client
}

// NewDistributedRelay 创建协调器
func NewDistributedRelay(store *Store, registry *RelayRegistry, nodeID string, asynqClient *asynq.Client) *DistributedRelay {
	return &DistributedRelay{
		store:    store,
		registry: registry,
		nodeID:   nodeID,
		asynq:    asynqClient,
	}
}

// StartRelay 启动中继：写状态 → 抢租约 → 本地启动。
// target = 基础地址（状态/租约标识，不含鉴权参数）；
// relayURL = 含鉴权参数的完整地址（ffmpeg 推流目标）。
func (d *DistributedRelay) StartRelay(ctx context.Context, source, target, relayURL string, maxDurationSeconds uint64) (string, error) {
	clog := logx.WithContext(ctx)

	// 标准化 target：去掉 scheme，防止 http/rtmp 指向同一地址被当作两个不同 relay
	target = NormalizeTarget(target)

	// 分布式锁：保护 read-state → claim-lease → start-local 关键段
	lock, ok, err := d.store.Lock(ctx, target)
	if err != nil {
		return "", err
	}
	if !ok {
		clog.Infof("lock held by another node, skip start: target=%s", target)
		return target, nil
	}
	defer lock.Release()

	// 幂等：同目标有租约 → 已有节点在跑，不覆盖状态
	has, err := d.store.HasLease(ctx, target)
	if err != nil {
		return "", fmt.Errorf("check lease: %w", err)
	}
	if has {
		return target, nil
	}

	// 写入期望状态
	deadline := int64(0)
	previous, err := d.store.GetState(ctx, target)
	if err != nil {
		return "", fmt.Errorf("get relay state: %w", err)
	}
	if previous != nil {
		deadline = previous.DeadlineAtUnix
	} else if maxDurationSeconds > 0 {
		deadline = time.Now().Add(time.Duration(maxDurationSeconds) * time.Second).Unix()
	}
	deadlineStr := ""
	if deadline > 0 {
		deadlineStr = carbonx.FormatDateTime(time.Unix(deadline, 0))
	}
	if err := d.store.SaveState(ctx, &RelayState{
		Source: source, Target: target, RelayURL: relayURL, DeadlineAtUnix: deadline, DeadlineAtStr: deadlineStr, CreatedAtStr: carbonx.NowDateTime(),
	}); err != nil {
		return "", fmt.Errorf("save relay state: %w", err)
	}

	// 抢占租约
	claimed, err := d.store.TryClaim(ctx, target, d.nodeID)
	if err != nil {
		return "", fmt.Errorf("claim lease: %w", err)
	}
	if !claimed {
		clog.Infof("lease held by another node, skip local start: target=%s", target)
		return target, nil
	}

	// 启动本地进程
	if err := d.registry.StartRelay(ctx, source, target, relayURL, remainingDuration(deadline)); err != nil {
		d.releaseAndEnqueue(ctx, target)
		return "", fmt.Errorf("start ffmpeg: %w", err)
	}
	return target, nil
}

// StopRelay 停止中继：删 state → 删 lease → 停本地进程。
func (d *DistributedRelay) StopRelay(ctx context.Context, target string) bool {
	lock, ok, err := d.store.Lock(ctx, target)
	if err != nil {
		logx.WithContext(ctx).Errorf("[relay] acquire lock failed for stop: target=%s err=%v", target, err)
		return false
	}
	if !ok {
		logx.WithContext(ctx).Infof("[relay] lock held by another node, skip stop: target=%s", target)
		return false
	}
	defer lock.Release()

	_ = d.store.DeleteState(ctx, target)
	_ = d.store.DeleteLease(ctx, target)
	return d.registry.StopRelay(target)
}

// Reconcile 补拉入口（Asynq 任务消费）：读状态 → 检查重试次数 → 抢租约 → 复查状态 → 启动。
func (d *DistributedRelay) Reconcile(ctx context.Context, target string, retryCount int) error {
	// 分布式锁：保护 read → claim → recheck → start 关键段，防止与 Stop 竞争
	lock, ok, err := d.store.Lock(ctx, target)
	if err != nil {
		return err
	}
	if !ok {
		return nil // 另一个节点正在操作此 target
	}
	defer lock.Release()

	st, err := d.store.GetState(ctx, target)
	if err != nil {
		return err
	}
	if st == nil {
		return nil // 已停止/被替换 → 不复活
	}
	if deadlineExpired(st.DeadlineAtUnix) {
		_ = d.store.DeleteState(ctx, target)
		_ = d.store.DeleteLease(ctx, target)
		return nil
	}
	// 检查重试次数（防御性：即使 payload 携带了 retryCount，也以 state 为准）
	if st.RetryCount > maxReconcileRetries {
		logx.WithContext(ctx).Errorf("[relay] max reconcile retries exceeded, stop: target=%s retryCount=%d", target, st.RetryCount)
		_ = d.store.DeleteState(ctx, target)
		_ = d.store.DeleteLease(ctx, target)
		return nil
	}

	claimed, err := d.store.TryClaim(ctx, target, d.nodeID)
	if err != nil {
		return err
	}
	if !claimed {
		return nil // 别的节点在跑 → ack
	}

	// 复查：抢租约期间被 Stop/替换
	st2, err := d.store.GetState(ctx, target)
	if err != nil {
		_, _ = d.store.Release(ctx, target, d.nodeID)
		return err
	}
	if st2 == nil {
		_, _ = d.store.Release(ctx, target, d.nodeID)
		return nil
	}
	if deadlineExpired(st2.DeadlineAtUnix) {
		_, _ = d.store.Release(ctx, target, d.nodeID)
		_ = d.store.DeleteState(ctx, target)
		return nil
	}

	if err := d.registry.StartRelay(ctx, st2.Source, st2.Target, st2.RelayURL, remainingDuration(st2.DeadlineAtUnix)); err != nil {
		_, _ = d.store.Release(ctx, target, d.nodeID)
		return fmt.Errorf("start ffmpeg: %w", err)
	}
	return nil
}

// reconcileDelay 根据重试次数计算递增延迟：5s, 10s, 20s, 40s, ... 上限 30 分钟。
func reconcileDelay(retryCount int) time.Duration {
	delay := time.Duration(5<<retryCount) * time.Second
	if delay > 30*time.Minute {
		delay = 30 * time.Minute
	}
	return delay
}

// EnqueueReconcile 入队补拉任务（递增延迟执行）
func (d *DistributedRelay) EnqueueReconcile(ctx context.Context, target string, retryCount int) error {
	delay := reconcileDelay(retryCount)
	payload, err := json.Marshal(ReconcilePayload{Target: target, RetryCount: retryCount})
	if err != nil {
		return err
	}
	task := asynq.NewTask(RelayReconcileTask, payload)
	_, err = d.asynq.EnqueueContext(ctx, task,
		asynq.Queue(RelayQueue),
		asynq.Retention(7*24*time.Hour),
		asynq.ProcessIn(delay),
	)
	if err != nil {
		logx.WithContext(ctx).Errorf("[asynq] enqueue reconcile failed: target=%s err=%v", target, err)
		return err
	}
	logx.WithContext(ctx).Infof("[asynq] enqueue reconcile: target=%s retryCount=%d delay=%s", target, retryCount, delay)
	return nil
}

// OnProgress ffmpeg progress 帧回调：续租；续租失败（租约被抢/被删）→ 停止本地进程。
func (d *DistributedRelay) OnProgress(ctx context.Context, target string) {
	logx.WithContext(ctx).Debugf("[relay] on progress: target=%s", target)
	ok, err := d.store.Renew(ctx, target, d.nodeID)
	if err != nil {
		logx.WithContext(ctx).Errorf("[relay] renew lease failed: target=%s err=%v", target, err)
		return
	}
	if !ok {
		logx.WithContext(ctx).Infof("[relay] lease lost, stop local process: target=%s", target)
		d.registry.StopRelay(target)
	}
}

// OnProcessExit ffmpeg 异常退出回调：读 state retry_count → +1 → 超限则停止 → 否则入队补拉。
func (d *DistributedRelay) OnProcessExit(ctx context.Context, target string, result ffmpegx.ExitResult) {
	logx.WithContext(ctx).Infof("[relay] on process exit: target=%s waitErr=%v contextErr=%v", target, result.WaitErr, result.ContextErr)
	if result.ContextErr == context.DeadlineExceeded {
		logx.WithContext(ctx).Infof("[relay] deadline exceeded, delete state: target=%s", target)
		_ = d.store.DeleteState(ctx, target)
		_ = d.store.DeleteLease(ctx, target)
		return
	}

	// 读当前 state 获取 retry_count
	st, _ := d.store.GetState(ctx, target)
	retryCount := 0
	if st != nil {
		retryCount = st.RetryCount + 1
	}

	// 释放租约
	_, _ = d.store.Release(ctx, target, d.nodeID)

	// 超限 → 停止补拉，删状态
	if retryCount > maxReconcileRetries {
		logx.WithContext(ctx).Errorf("[relay] max reconcile retries exceeded, stop: target=%s retryCount=%d", target, retryCount)
		_ = d.store.DeleteState(ctx, target)
		_ = d.store.DeleteLease(ctx, target)
		return
	}

	// 更新 state 的 retry_count
	if st != nil {
		st.RetryCount = retryCount
		if err := d.store.SaveState(ctx, st); err != nil {
			logx.WithContext(ctx).Errorf("[relay] save retry count failed: target=%s err=%v", target, err)
		}
	}

	// 入队补拉（带延迟）
	if err := d.EnqueueReconcile(ctx, target, retryCount); err != nil {
		logx.WithContext(ctx).Errorf("[relay] enqueue reconcile failed: target=%s err=%v", target, err)
	}
}

func deadlineExpired(deadline int64) bool { return deadline > 0 && time.Now().Unix() >= deadline }

func remainingDuration(deadline int64) time.Duration {
	if deadline == 0 {
		return 0
	}
	return time.Until(time.Unix(deadline, 0))
}

// releaseAndEnqueue 启动失败路径：释放租约 + 读 retry_count +1 → 入队补拉
func (d *DistributedRelay) releaseAndEnqueue(ctx context.Context, target string) {
	_, _ = d.store.Release(ctx, target, d.nodeID)
	st, _ := d.store.GetState(ctx, target)
	retryCount := 0
	if st != nil {
		retryCount = st.RetryCount + 1
		st.RetryCount = retryCount
		_ = d.store.SaveState(ctx, st)
	}
	if retryCount <= maxReconcileRetries {
		if err := d.EnqueueReconcile(ctx, target, retryCount); err != nil {
			logx.WithContext(ctx).Errorf("[relay] enqueue reconcile failed: target=%s err=%v", target, err)
		}
	} else {
		logx.WithContext(ctx).Errorf("[relay] max reconcile retries exceeded in releaseAndEnqueue: target=%s retryCount=%d", target, retryCount)
	}
}

// EnqueueStop 入队补停任务（延迟 5s 执行，给广播恢复窗口）
func (d *DistributedRelay) EnqueueStop(ctx context.Context, target, app, stream string) error {
	payload, err := json.Marshal(StopPayload{Target: target, App: app, Stream: stream})
	if err != nil {
		return err
	}
	task := asynq.NewTask(RelayStopTask, payload)
	_, err = d.asynq.EnqueueContext(ctx, task,
		asynq.Queue(RelayQueue),
		asynq.Retention(7*24*time.Hour),
		asynq.ProcessIn(5*time.Second),
	)
	if err != nil {
		logx.WithContext(ctx).Errorf("[asynq] enqueue stop failed: target=%s app=%s stream=%s err=%v", target, app, stream, err)
		return err
	}
	logx.WithContext(ctx).Infof("[asynq] enqueue stop: target=%s app=%s stream=%s delay=5s", target, app, stream)
	return nil
}
