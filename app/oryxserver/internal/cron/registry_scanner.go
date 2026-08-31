package cron

import (
	"context"
	"time"

	"zero-service/app/oryxserver/internal/relay"
	"zero-service/app/oryxserver/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	scanInterval = 30 * time.Second
)

// RegistryScanner 定时扫描 Sorted Set 索引，发现无 lease 的孤儿 relay → EnqueueReconcile。
// 使用 Sorted Set 的 ZRANGEBYSCORE 只取过期条目（O(log N + M)），不扫全量。
type RegistryScanner struct {
	svcCtx     *svc.ServiceContext
	cancelChan chan struct{}
}

func NewRegistryScanner(svcCtx *svc.ServiceContext) *RegistryScanner {
	return &RegistryScanner{svcCtx: svcCtx}
}

func (s *RegistryScanner) Start() {
	if s.cancelChan != nil {
		return
	}
	s.cancelChan = make(chan struct{})
	logx.Info("[registry-scanner] started")
	go s.loop()
}

func (s *RegistryScanner) Stop() {
	if s.cancelChan != nil {
		close(s.cancelChan)
		s.cancelChan = nil
		logx.Info("[registry-scanner] stopped")
	}
}

func (s *RegistryScanner) loop() {
	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.cancelChan:
			return
		case <-ticker.C:
			s.scan()
		}
	}
}

func (s *RegistryScanner) scan() {
	ctx := context.Background()
	store := s.svcCtx.StateStore
	registry := s.svcCtx.RelayRegistry

	stale, err := store.ScanStale(ctx)
	if err != nil {
		logx.Errorf("[registry-scanner] scan stale failed: %v", err)
		return
	}
	if len(stale) == 0 {
		return
	}
	logx.Infof("[registry-scanner] found %d stale entries", len(stale))

	for _, entry := range stale {
		s.processStale(ctx, store, registry, entry)
	}
}

func (s *RegistryScanner) processStale(ctx context.Context, store *relay.Store, registry *relay.RelayRegistry, entry *relay.StaleEntry) {
	app := entry.State.App
	stream := entry.State.Stream
	uuid := entry.State.UUID

	// 拿锁再判断：防止和 Reconcile/StartRelay/StopRelay 并发
	lock, ok, err := store.Lock(ctx, app, stream)
	if err != nil {
		logx.Errorf("[registry-scanner] lock failed: app=%s stream=%s err=%v", app, stream, err)
		return
	}
	if !ok {
		return // 别人在操作，跳过
	}
	defer lock.Release()

	// 拿锁后重新读 state（可能已被删除或修改）
	latest, err := store.GetState(ctx, app, stream)
	if err != nil {
		logx.Errorf("[registry-scanner] get state failed: app=%s stream=%s err=%v", app, stream, err)
		return
	}
	if latest == nil {
		// state 已删，清理 registry 索引
		_ = store.RemoveFromRegistry(ctx, app, stream)
		return
	}

	// UUID 校验：不匹配说明已被新中继覆盖，跳过
	if latest.UUID != uuid {
		logx.Infof("[registry-scanner] UUID mismatch, skipping: app=%s stream=%s entryUUID=%s latestUUID=%s", app, stream, uuid, latest.UUID)
		_ = store.RemoveFromRegistry(ctx, app, stream)
		return
	}

	// 检查是否有活跃租约
	has, err := store.HasLease(ctx, app, stream)
	if err != nil {
		logx.Errorf("[registry-scanner] check lease failed: app=%s stream=%s err=%v", app, stream, err)
		return
	}
	if has {
		// 有 lease 说明正在运行，刷新 registry score
		_ = store.RenewRegistry(ctx, app, stream)
		return
	}
	// 无 lease + pending=true → 通常跳过（等 Asynq 任务消费）
	if latest.PendingReconcile {
		// 最终补偿：score 超过 pendingStaleThreshold 说明 Asynq 任务可能丢失，重置 pending 并重新入队
		scoreAge := time.Since(time.Unix(entry.Score, 0))
		if scoreAge > relay.PendingStaleThreshold {
			logx.Infof("[registry-scanner] pending stuck app=%s stream=%s score_age=%v, resetting and re-enqueuing", app, stream, scoreAge)
			latest.PendingReconcile = false
			_ = store.SaveState(ctx, latest)
		} else {
			return
		}
	}
	// 无 lease + !pending → 入队 → 置 pending=true（先入队再更新，入队失败不改 state）
	if err := registry.EnqueueReconcile(ctx, relay.ReconcilePayload{App: app, Stream: stream, UUID: uuid, RetryCount: latest.RetryCount}); err != nil {
		logx.Errorf("[registry-scanner] enqueue reconcile failed: app=%s stream=%s err=%v", app, stream, err)
		return
	}
	latest.PendingReconcile = true
	if err := store.SaveState(ctx, latest); err != nil {
		logx.Errorf("[registry-scanner] save state failed: app=%s stream=%s err=%v", app, stream, err)
	}
	logx.Infof("[registry-scanner] enqueue reconcile: app=%s stream=%s retryCount=%d", app, stream, latest.RetryCount)
}
