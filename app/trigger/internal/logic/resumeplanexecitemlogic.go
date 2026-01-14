package logic

import (
	"context"
	"database/sql"
	"time"
	"zero-service/common/tool"
	"zero-service/model"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ResumePlanExecItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResumePlanExecItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResumePlanExecItemLogic {
	return &ResumePlanExecItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 恢复执行项
func (l *ResumePlanExecItemLogic) ResumePlanExecItem(in *trigger.ResumePlanExecItemReq) (*trigger.ResumePlanExecItemRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	execItem, err := l.svcCtx.PlanExecItemModel.FindOne(l.ctx, in.Id)
	if err != nil {
		if err == sqlx.ErrNotFound {
			return &trigger.ResumePlanExecItemRes{}, nil
		}
		return nil, err
	}

	if execItem.Status != int64(model.StatusPaused) {
		return &trigger.ResumePlanExecItemRes{}, nil
	}

	plan, err := l.svcCtx.PlanModel.FindOne(l.ctx, execItem.PlanPk)
	if err != nil {
		if err == sqlx.ErrNotFound {
			return &trigger.ResumePlanExecItemRes{}, nil
		}
		return nil, err
	}

	if plan.Status == int64(model.PlanStatusTerminated) {
		return &trigger.ResumePlanExecItemRes{}, nil
	}

	// 执行事务
	err = l.svcCtx.PlanModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		execItem.Status = int64(model.StatusWaiting)
		execItem.PausedTime = sql.NullTime{}
		execItem.PausedReason = sql.NullString{}
		execItem.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(in.CurrentUser), Valid: tool.GetCurrentUserId(in.CurrentUser) != ""}
		execItem.UpdateTime = time.Now()

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

	return &trigger.ResumePlanExecItemRes{}, nil
}
