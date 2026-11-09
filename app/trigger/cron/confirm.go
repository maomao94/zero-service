package cron

import (
	"context"
	"time"

	"zero-service/app/trigger/internal/confirm"
	"zero-service/app/trigger/internal/logic"
	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/carbonx"
	"zero-service/model"

	"github.com/zeromicro/go-zero/core/logx"
)

// planConfirmScanInterval 批次确认扫描周期。执行项扫描可能持续繁忙，确认扫描无需同频。
const planConfirmScanInterval = 5 * time.Second

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

	if pending, err := gormmodel.FindPendingPlanConfirmNotify(ctx, db, now); err != nil {
		scope.Logger(ctx).Errorf(scope.LogMessage("批次确认扫描：查询待通知记录失败: %v"), err)
	} else if pending != nil {
		s.withConfirmLock(ctx, pending.BatchPk, func() {
			latest, findErr := gormmodel.FindPlanConfirmByTarget(ctx, db, model.ConfirmTypeBatch, pending.TargetPk)
			if findErr != nil {
				scope.Logger(ctx).Errorf(scope.LogMessage("批次确认扫描：锁内重新加载失败: %v"), findErr)
				return
			}
			if latest == nil || latest.ConfirmStatus != model.PlanConfirmStatusWaiting ||
				latest.NotifiedTime.Valid || latest.NotifyAt.After(time.Now()) ||
				!latest.Deadline.After(time.Now()) {
				return
			}
			if err := gormmodel.MarkPlanConfirmNotified(ctx, db, latest.Id, time.Now()); err != nil {
				scope.Logger(ctx).Errorf(scope.LogMessage("批次确认扫描：标记通知时间失败: %v"), err)
				return
			}
			s.notifyBatchConfirm(ctx, latest)
		})
	}

	if pending, err := gormmodel.FindPendingPlanConfirmExpire(ctx, db, now); err != nil {
		scope.Logger(ctx).Errorf(scope.LogMessage("批次确认扫描：查询超时记录失败: %v"), err)
	} else if pending != nil {
		s.withConfirmLock(ctx, pending.BatchPk, func() {
			latest, findErr := gormmodel.FindPlanConfirmByTarget(ctx, db, model.ConfirmTypeBatch, pending.TargetPk)
			if findErr != nil {
				scope.Logger(ctx).Errorf(scope.LogMessage("批次确认扫描：锁内重新加载失败: %v"), findErr)
				return
			}
			if latest == nil || latest.ConfirmStatus != model.PlanConfirmStatusWaiting ||
				time.Now().Before(latest.Deadline) {
				return
			}
			if err := gormmodel.ExpirePlanConfirm(ctx, db, latest.Id, confirm.ReasonExpired, time.Now()); err != nil {
				scope.Logger(ctx).Errorf(scope.LogMessage("批次确认扫描：标记超时失败: %v"), err)
				return
			}
			s.expireBatchConfirm(ctx, latest)
		})
	}
}

// withConfirmLock 以批次为粒度串行化通知、确认、拒绝和超时处理。
func (s *CronService) withConfirmLock(ctx context.Context, batchPk string, fn func()) {
	release, ok, err := confirm.AcquireLock(ctx, s.svcCtx.Redis, batchPk)
	if err != nil {
		planscope.CronLockScope().Logger(ctx).Errorf("批次确认锁：获取失败 batch_pk=%s err=%v", batchPk, err)
		return
	}
	if !ok {
		return
	}
	defer release()
	fn()
}

func (s *CronService) notifyBatchConfirm(ctx context.Context, record *gormmodel.PlanConfirm) {
	db := s.svcCtx.DB.WithContext(ctx).DB
	scope := planscope.CronLockScope().WithFields(
		logx.Field("plan_pk", record.PlanPk),
		logx.Field("batch_pk", record.BatchPk),
	)
	log := scope.Logger(ctx)

	var batch gormmodel.PlanBatch
	if err := db.Where("id = ?", record.BatchPk).First(&batch).Error; err != nil {
		log.Errorf(scope.LogMessage("批次确认通知：加载批次失败: %v"), err)
		return
	}
	var plan gormmodel.Plan
	if err := db.Where("id = ?", batch.PlanPk).First(&plan).Error; err != nil {
		log.Errorf(scope.LogMessage("批次确认通知：加载计划失败: %v"), err)
		return
	}
	if plan.Status == model.PlanStatusTerminated || plan.FinishedTime.Valid ||
		batch.Status == model.PlanStatusTerminated || batch.FinishedTime.Valid {
		log.Info(scope.LogMessage("批次确认通知：计划或批次已结束，跳过本次推送"))
		return
	}
	if plan.Status != model.PlanStatusEnabled || batch.Status != model.PlanStatusEnabled {
		log.Info(scope.LogMessage("批次确认通知：计划或批次非启用状态，跳过本次推送"))
		return
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
		return
	}
	log.WithFields(
		logx.Field("deadline", payload.ConfirmDeadline),
		logx.Field("item_count", payload.ItemCount),
	).Info(scope.LogMessage("批次确认通知：已推送 Web"))
}

func (s *CronService) expireBatchConfirm(ctx context.Context, record *gormmodel.PlanConfirm) {
	scope := planscope.CronLockScope().WithFields(
		logx.Field("plan_pk", record.PlanPk),
		logx.Field("batch_pk", record.BatchPk),
	)
	log := scope.Logger(ctx)

	// 超时终止直接复用批次终止逻辑，统一处理批次状态、确认记录和收尾通知。
	if _, err := logic.NewTerminatePlanBatchLogic(ctx, s.svcCtx).TerminatePlanBatch(&trigger.TerminatePlanBatchReq{
		Id:     record.BatchPk,
		Reason: confirm.ReasonExpired,
	}); err != nil {
		log.Errorf(scope.LogMessage("批次确认超时：终止批次失败: %v"), err)
		return
	}
	log.WithFields(
		logx.Field("deadline", carbonx.FormatDateTime(record.Deadline)),
	).Info(scope.LogMessage("批次确认超时：已复用终止批次逻辑"))
}
