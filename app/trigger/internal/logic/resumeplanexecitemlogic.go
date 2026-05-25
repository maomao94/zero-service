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
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "参数错误")
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
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_DB, "查询执行项失败")
	}

	if execItem.Status != int64(model.StatusPaused) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划执行项非暂停,不可恢复")
	}

	_, err = l.svcCtx.PlanModel.FindOne(l.ctx, execItem.PlanPk)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询计划失败")
	}

	// 执行事务
	err = l.svcCtx.PlanModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		execItem.Status = int64(model.StatusWaiting)
		execItem.PausedTime = sql.NullTime{}
		execItem.PausedReason = sql.NullString{}
		execItem.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(l.ctx, in.CurrentUser), Valid: tool.GetCurrentUserId(l.ctx, in.CurrentUser) != ""}
		execItem.UpdateTime = time.Now()

		// 更新执行项
		transErr := l.svcCtx.PlanExecItemModel.UpdateWithVersion(ctx, tx, execItem)
		if transErr != nil {
			return transErr
		}

		return nil
	})

	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "恢复执行项事务失败")
	}

	planscope.ExecScope(execItem).Logger(l.ctx).Info("RPC 恢复执行项：执行项状态已更新，事务已提交")
	return &trigger.ResumePlanExecItemRes{}, nil
}
