package logic

import (
	"context"
	"errors"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type GetPlanExecLogLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetPlanExecLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPlanExecLogLogic {
	return &GetPlanExecLogLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取计划执行日志详情
func (l *GetPlanExecLogLogic) GetPlanExecLog(in *trigger.GetPlanExecLogReq) (*trigger.GetPlanExecLogRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 查询日志
	var execLog gormmodel.PlanExecLog
	err = l.svcCtx.DB.WithContext(l.ctx).Where("id = ?", in.Id).First(&execLog).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
		}
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_DB, "查询执行日志失败")
	}

	// 构建响应
	pbLog := &trigger.PlanExecLogPb{
		CreateTime:  carbon.CreateFromStdTime(execLog.CreateTime).ToDateTimeString(),
		UpdateTime:  carbon.CreateFromStdTime(execLog.UpdateTime).ToDateTimeString(),
		CreateUser:  execLog.CreateUser.String,
		UpdateUser:  execLog.UpdateUser.String,
		DeptCode:    execLog.DeptCode.String,
		Id:          execLog.Id,
		PlanPk:      execLog.PlanPk,
		PlanId:      execLog.PlanId,
		PlanName:    execLog.PlanName.String,
		BatchPk:     execLog.BatchPk,
		BatchId:     execLog.BatchId,
		ItemPk:      execLog.ItemPk,
		ExecId:      execLog.ExecId,
		ItemId:      execLog.ItemId,
		ItemType:    execLog.ItemType.String,
		ItemName:    execLog.ItemName.String,
		PointId:     execLog.PointId.String,
		TriggerTime: carbon.CreateFromStdTime(execLog.TriggerTime).ToDateTimeString(),
		ExecResult:  execLog.ExecResult.String,
		Message:     execLog.Message.String,
	}

	return &trigger.GetPlanExecLogRes{
		PlanExecLog: pbLog,
	}, nil
}
