package logic

import (
	"context"
	"database/sql"
	"errors"
	"time"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/common/tool"
	"zero-service/model"
	"zero-service/third_party/extproto"

	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
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
	if strutil.IsBlank(in.Id) && strutil.IsBlank(in.ExecId) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "参数错误")
	}

	// 查询执行项
	db := l.svcCtx.DB.WithContext(l.ctx).DB
	var execItem gormmodel.PlanExecItem
	if !strutil.IsBlank(in.Id) {
		err = db.Where("id = ?", in.Id).First(&execItem).Error
	} else {
		err = db.Where("exec_id = ?", in.ExecId).First(&execItem).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &trigger.ResumePlanExecItemRes{}, nil
		}
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_DB, "查询执行项失败")
	}

	if execItem.Status != model.StatusPaused {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "计划执行项非暂停,不可恢复")
	}

	var plan gormmodel.Plan
	if err := db.Where("id = ?", execItem.PlanPk).First(&plan).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询计划失败")
	}

	// 执行事务
	err = db.Transaction(func(tx *gorm.DB) error {
		execItem.Status = model.StatusWaiting
		execItem.PausedTime = sql.NullTime{}
		execItem.PausedReason = sql.NullString{}
		execItem.UpdateUser = sql.NullString{String: tool.GetCurrentUserId(l.ctx, nil), Valid: tool.GetCurrentUserId(l.ctx, nil) != ""}
		execItem.UpdateTime = time.Now()

		// 更新执行项
		return tx.Save(&execItem).Error
	})

	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "恢复执行项事务失败")
	}

	planscope.ExecScope(&execItem).Logger(l.ctx).Info("RPC 恢复执行项：执行项状态已更新，事务已提交")
	return &trigger.ResumePlanExecItemRes{}, nil
}
