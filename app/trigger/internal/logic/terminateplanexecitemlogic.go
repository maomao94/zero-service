package logic

import (
	"context"
	"database/sql"
	"time"
	"zero-service/app/trigger/internal/confirm"
	"zero-service/app/trigger/internal/planscope"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/authctx"
	"zero-service/common/errx"

	"zero-service/model"
	"zero-service/third_party/extproto"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
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
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询执行项失败")
	}

	// 查询计划批次
	var planBatch gormmodel.PlanBatch
	if err := db.Where("id = ?", execItem.BatchPk).First(&planBatch).Error; err != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询计划批次失败")
	}
	// 查询计划
	var plan gormmodel.Plan
	if err := db.Where("plan_id = ?", execItem.PlanId).First(&plan).Error; err != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询计划失败")
	}
	if plan.Status == model.PlanStatusTerminated || plan.FinishedTime.Valid {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划状态已结束,无需终止")
	}

	if planBatch.Status == model.PlanStatusTerminated || planBatch.FinishedTime.Valid {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划批次状态已结束,无需终止")
	}

	if execItem.Status == model.StatusCompleted || execItem.Status == model.StatusTerminated {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "执行项状态已结束,无需终止")
	}

	scope := planscope.ExecScope(&execItem)
	log := scope.Logger(l.ctx)

	// 未填写终止原因时与计划/批次终止保持一致，写入默认原因，避免执行项终止原因为空。
	closeReason := in.Reason
	if closeReason == "" {
		closeReason = "执行项被终止"
	}

	// 更新执行项为终止态；更新带上读取时的状态条件（未结束），期间被其它链路推进到终态时不覆盖。
	now := time.Now()
	result := db.Model(&gormmodel.PlanExecItem{}).
		Where("id = ?", execItem.Id).
		Where("status NOT IN ?", []int{model.StatusCompleted, model.StatusTerminated}).
		Updates(map[string]any{
			"status":            model.StatusTerminated,
			"terminated_reason": sql.NullString{String: closeReason, Valid: true},
			"paused_time":       sql.NullTime{},
			"paused_reason":     sql.NullString{},
			"update_user":       sql.NullString{String: authctx.GetUserId(l.ctx), Valid: authctx.GetUserId(l.ctx) != ""},
			"update_time":       now,
		})
	if result.Error != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, result.Error, "终止执行项失败")
	}
	if result.RowsAffected == 0 {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "执行项状态已变化，请刷新后重试")
	}
	execItem.Status = model.StatusTerminated
	batch := &gormmodel.PlanBatch{PlanPk: execItem.PlanPk, PlanId: execItem.PlanId, BatchId: execItem.BatchId}
	batch.Id = execItem.BatchPk
	if _, _, notifyErr := confirm.NotifyBatchFinishedIfNeeded(l.ctx, db, l.svcCtx.StreamEventCli, &plan, batch, closeReason); notifyErr != nil {
		log.Errorf("更新批次/计划 finished_time 失败: %v", notifyErr)
	}

	log.Info("RPC 终止执行项：执行项状态已更新")
	return &trigger.TerminatePlanExecItemRes{}, nil
}
