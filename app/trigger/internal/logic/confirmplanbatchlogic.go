package logic

import (
	"context"
	"errors"
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
	"gorm.io/gorm"
)

type ConfirmPlanBatchLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewConfirmPlanBatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConfirmPlanBatchLogic {
	return &ConfirmPlanBatchLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 确认计划批次（批次级二次确认）
func (l *ConfirmPlanBatchLogic) ConfirmPlanBatch(in *trigger.ConfirmPlanBatchReq) (*trigger.ConfirmPlanBatchRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if strutil.IsBlank(in.Id) && strutil.IsBlank(in.BatchId) {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_PARAM, "参数错误")
	}

	db := l.svcCtx.DB.WithContext(l.ctx).DB
	var batch gormmodel.PlanBatch
	var err error
	if !strutil.IsBlank(in.Id) {
		err = db.Where("id = ?", in.Id).First(&batch).Error
	} else {
		err = db.Where("batch_id = ?", in.BatchId).First(&batch).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errx.NewErrorByPbCode(extproto.Code__1_00_RECORD_NOT_EXIST)
		}
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询计划批次失败")
	}

	release, locked, lockErr := confirm.AcquireLock(l.ctx, l.svcCtx.Redis, batch.Id)
	if lockErr != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_CACHE, lockErr, "获取批次确认锁失败")
	}
	if !locked {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "批次确认处理中，请稍后重试")
	}
	defer release()
	// 锁内重新加载，避免使用锁前读到的旧状态。
	if err := db.Where("id = ?", batch.Id).First(&batch).Error; err != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询计划批次失败")
	}

	var plan gormmodel.Plan
	if err := db.Where("id = ?", batch.PlanPk).First(&plan).Error; err != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询计划失败")
	}
	if plan.Status == model.PlanStatusTerminated || plan.FinishedTime.Valid {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划状态已结束,不可确认")
	}
	if batch.Status == model.PlanStatusTerminated || batch.FinishedTime.Valid {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "计划批次状态已结束,不可确认")
	}

	record, err := gormmodel.FindPlanConfirmByTarget(l.ctx, db, model.ConfirmTypeBatch, batch.Id)
	if err != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "查询批次确认记录失败")
	}
	if record == nil {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "该批次未配置二次确认")
	}
	// 重复确认按幂等成功处理。
	if record.ConfirmStatus == model.PlanConfirmStatusConfirmed {
		return &trigger.ConfirmPlanBatchRes{}, nil
	}
	now := time.Now()
	if now.After(record.Deadline) {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "批次确认已超过截止时间")
	}

	rows, err := gormmodel.ConfirmPlanTarget(l.ctx, db, model.ConfirmTypeBatch, batch.Id, authctx.GetUserId(l.ctx), now)
	if err != nil {
		return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, err, "确认计划批次失败")
	}
	if rows == 0 {
		// 并发确认或恰好被超时扫描终止，以数据库最终状态为准。
		latest, findErr := gormmodel.FindPlanConfirmByTarget(l.ctx, db, model.ConfirmTypeBatch, batch.Id)
		if findErr != nil {
			return nil, errx.NewErrorByPbCodeWrap(extproto.Code__1_00_DB, findErr, "查询批次确认记录失败")
		}
		if latest != nil && latest.ConfirmStatus == model.PlanConfirmStatusConfirmed {
			return &trigger.ConfirmPlanBatchRes{}, nil
		}
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_BIZ_STATE, "批次确认状态已变化，请刷新后重试")
	}

	planscope.BatchScope(&plan, &batch).WithFields(
		logx.Field("confirm_deadline", record.Deadline.Format(time.RFC3339)),
	).Logger(l.ctx).Info("RPC 确认计划批次：批次已放行，等待定时扫表下发执行项")
	return &trigger.ConfirmPlanBatchRes{}, nil
}
