package cron

import (
	"context"
	"time"

	"zero-service/app/trigger/internal/confirm"
	"zero-service/app/trigger/internal/logic"
	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/authctx"
	"zero-service/common/carbonx"
	"zero-service/common/gormx"
	"zero-service/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

const (
	// planConfirmScanInterval 批次确认扫描周期。执行项扫描可能持续繁忙，确认扫描无需同频。
	planConfirmScanInterval = 5 * time.Second
	// planConfirmScanBatch 单次扫描最多处理的通知/超时记录数，避免积压时长期占用调度循环。
	planConfirmScanBatch = 50
)

// ScanPlanConfirm 扫描批次二次确认：
//   - 到通知时间且未通知的，推送一次 Web 通知；
//   - 超过截止时间且未确认的，终止批次及其执行项。
//
// 两类处理都通过条件更新抢占，多实例并发下不会重复通知或重复终止。
func (s *CronService) ScanPlanConfirm() {
	ctx := context.Background()
	db := s.svcCtx.DB.WithContext(ctx).DB
	now := time.Now()
	scope := planscope.CronLockScope()

	for i := 0; i < planConfirmScanBatch; i++ {
		pending, err := gormmodel.FindPendingPlanConfirmNotify(ctx, db, now)
		if err != nil {
			scope.Logger(ctx).Errorf(scope.LogMessage("批次确认扫描：查询待通知记录失败: %v"), err)
			break
		}
		if pending == nil || !s.handleConfirmNotify(ctx, db, pending) {
			break
		}
	}

	for i := 0; i < planConfirmScanBatch; i++ {
		pending, err := gormmodel.FindPendingPlanConfirmExpire(ctx, db, now)
		if err != nil {
			scope.Logger(ctx).Errorf(scope.LogMessage("批次确认扫描：查询超时记录失败: %v"), err)
			break
		}
		if pending == nil || !s.handleConfirmExpire(ctx, db, pending) {
			break
		}
	}
}

// handleConfirmNotify 处理一条待通知记录，返回是否已推进该记录。
func (s *CronService) handleConfirmNotify(ctx context.Context, db *gorm.DB, pending *gormmodel.PlanConfirm) bool {
	notified := false
	ran := s.withConfirmLock(ctx, pending.BatchPk, func() {
		// 扫描链路静默成功 SQL，与 LockTriggerItem 保持一致；该函数同时被 RPC 复用，故在调用侧加开关。
		latest, err := gormmodel.FindPlanConfirmByTarget(gormx.WithoutSQLTrace(ctx), db, model.ConfirmTypeBatch, pending.TargetPk)
		if err != nil {
			planscope.CronLockScope().Logger(ctx).Errorf(planscope.CronLockScope().LogMessage("批次确认扫描：锁内重新加载失败: %v"), err)
			return
		}
		if latest == nil || latest.ConfirmStatus != model.PlanConfirmStatusWaiting ||
			latest.NotifiedTime.Valid || latest.NotifyAt.After(time.Now()) ||
			!latest.Deadline.After(time.Now()) {
			return
		}
		notified = s.notifyBatchConfirm(ctx, latest)
	})
	return ran && notified
}

// handleConfirmExpire 处理一条超时记录，返回是否已推进该记录。
func (s *CronService) handleConfirmExpire(ctx context.Context, db *gorm.DB, pending *gormmodel.PlanConfirm) bool {
	expired := false
	ran := s.withConfirmLock(ctx, pending.BatchPk, func() {
		// 扫描链路静默成功 SQL，与 LockTriggerItem 保持一致；该函数同时被 RPC 复用，故在调用侧加开关。
		latest, err := gormmodel.FindPlanConfirmByTarget(gormx.WithoutSQLTrace(ctx), db, model.ConfirmTypeBatch, pending.TargetPk)
		if err != nil {
			planscope.CronLockScope().Logger(ctx).Errorf(planscope.CronLockScope().LogMessage("批次确认扫描：锁内重新加载失败: %v"), err)
			return
		}
		if latest == nil || latest.ConfirmStatus != model.PlanConfirmStatusWaiting ||
			time.Now().Before(latest.Deadline) {
			return
		}
		expired = s.expireBatchConfirm(ctx, latest)
	})
	return ran && expired
}

// withConfirmLock 以批次为粒度串行化通知、确认、拒绝和超时处理，返回是否执行了 fn。
func (s *CronService) withConfirmLock(ctx context.Context, batchPk string, fn func()) bool {
	release, ok, err := confirm.AcquireLock(ctx, s.svcCtx.Redis, batchPk)
	if err != nil {
		planscope.CronLockScope().Logger(ctx).Errorf(planscope.CronLockScope().LogMessage("批次确认锁：获取失败 batch_pk=%s err=%v"), batchPk, err)
		return false
	}
	if !ok {
		return false
	}
	defer release()
	fn()
	return true
}

// notifyBatchConfirm 推送批次待确认通知，返回是否已推进记录（notified_time 已写入）。
func (s *CronService) notifyBatchConfirm(ctx context.Context, record *gormmodel.PlanConfirm) bool {
	db := s.svcCtx.DB.WithContext(ctx).DB
	scope := planscope.CronLockScope().WithFields(
		logx.Field("plan_pk", record.PlanPk),
		logx.Field("batch_pk", record.BatchPk),
	)
	log := scope.Logger(ctx)

	var batch gormmodel.PlanBatch
	if err := db.Where("id = ?", record.BatchPk).First(&batch).Error; err != nil {
		log.Errorf(scope.LogMessage("批次确认通知：加载批次失败: %v"), err)
		return false
	}
	var plan gormmodel.Plan
	if err := db.Where("id = ?", batch.PlanPk).First(&plan).Error; err != nil {
		log.Errorf(scope.LogMessage("批次确认通知：加载计划失败: %v"), err)
		return false
	}
	if plan.Status == model.PlanStatusTerminated || plan.FinishedTime.Valid ||
		batch.Status == model.PlanStatusTerminated || batch.FinishedTime.Valid {
		log.Info(scope.LogMessage("批次确认通知：计划或批次已结束，跳过本次推送"))
		return false
	}
	if plan.Status != model.PlanStatusEnabled || batch.Status != model.PlanStatusEnabled {
		// 暂停等非启用状态不写 notified_time，恢复后仍可补发通知。
		log.Info(scope.LogMessage("批次确认通知：计划或批次非启用状态，跳过本次推送"))
		return false
	}
	// 状态确认后再标记已通知，避免暂停/终止竞态下永久丢失通知。
	if err := gormmodel.MarkPlanConfirmNotified(ctx, db, record.Id, time.Now()); err != nil {
		log.Errorf(scope.LogMessage("批次确认通知：标记通知时间失败: %v"), err)
		return false
	}

	itemCount, err := gormmodel.GetBatchTotalExecItems(ctx, db, batch.Id)
	if err != nil {
		log.Errorf(scope.LogMessage("批次确认通知：统计执行项数量失败: %v"), err)
		itemCount = 0
	}

	payload := confirm.BatchConfirmPayload{
		PlanPk:          plan.Id,
		PlanId:          plan.PlanId,
		PlanName:        plan.PlanName.String,
		BatchPk:         batch.Id,
		BatchId:         batch.BatchId,
		BatchNum:        batch.BatchNum.String,
		DeptCode:        batch.DeptCode.String,
		PlanTriggerTime: carbonx.FormatDateTime(batch.PlanTriggerTime.Time),
		ConfirmDeadline: carbonx.FormatDateTime(record.Deadline),
		LeadMinutes:     record.LeadMinutes,
		ItemCount:       itemCount,
		NotifiedAt:      carbonx.NowDateTime(),
	}
	if err := s.svcCtx.PlanConfirmNotifier.NotifyBatchConfirmRequired(ctx, payload); err != nil {
		// 通知失败不回滚 notified_time，避免下游恢复后重复轰炸；前端仍可通过批次列表看到待确认状态。
		log.Errorf(scope.LogMessage("批次确认通知：socketpush 推送失败: %v"), err)
		return true
	}
	log.WithFields(
		logx.Field("deadline", payload.ConfirmDeadline),
		logx.Field("item_count", payload.ItemCount),
	).Info(scope.LogMessage("批次确认通知：已推送 Web"))
	return true
}

// expireBatchConfirm 超时终止批次，返回是否成功推进。
func (s *CronService) expireBatchConfirm(ctx context.Context, record *gormmodel.PlanConfirm) bool {
	scope := planscope.CronLockScope().WithFields(
		logx.Field("plan_pk", record.PlanPk),
		logx.Field("batch_pk", record.BatchPk),
	)
	log := scope.Logger(ctx)

	// 超时终止直接复用批次终止逻辑。确认记录由终止事务内的 ClosePlanConfirmsByBatch 关闭，
	// 保证「关闭确认」和「终止批次」原子生效，避免终止失败时提前放行执行项。
	ctx = authctx.WithUserID(ctx, confirm.SystemUser)
	if _, err := logic.NewTerminatePlanBatchLogic(ctx, s.svcCtx).TerminatePlanBatch(&trigger.TerminatePlanBatchReq{
		Id:     record.BatchPk,
		Reason: confirm.ReasonExpired,
	}); err != nil {
		log.Errorf(scope.LogMessage("批次确认超时：终止批次失败: %v"), err)
		return false
	}
	log.WithFields(
		logx.Field("deadline", carbonx.FormatDateTime(record.Deadline)),
	).Info(scope.LogMessage("批次确认超时：已复用终止批次逻辑"))
	return true
}
