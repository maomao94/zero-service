package logic

import (
	"context"
	"database/sql"
	"time"
	"zero-service/common/tool"
	"zero-service/model"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

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

	// 查询执行项
	execItem, err := l.svcCtx.PlanExecItemModel.FindOne(l.ctx, in.Id)
	if err != nil {
		if err == sqlx.ErrNotFound {
			return &trigger.PausePlanExecItemRes{}, nil
		}
		return nil, err
	}

	if execItem.Status == int64(model.StatusCompleted) || execItem.Status == int64(model.StatusTerminated) || execItem.Status == int64(model.StatusPaused) {
		return &trigger.PausePlanExecItemRes{}, nil
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
