package logic

import (
	"context"
	"database/sql"
	"errors"
	"time"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/common/authctx"
	"zero-service/common/errx"
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
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_PARAM, "参数错误")
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
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_DB, "查询执行项失败")
	}

	if execItem.Status != model.StatusPaused {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划执行项非暂停,不可恢复")
	}

	var plan gormmodel.Plan
	if err := db.Where("id = ?", execItem.PlanPk).First(&plan).Error; err != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询计划失败")
	}

	// 更新执行项状态为等待调度；更新带上读取时的状态条件，期间被终止或抢占时不覆盖已变更的状态。
	now := time.Now()
	result := db.Model(&gormmodel.PlanExecItem{}).
		Where("id = ?", execItem.Id).
		Where("status = ?", model.StatusPaused).
		Updates(map[string]any{
			"status":        model.StatusWaiting,
			"paused_time":   sql.NullTime{},
			"paused_reason": sql.NullString{},
			"update_user":   sql.NullString{String: authctx.GetUserId(l.ctx), Valid: authctx.GetUserId(l.ctx) != ""},
			"update_time":   now,
		})
	if result.Error != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, result.Error, "恢复执行项失败")
	}
	if result.RowsAffected == 0 {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "执行项状态已变化，请刷新后重试")
	}
	execItem.Status = model.StatusWaiting

	planscope.ExecScope(&execItem).Logger(l.ctx).Info("RPC 恢复执行项：执行项状态已更新")
	return &trigger.ResumePlanExecItemRes{}, nil
}
