package logic

import (
	"context"
	"zero-service/model"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/dromara/carbon/v2"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/songzhibin97/gkit/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetPlanExecItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetPlanExecItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPlanExecItemLogic {
	return &GetPlanExecItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取执行项详情
func (l *GetPlanExecItemLogic) GetPlanExecItem(in *trigger.GetPlanExecItemReq) (*trigger.GetPlanExecItemRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	if in.Id <= 0 && strutil.IsBlank(in.ExecId) {
		return nil, errors.BadRequest("", "参数错误")
	}
	var execItem *model.PlanExecItem
	if in.Id > 0 {
		execItem, err = l.svcCtx.PlanExecItemModel.FindOne(l.ctx, in.Id)
	} else {
		execItem, err = l.svcCtx.PlanExecItemModel.FindOneByExecId(l.ctx, in.ExecId)
	}
	if err != nil {
		return nil, err
	}

	// 构建响应
	pbExecItem := &trigger.PbPlanExecItem{
		CreateTime:       carbon.CreateFromStdTime(execItem.CreateTime).ToDateTimeString(),
		UpdateTime:       carbon.CreateFromStdTime(execItem.UpdateTime).ToDateTimeString(),
		CreateUser:       execItem.CreateUser.String,
		UpdateUser:       execItem.UpdateUser.String,
		DeptCode:         execItem.DeptCode.String,
		Id:               execItem.Id,
		PlanPk:           execItem.PlanPk,
		PlanId:           execItem.PlanId,
		BatchPk:          execItem.BatchPk,
		BatchId:          execItem.BatchId,
		ExecId:           execItem.ExecId,
		ItemId:           execItem.ItemId,
		ItemType:         execItem.ItemType.String,
		ItemName:         execItem.ItemName.String,
		PointId:          execItem.PointId.String,
		ServiceAddr:      execItem.ServiceAddr,
		Payload:          execItem.Payload,
		RequestTimeout:   execItem.RequestTimeout,
		PlanTriggerTime:  carbon.CreateFromStdTime(execItem.PlanTriggerTime).ToDateTimeString(),
		NextTriggerTime:  carbon.CreateFromStdTime(execItem.NextTriggerTime).ToDateTimeString(),
		TriggerCount:     int32(execItem.TriggerCount),
		Status:           int32(execItem.Status),
		LastResult:       execItem.LastResult.String,
		LastMessage:      execItem.LastMessage.String,
		LastReason:       execItem.LastReason.String,
		TerminatedReason: execItem.TerminatedReason.String,
		PausedReason:     execItem.PausedReason.String,
		Ext1:             execItem.Ext1.String,
		Ext2:             execItem.Ext2.String,
		Ext3:             execItem.Ext3.String,
		Ext4:             execItem.Ext4.String,
		Ext5:             execItem.Ext5.String,
	}

	// 设置上次触发时间
	if execItem.LastTriggerTime.Valid {
		pbExecItem.LastTriggerTime = carbon.CreateFromStdTime(execItem.LastTriggerTime.Time).ToDateTimeString()
	}

	// 设置终止时间和原因
	if execItem.TerminatedTime.Valid {
		pbExecItem.TerminatedTime = carbon.CreateFromStdTime(execItem.TerminatedTime.Time).ToDateTimeString()
	}

	// 设置暂停时间和原因
	if execItem.PausedTime.Valid {
		pbExecItem.PausedTime = carbon.CreateFromStdTime(execItem.PausedTime.Time).ToDateTimeString()
	}

	return &trigger.GetPlanExecItemRes{
		PlanExecItem: []*trigger.PbPlanExecItem{pbExecItem},
	}, nil
}
