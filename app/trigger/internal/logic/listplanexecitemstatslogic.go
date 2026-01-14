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

	// 定义分组映射结构
	type statusCountMap map[int64]int64
	type statGroup struct {
		planId          string
		batchId         string
		planTriggerTime string
		total           int64
		statusCounts    statusCountMap
	}

	// 按 plan_id, batch_id, plan_trigger_time 分组统计
	statMap := make(map[string]*statGroup)
	for _, item := range execItems {
		// 构建分组键
		triggerTimeStr := item.PlanTriggerTime.Format("2006-01-02 15:04:05")
		key := fmt.Sprintf("%s_%s_%s", item.PlanId, item.BatchId, triggerTimeStr)
		group, exists := statMap[key]
		if !exists {
			group = &statGroup{
				planId:          item.PlanId,
				batchId:         item.BatchId,
				planTriggerTime: triggerTimeStr,
				total:           0,
				statusCounts:    make(statusCountMap),
			}
			statMap[key] = group
		}

		// 统计总数量
		group.total++

		// 统计各状态数量
		group.statusCounts[item.Status]++
	}

	// 转换为响应格式
	stats := make([]*trigger.PbPlanExecItemStat, 0, len(statMap))
	for _, group := range statMap {
		// 创建统计对象
		stat := &trigger.PbPlanExecItemStat{
			PlanId:          group.planId,
			BatchId:         group.batchId,
			PlanTriggerTime: group.planTriggerTime,
			Total:           group.total,
			StatusStats:     make([]*trigger.PbExecItemStatusCount, 0, len(group.statusCounts)),
		}

		// 遍历所有可能的状态，确保每个状态都有统计
		allStatuses := []int64{
			int64(model.StatusWaiting),
			int64(model.StatusDelayed),
			int64(model.StatusRunning),
			int64(model.StatusPaused),
			int64(model.StatusCompleted),
			int64(model.StatusTerminated),
		}

		for _, status := range allStatuses {
			count := group.statusCounts[status]
			if count > 0 {
				// 转换为proto响应格式
				stat.StatusStats = append(stat.StatusStats, &trigger.PbExecItemStatusCount{
					Status:     trigger.PbExecItemStatus(status),
					StatusName: statusNames[status],
					Count:      count,
				})
			}
		}

		stats = append(stats, stat)
	}

	// 计算分页参数
	pageSize := in.PageSize
	if pageSize <= 0 {
		pageSize = 10 // 默认分页大小
	}

	pageNum := in.PageNum
	if pageNum <= 0 {
		pageNum = 1 // 默认页码
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
