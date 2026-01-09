package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
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
	log, err := l.svcCtx.PlanExecLogModel.FindOne(l.ctx, in.Id)
	if err != nil {
		if err == sqlx.ErrNotFound {
			return nil, err
		}
		return nil, err
	}

	// 构建响应
	pbLog := &trigger.PbPlanExecLog{
		LogId:       log.Id,
		PlanId:      log.PlanId,
		PlanName:    log.PlanName,
		ItemId:      log.ItemId,
		ItemName:    log.ItemName,
		TriggerTime: carbon.CreateFromStdTime(log.TriggerTime).ToDateTimeString(),
		ExecResult:  int32(log.ExecResult),
		Message:     log.Message,
	}

	return &trigger.GetPlanExecLogRes{
		PlanExecLog: pbLog,
	}, nil
}
