package logic

import (
	"context"
	"database/sql"
	"time"
	"zero-service/common/tool"
	"zero-service/model"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/songzhibin97/gkit/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type PausePlanExecItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPausePlanExecItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PausePlanExecItemLogic {
	return &PausePlanExecItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 暂停执行项
func (l *PausePlanExecItemLogic) PausePlanExecItem(in *trigger.PausePlanExecItemReq) (*trigger.PausePlanExecItemRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 检查参数
	if in.Id <= 0 && strutil.IsBlank(in.ExecId) {
		return nil, errors.BadRequest("", "参数错误")
	}

	// 查询执行项
	var execItem *model.PlanExecItem
	if in.Id > 0 {
		execItem, err = l.svcCtx.PlanExecItemModel.FindOne(l.ctx, in.Id)
	} else {
		execItem, err = l.svcCtx.PlanExecItemModel.FindOneByExecId(l.ctx, in.ExecId)
	}
	if err != nil {
		return nil, err
	}
	// 查询计划批次
	_, err = l.svcCtx.PlanBatchModel.FindOne(l.ctx, execItem.BatchPk)
	if err != nil {
		return nil, err
	}
	// 查询计划
	_, err = l.svcCtx.PlanModel.FindOneByPlanId(l.ctx, execItem.PlanId)
	if err != nil {
		return nil, err
	}

	if execItem.Status == int64(model.StatusCompleted) || execItem.Status == int64(model.StatusTerminated) || execItem.Status == int64(model.StatusPaused) {
		return nil, errors.BadRequest("", "执行项状态已结束,无需暂停")
	}

	if execItem.Status == int64(model.StatusRunning) {
		return &trigger.PausePlanExecItemRes{}, errors.BadRequest("", "执行项正在运行中，请稍后再试")
	}

	// 执行事务
	err = l.svcCtx.PlanModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		// 更新执行项状态为暂停
		execItem.Status = int64(model.StatusPaused)
		execItem.PausedTime = sql.NullTime{Time: time.Now(), Valid: true}
		execItem.PausedReason = sql.NullString{String: in.Reason, Valid: in.Reason != ""}
		execItem.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(in.CurrentUser), Valid: tool.GetCurrentUserId(in.CurrentUser) != ""}

		// 更新执行项
		transErr := l.svcCtx.PlanExecItemModel.UpdateWithVersion(ctx, tx, execItem)
		if transErr != nil {
			return transErr
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &trigger.PausePlanExecItemRes{}, nil
}
