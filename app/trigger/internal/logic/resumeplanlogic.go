package logic

import (
	"context"
	"database/sql"
	"time"
	"zero-service/common/tool"
	"zero-service/model"
	"zero-service/third_party/extproto"

	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ResumePlanLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResumePlanLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResumePlanLogic {
	return &ResumePlanLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 恢复计划
func (l *ResumePlanLogic) ResumePlan(in *trigger.ResumePlanReq) (*trigger.ResumePlanRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 检查参数
	if in.Id <= 0 && strutil.IsBlank(in.PlanId) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "参数错误")
	}

	// 查询计划
	var plan *model.Plan
	if in.Id > 0 {
		plan, err = l.svcCtx.PlanModel.FindOne(l.ctx, in.Id)
	} else {
		plan, err = l.svcCtx.PlanModel.FindOneByPlanId(l.ctx, in.PlanId)
	}
	if err != nil {
		if err == sqlx.ErrNotFound {
			return &trigger.ResumePlanRes{}, nil
		}
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_DB, "查询计划失败")
	}

	// 只有暂停状态的计划才能恢复，已终止的计划无法恢复
	if plan.Status != int64(model.PlanStatusPaused) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划非暂停,不可恢复")
	}

	// 执行事务
	err = l.svcCtx.PlanModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		// 更新计划状态为启用
		plan.Status = int64(model.PlanStatusEnabled)
		plan.PausedTime = sql.NullTime{}
		plan.PausedReason = sql.NullString{}
		plan.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(l.ctx, in.CurrentUser), Valid: tool.GetCurrentUserId(l.ctx, in.CurrentUser) != ""}
		plan.UpdateTime = time.Now()

		// 更新计划
		transErr := l.svcCtx.PlanModel.UpdateWithVersion(ctx, tx, plan)
		if transErr != nil {
			return transErr
		}

		builder := l.svcCtx.PlanBatchModel.UpdateBuilder().
			Set("status", int64(model.PlanStatusEnabled)).
			Set("paused_time", sql.NullTime{}).
			Set("paused_reason", sql.NullString{}).
			Set("update_user", sql.NullString{String: tool.GetCurrentUserId(l.ctx, in.CurrentUser), Valid: tool.GetCurrentUserId(l.ctx, in.CurrentUser) != ""}).
			Where("plan_id = ?", plan.PlanId).
			Where("status = ?", int64(model.PlanStatusPaused))
		_, transErr = l.svcCtx.PlanBatchModel.UpdateWithBuilder(ctx, tx, builder)
		if transErr != nil {
			return transErr
		}

		return nil
	})

	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "恢复计划事务失败")
	}

	planscope.PlanScope(plan).Logger(l.ctx).Info("RPC 恢复计划：计划状态已更新，事务已提交")
	return &trigger.ResumePlanRes{}, nil
}
