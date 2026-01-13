package logic

import (
	"context"
	"database/sql"
	"time"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type TerminatePlanExecItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTerminatePlanExecItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TerminatePlanExecItemLogic {
	return &TerminatePlanExecItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 终止执行项
func (l *TerminatePlanExecItemLogic) TerminatePlanExecItem(in *trigger.TerminatePlanExecItemReq) (*trigger.TerminatePlanExecItemRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 查询执行项
	execItem, err := l.svcCtx.PlanExecItemModel.FindOne(l.ctx, in.Id)
	if err != nil {
		if err == sqlx.ErrNotFound {
			return &trigger.TerminatePlanExecItemRes{}, nil
		}
		return nil, err
	}

	if execItem.Status == 1 || execItem.Status == 2 || execItem.Status == 5 {
		return &trigger.TerminatePlanExecItemRes{}, nil
	}

	// 执行事务
	err = l.svcCtx.PlanModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		// 更新执行项状态为已终止
		execItem.Status = 5 // 5-已终止
		execItem.IsTerminated = 1
		execItem.IsPaused = 0
		execItem.TerminatedTime = sql.NullTime{Time: time.Now(), Valid: true}
		execItem.TerminatedReason = sql.NullString{String: in.Reason, Valid: in.Reason != ""}
		execItem.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(in.CurrentUser), Valid: tool.GetCurrentUserId(in.CurrentUser) != ""}

		// 更新执行项
		_, transErr := l.svcCtx.PlanExecItemModel.Update(ctx, tx, execItem)
		if transErr != nil {
			return transErr
		}

		return nil
	})

	if err != nil {
		return &trigger.TerminatePlanExecItemRes{}, nil
	}

	return &trigger.TerminatePlanExecItemRes{}, nil
}
