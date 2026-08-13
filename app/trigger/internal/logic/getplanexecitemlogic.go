package logic

import (
	"context"
	"errors"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/common/carbonx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
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
	if strutil.IsBlank(in.Id) && strutil.IsBlank(in.ExecId) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "参数错误")
	}
	var execItem gormmodel.PlanExecItem
	if !strutil.IsBlank(in.Id) {
		err = l.svcCtx.DB.WithContext(l.ctx).Where("id = ?", in.Id).First(&execItem).Error
	} else {
		err = l.svcCtx.DB.WithContext(l.ctx).Where("exec_id = ?", in.ExecId).First(&execItem).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
		}
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_DB, "查询执行项失败")
	}

	// 构建响应
	pbExecItem := &trigger.PlanExecItemPb{
		CreateTime:       carbonx.FormatDateTime(execItem.CreateTime),
		UpdateTime:       carbonx.FormatDateTime(execItem.UpdateTime),
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
		ItemRowId:        execItem.ItemRowId,
		PointId:          execItem.PointId.String,
		Payload:          execItem.Payload,
		RequestTimeout:   execItem.RequestTimeout,
		PlanTriggerTime:  carbonx.FormatDateTime(execItem.PlanTriggerTime),
		NextTriggerTime:  carbonx.FormatDateTime(execItem.NextTriggerTime),
		TriggerCount:     int32(execItem.TriggerCount),
		Status:           trigger.ExecItemStatusPb(execItem.Status),
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
		pbExecItem.LastTriggerTime = carbonx.FormatDateTime(execItem.LastTriggerTime.Time)
	}

	// 设置暂停时间和原因
	if execItem.PausedTime.Valid {
		pbExecItem.PausedTime = carbonx.FormatDateTime(execItem.PausedTime.Time)
	}
	return &trigger.GetPlanExecItemRes{
		PlanExecItem: []*trigger.PlanExecItemPb{pbExecItem},
	}, nil
}
