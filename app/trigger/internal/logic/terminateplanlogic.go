package logic

import (
	"context"
	"database/sql"
	"time"
	"zero-service/facade/streamevent/streamevent"

	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/tool"
	"zero-service/model"
	"zero-service/third_party/extproto"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type TerminatePlanLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTerminatePlanLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TerminatePlanLogic {
	return &TerminatePlanLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 终止计划
func (l *TerminatePlanLogic) TerminatePlan(in *trigger.TerminatePlanReq) (*trigger.TerminatePlanRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 检查参数
	if strutil.IsBlank(in.Id) && strutil.IsBlank(in.PlanId) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "参数错误")
	}

	// 查询计划
	db := l.svcCtx.DB.WithContext(l.ctx).DB
	var plan gormmodel.Plan
	if !strutil.IsBlank(in.Id) {
		err = db.Where("id = ?", in.Id).First(&plan).Error
	} else {
		err = db.Where("plan_id = ?", in.PlanId).First(&plan).Error
	}
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询计划失败")
	}

	if plan.Status == int64(model.PlanStatusTerminated) || plan.FinishedTime.Valid {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划状态已结束,无需终止")
	}

	// 执行事务
	err = db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		// 更新计划状态为已终止
		plan.Status = int64(model.PlanStatusTerminated) // 终止
		plan.PausedTime = sql.NullTime{Time: time.Now(), Valid: true}
		plan.PausedReason = sql.NullString{String: in.Reason, Valid: in.Reason != ""}
		plan.FinishedTime = sql.NullTime{Time: now, Valid: true}
		plan.TerminatedReason = sql.NullString{String: in.Reason, Valid: in.Reason != ""}
		plan.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(l.ctx, in.CurrentUser), Valid: tool.GetCurrentUserId(l.ctx, in.CurrentUser) != ""}

		// 更新计划
		transErr := tx.Save(&plan).Error
		if transErr != nil {
			return transErr
		}

		// 更新批次
		transErr = tx.Model(&gormmodel.PlanBatch{}).
			Where("plan_id = ?", plan.PlanId).
			Where("status != ?", int64(model.PlanStatusTerminated)).
			Where("finished_time IS NULL").
			Updates(map[string]any{
				"status":            int64(model.PlanStatusTerminated),
				"terminated_reason": sql.NullString{String: in.Reason, Valid: in.Reason != ""},
				"finished_time":     now,
				"update_user":       sql.NullString{String: tool.GetCurrentUserId(l.ctx, in.CurrentUser), Valid: tool.GetCurrentUserId(l.ctx, in.CurrentUser) != ""},
			}).Error
		return transErr
	})

	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "终止计划事务失败")
	}
	planPlanReq := streamevent.NotifyPlanEventReq{
		EventType:  streamevent.PlanEventType_PLAN_FINISHED,
		PlanId:     plan.PlanId,
		PlanType:   plan.Type.String,
		Attributes: map[string]string{},
	}
	planLog := planscope.PlanScope(&plan).Logger(l.ctx)
	planLog.WithFields(logx.Field("notify_event", planscope.NotifyEventPlanFinished)).Info("下游通知：调用 NotifyPlanEvent（计划收尾）")
	l.svcCtx.StreamEventCli.NotifyPlanEvent(l.ctx, &planPlanReq)

	planLog.Info("RPC 终止计划：计划状态已更新，事务已提交")
	return &trigger.TerminatePlanRes{}, nil
}
