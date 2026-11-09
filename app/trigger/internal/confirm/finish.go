package confirm

import (
	"context"

	"zero-service/app/trigger/model/gormmodel"
	"zero-service/facade/streamevent/streamevent"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

const (
	// ReasonRejected 用户明确拒绝启动批次时的终止原因。
	ReasonRejected = "人工拒绝启动批次"
	// ReasonExpired 超过确认截止时间时的终止原因。
	ReasonExpired = "超过确认截止时间，调度平台自动终止"
)

// NotifyBatchFinished 批次已经结束时调用：通知 BATCH_FINISHED，并在计划全部批次结束后补发 PLAN_FINISHED。
// 返回本次是否更新了计划 finished_time。
func NotifyBatchFinished(ctx context.Context, db *gorm.DB, cli streamevent.StreamEventClient, plan *gormmodel.Plan, batch *gormmodel.PlanBatch, reason string) int64 {
	if cli == nil {
		return 0
	}
	notifyBatchFinished(ctx, cli, plan, batch, reason)

	planRows, err := gormmodel.UpdatePlanFinishedTime(ctx, db, batch.PlanPk)
	if err != nil {
		logx.WithContext(ctx).Errorf("更新计划 finished_time 失败: %v", err)
		return 0
	}
	if planRows > 0 {
		notifyPlanFinished(ctx, cli, plan, reason)
	}
	return planRows
}

// NotifyBatchFinishedIfNeeded 执行项结束后调用：只有批次真正收尾时才通知，避免多执行项重复发 BATCH_FINISHED。
// 返回批次与计划本次是否收尾。
func NotifyBatchFinishedIfNeeded(ctx context.Context, db *gorm.DB, cli streamevent.StreamEventClient, plan *gormmodel.Plan, batch *gormmodel.PlanBatch, reason string) (int64, int64) {
	batchRows, err := gormmodel.UpdatePlanBatchFinishedTime(ctx, db, batch.Id)
	if err != nil {
		logx.WithContext(ctx).Errorf("更新批次 finished_time 失败: %v", err)
		return 0, 0
	}
	if batchRows == 0 {
		return 0, 0
	}
	planRows := NotifyBatchFinished(ctx, db, cli, plan, batch, reason)
	return batchRows, planRows
}

// NotifyPlanFinished 计划已经结束时调用：只通知 PLAN_FINISHED。
func NotifyPlanFinished(ctx context.Context, cli streamevent.StreamEventClient, plan *gormmodel.Plan, reason string) {
	if cli == nil {
		return
	}
	notifyPlanFinished(ctx, cli, plan, reason)
}

func notifyBatchFinished(ctx context.Context, cli streamevent.StreamEventClient, plan *gormmodel.Plan, batch *gormmodel.PlanBatch, reason string) {
	req := streamevent.NotifyPlanEventReq{
		EventType:  streamevent.PlanEventType_BATCH_FINISHED,
		PlanId:     batch.PlanId,
		PlanType:   plan.Type.String,
		BatchId:    batch.BatchId,
		Attributes: map[string]string{},
	}
	if reason != "" {
		req.Attributes["reason"] = reason
	}
	if _, err := cli.NotifyPlanEvent(ctx, &req); err != nil {
		logx.WithContext(ctx).Errorf("批次收尾通知失败（BATCH_FINISHED）: %v", err)
	}
}

func notifyPlanFinished(ctx context.Context, cli streamevent.StreamEventClient, plan *gormmodel.Plan, reason string) {
	req := streamevent.NotifyPlanEventReq{
		EventType:  streamevent.PlanEventType_PLAN_FINISHED,
		PlanId:     plan.PlanId,
		PlanType:   plan.Type.String,
		Attributes: map[string]string{},
	}
	if reason != "" {
		req.Attributes["reason"] = reason
	}
	if _, err := cli.NotifyPlanEvent(ctx, &req); err != nil {
		logx.WithContext(ctx).Errorf("计划收尾通知失败（PLAN_FINISHED）: %v", err)
	}
}
