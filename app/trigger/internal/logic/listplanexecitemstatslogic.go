package logic

import (
	"context"
	"fmt"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/model"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListPlanExecItemStatsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListPlanExecItemStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPlanExecItemStatsLogic {
	return &ListPlanExecItemStatsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 分页获取计划执行项统计
func (l *ListPlanExecItemStatsLogic) ListPlanExecItemStats(in *trigger.ListPlanExecItemStatsReq) (*trigger.ListPlanExecItemStatsRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 构建查询条件
	builder := l.svcCtx.PlanExecItemModel.SelectBuilder().
		Select("plan_id", "batch_id", "plan_trigger_time", "status").
		Where("del_state = ?", 0)

	// 添加过滤条件
	if in.PlanId != "" {
		builder = builder.Where("plan_id = ?", in.PlanId)
	}

	if in.StartTime != "" {
		builder = builder.Where("plan_trigger_time >= ?", in.StartTime)
	}

	if in.EndTime != "" {
		builder = builder.Where("plan_trigger_time <= ?", in.EndTime)
	}

	// 执行查询获取所有符合条件的数据
	execItems, err := l.svcCtx.PlanExecItemModel.FindAll(l.ctx, builder, "plan_trigger_time DESC")
	if err != nil {
		return nil, err
	}

	// 按 plan_id, batch_id, plan_trigger_time 分组统计
	statMap := make(map[string]*trigger.PbPlanExecItemStat)
	for _, item := range execItems {
		// 构建分组键
		key := fmt.Sprintf("%s_%s_%s", item.PlanId, item.BatchId, item.PlanTriggerTime.Format("2006-01-02 15:04:05"))
		stat, exists := statMap[key]
		if !exists {
			stat = &trigger.PbPlanExecItemStat{
				PlanId:          item.PlanId,
				BatchId:         item.BatchId,
				PlanTriggerTime: item.PlanTriggerTime.Format("2006-01-02 15:04:05"),
				Total:           0,
				Success:         0,
				Failed:          0,
				Running:         0,
			}
			statMap[key] = stat
		}

		// 统计总数量
		stat.Total++

		// 根据状态统计
		switch item.Status {
		case int64(model.StatusCompleted):
			stat.Success++
		case int64(model.StatusTerminated):
			stat.Failed++
		case int64(model.StatusRunning):
			stat.Running++
		}
	}

	// 转换为切片
	stats := make([]*trigger.PbPlanExecItemStat, 0, len(statMap))
	for _, stat := range statMap {
		stats = append(stats, stat)
	}

	// 计算分页参数
	pageSize := in.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	pageNum := in.PageNum
	if pageNum <= 0 {
		pageNum = 1
	}

	// 执行分页
	start := (pageNum - 1) * pageSize
	end := start + pageSize
	total := int64(len(stats))

	if start >= total {
		return &trigger.ListPlanExecItemStatsRes{
			Stats: make([]*trigger.PbPlanExecItemStat, 0),
			Total: total,
		}, nil
	}

	if end > total {
		end = total
	}

	// 构建响应
	resp := &trigger.ListPlanExecItemStatsRes{
		Stats: stats[start:end],
		Total: total,
	}

	return resp, nil
}
