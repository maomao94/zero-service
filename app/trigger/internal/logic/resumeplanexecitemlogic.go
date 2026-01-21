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
		if err == sqlx.ErrNotFound {
			return &trigger.ResumePlanExecItemRes{}, nil
		}
		return nil, err
	}

	if execItem.Status != int64(model.StatusPaused) {
		return nil, errors.BadRequest("", "计划执行项非暂停,不可恢复")
	}

	_, err = l.svcCtx.PlanModel.FindOne(l.ctx, execItem.PlanPk)
	if err != nil {
		return nil, err
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
