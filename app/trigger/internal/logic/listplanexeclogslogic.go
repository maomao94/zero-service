package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListPlanExecLogsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListPlanExecLogsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPlanExecLogsLogic {
	return &ListPlanExecLogsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 分页获取执行日志列表
func (l *ListPlanExecLogsLogic) ListPlanExecLogs(in *trigger.ListPlanExecLogsReq) (*trigger.ListPlanExecLogsRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 构建查询条件
	builder := l.svcCtx.PlanExecLogModel.SelectBuilder()
	if in.PlanId != "" {
		builder = builder.Where("plan_id = ?", in.PlanId)
	}
	if in.ItemId != "" {
		builder = builder.Where("item_id = ?", in.ItemId)
	}
	if in.StartTime != "" {
		builder = builder.Where("trigger_time >= ?", in.StartTime)
	}
	if in.EndTime != "" {
		builder = builder.Where("trigger_time <= ?", in.EndTime)
	}
	if len(in.ExecResult) > 0 {
		execResultInts := make([]int64, len(in.ExecResult))
		for i, execResult := range in.ExecResult {
			execResultInts[i] = int64(execResult)
		}
		builder = builder.Where("exec_result IN (?) ", execResultInts)
	}

	// 查询日志列表
	logs, total, err := l.svcCtx.PlanExecLogModel.FindPageListByPageWithTotal(l.ctx, builder, in.PageNum, in.PageSize, "id DESC")
	if err != nil {
		return nil, err
	}

	// 构建响应
	resp := &trigger.ListPlanExecLogsRes{
		PlanExecLogs: make([]*trigger.PbPlanExecLog, 0, len(logs)),
		Total:        total,
	}

	// 转换日志列表
	for _, log := range logs {
		pbLog := &trigger.PbPlanExecLog{
			CreateTime:  carbon.CreateFromStdTime(log.CreateTime).ToDateTimeString(),
			UpdateTime:  carbon.CreateFromStdTime(log.UpdateTime).ToDateTimeString(),
			CreateUser:  log.CreateUser.String,
			UpdateUser:  log.UpdateUser.String,
			Id:          log.Id,
			PlanPk:      log.PlanPk,
			PlanId:      log.PlanId,
			PlanName:    log.PlanName.String,
			ItemPk:      log.ItemPk,
			ItemId:      log.ItemId,
			ItemName:    log.ItemName.String,
			PointId:     log.PointId.String,
			TriggerTime: carbon.CreateFromStdTime(log.TriggerTime).ToDateTimeString(),
			ExecResult:  log.ExecResult.String,
			Message:     log.Message.String,
		}

		resp.PlanExecLogs = append(resp.PlanExecLogs, pbLog)
	}

	return resp, nil
}
