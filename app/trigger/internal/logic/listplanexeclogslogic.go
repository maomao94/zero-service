package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/gormx"

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
	db := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.PlanExecLog{})
	if in.PlanId != "" {
		db = db.Where("plan_id = ?", in.PlanId)
	}
	if in.ItemId != "" {
		db = db.Where("item_id = ?", in.ItemId)
	}
	if in.ExecId != "" {
		db = db.Where("exec_id = ?", in.ExecId)
	}
	if in.StartTime != "" {
		db = db.Where("trigger_time >= ?", in.StartTime)
	}
	if in.EndTime != "" {
		db = db.Where("trigger_time <= ?", in.EndTime)
	}
	if len(in.ExecResult) > 0 {
		execResultInts := make([]int64, len(in.ExecResult))
		for i, execResult := range in.ExecResult {
			execResultInts[i] = int64(execResult)
		}
		db = db.Where("exec_result IN ?", execResultInts)
	}

	var logs []gormmodel.PlanExecLog
	page, err := gormx.QueryPage(db.Order("create_time DESC, id DESC"), int(in.PageNum), int(in.PageSize), &logs)
	if err != nil {
		return nil, err
	}

	// 构建响应
	resp := &trigger.ListPlanExecLogsRes{
		PlanExecLogs: make([]*trigger.PlanExecLogPb, 0, len(logs)),
		Total:        page.Total,
	}

	// 转换日志列表
	for i := range logs {
		pbLog := &trigger.PlanExecLogPb{
			CreateTime:  carbon.CreateFromStdTime(logs[i].CreateTime).ToDateTimeString(),
			UpdateTime:  carbon.CreateFromStdTime(logs[i].UpdateTime).ToDateTimeString(),
			CreateUser:  logs[i].CreateUser.String,
			UpdateUser:  logs[i].UpdateUser.String,
			DeptCode:    logs[i].DeptCode.String,
			Id:          logs[i].Id,
			PlanPk:      logs[i].PlanPk,
			PlanId:      logs[i].PlanId,
			PlanName:    logs[i].PlanName.String,
			BatchPk:     logs[i].BatchPk,
			BatchId:     logs[i].BatchId,
			ItemPk:      logs[i].ItemPk,
			ExecId:      logs[i].ExecId,
			ItemId:      logs[i].ItemId,
			ItemType:    logs[i].ItemType.String,
			ItemName:    logs[i].ItemName.String,
			PointId:     logs[i].PointId.String,
			TriggerTime: carbon.CreateFromStdTime(logs[i].TriggerTime).ToDateTimeString(),
			ExecResult:  logs[i].ExecResult.String,
			Message:     logs[i].Message.String,
		}

		resp.PlanExecLogs = append(resp.PlanExecLogs, pbLog)
	}

	return resp, nil
}
