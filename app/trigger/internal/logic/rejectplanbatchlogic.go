package logic

import (
	"context"
	"errors"
	"fmt"

	"zero-service/app/trigger/internal/confirm"
	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/errx"
	"zero-service/model"
	"zero-service/third_party/extproto"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type RejectPlanBatchLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRejectPlanBatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RejectPlanBatchLogic {
	return &RejectPlanBatchLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 拒绝启动计划批次（批次级二次确认）
func (l *RejectPlanBatchLogic) RejectPlanBatch(in *trigger.RejectPlanBatchReq) (*trigger.RejectPlanBatchRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if strutil.IsBlank(in.Id) && strutil.IsBlank(in.BatchId) {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_PARAM, "参数错误")
	}

	db := l.svcCtx.DB.WithContext(l.ctx).DB
	var batch gormmodel.PlanBatch
	var err error
	if !strutil.IsBlank(in.Id) {
		err = db.Where("id = ?", in.Id).First(&batch).Error
	} else {
		err = db.Where("batch_id = ?", in.BatchId).First(&batch).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errx.NewErrorByPbCode(extproto.Code__1_00_RECORD_NOT_EXIST)
		}
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询计划批次失败")
	}

	release, locked, lockErr := confirm.AcquireLock(l.ctx, l.svcCtx.Redis, batch.Id)
	if lockErr != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_CACHE, lockErr, "获取批次确认锁失败")
	}
	if !locked {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "批次确认处理中，请稍后重试")
	}
	defer release()
	if err := db.Where("id = ?", batch.Id).First(&batch).Error; err != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询计划批次失败")
	}

	var plan gormmodel.Plan
	if err := db.Where("id = ?", batch.PlanPk).First(&plan).Error; err != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询计划失败")
	}
	record, err := gormmodel.FindPlanConfirmByTarget(l.ctx, db, model.ConfirmTypeBatch, batch.Id)
	if err != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询批次确认记录失败")
	}
	if record == nil {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "该批次未配置二次确认")
	}
	switch record.ConfirmStatus {
	case model.PlanConfirmStatusConfirmed:
		// 已确认：如果批次已经终止（拒绝/超时/人工终止），重复拒绝按幂等成功处理。
		if batch.Status == model.PlanStatusTerminated || batch.FinishedTime.Valid ||
			plan.Status == model.PlanStatusTerminated || plan.FinishedTime.Valid {
			return &trigger.RejectPlanBatchRes{}, nil
		}
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "批次已确认，不可拒绝")
	}
	if plan.Status == model.PlanStatusTerminated || plan.FinishedTime.Valid {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划状态已结束,不可拒绝")
	}
	if batch.Status == model.PlanStatusTerminated || batch.FinishedTime.Valid {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划批次状态已结束,不可拒绝")
	}

	reason := confirm.ReasonRejected
	if !strutil.IsBlank(in.Reason) {
		reason = fmt.Sprintf("%s: %s", confirm.ReasonRejected, in.Reason)
	}
	// TerminatePlanBatchReq.reason 与 RejectPlanBatchReq.reason 的 max_len 都是 2000，
	// 拼接错误前缀后按字符数截断，避免内部复用终止逻辑时被 pb 校验拒绝。
	reason = truncateRunes(reason, terminateReasonMaxRunes)

	// 拒绝 = 终止批次，直接复用终止逻辑，统一处理批次状态、确认记录和收尾通知。
	if _, err := NewTerminatePlanBatchLogic(l.ctx, l.svcCtx).TerminatePlanBatch(&trigger.TerminatePlanBatchReq{
		Id:     batch.Id,
		Reason: reason,
	}); err != nil {
		return nil, err
	}

	planscope.BatchScope(&plan, &batch).WithFields(
		logx.Field("reason", reason),
	).Logger(l.ctx).Info("RPC 拒绝启动批次：已复用终止批次逻辑")
	return &trigger.RejectPlanBatchRes{}, nil
}

// terminateReasonMaxRunes 与 trigger.proto 中 TerminatePlanBatchReq.reason /
// RejectPlanBatchReq.reason 的 max_len 保持一致，也与 plan_batch.terminated_reason、
// plan_confirm.reason 的 size 对齐（protoc-gen-validate 按 utf8 RuneCount 计算）。
const terminateReasonMaxRunes = 2000

// truncateRunes 按字符数截断，避免按字节截断中文导致乱码。
func truncateRunes(value string, maxRunes int) string {
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes-1]) + "…"
}
