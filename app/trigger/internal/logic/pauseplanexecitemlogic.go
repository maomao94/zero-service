package logic

import (
	"context"
	"database/sql"
	"time"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

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

	if execItem.Status == 1 || execItem.Status == 2 || execItem.Status == 5 {
		return &trigger.PausePlanExecItemRes{}, nil
	}

	// 执行事务
	err = l.svcCtx.PlanModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		// 更新执行项状态为暂停
		execItem.Status = 6 // 6-暂停
		execItem.IsPaused = 1
		execItem.IsTerminated = 0
		execItem.PausedTime = sql.NullTime{Time: time.Now(), Valid: true}
		execItem.PausedReason = in.Reason
		execItem.UpdateUser = in.CurrentUser.UserId

		// 更新执行项
		_, transErr := l.svcCtx.PlanExecItemModel.Update(ctx, tx, execItem)
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
