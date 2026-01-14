package logic

import (
	"context"
	"time"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/model"

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
	pageSize := in.PageSize
	if pageSize <= 0 {
		pageSize = 10 // 默认分页大小
	}
	pageNum := in.PageNum
	if pageNum <= 0 {
		pageNum = 1 // 默认页码
	}
	builder := l.svcCtx.PlanExecItemModel.SelectBuilder().
		Where("del_state = ?", 0)
	// 添加过滤条件
	if in.PlanId != "" {
		builder = builder.Where("plan_id = ?", in.PlanId)
	}
	if in.BatchId != "" {
		builder = builder.Where("batch_id = ?", in.BatchId)
	}
	if in.StartTime != "" {
		builder = builder.Where("plan_trigger_time >= ?", in.StartTime)
	}
	if in.EndTime != "" {
		builder = builder.Where("plan_trigger_time <= ?", in.EndTime)
	}
	// 使用新添加的分组统计接口获取数据
	execItems, err := l.svcCtx.PlanExecItemModel.FindGroupedStats(l.ctx, builder)
	if err != nil {
		return nil, err
	}

	// 按 plan_id, batch_id, plan_trigger_time 分组
	type groupKey struct {
		planId          string
		batchId         string
		planTriggerTime time.Time
	}
	type statGroup struct {
		total        int64
		statusCounts map[int64]int64
	}
	groupMap := make(map[groupKey]*statGroup)
	for _, item := range execItems {
		key := groupKey{
			planId:          item.PlanId,
			batchId:         item.BatchId,
			planTriggerTime: item.PlanTriggerTime,
		}
		group, exists := groupMap[key]
		if !exists {
			group = &statGroup{
				total:        0,
				statusCounts: make(map[int64]int64),
			}
			groupMap[key] = group
		}
		// 更新总数量
		group.total = item.TriggerCount
		// 更新状态数量
		group.statusCounts[item.Status] = item.TriggerCount
	}

	// 使用新添加的分组总数接口获取总数
	total, err := l.svcCtx.PlanExecItemModel.FindGroupedCount(l.ctx, builder)
	if err != nil {
		return nil, err
	}

	stats := make([]*trigger.PbPlanExecItemStat, 0, len(groupMap))
	for key, group := range groupMap {
		stat := &trigger.PbPlanExecItemStat{
			PlanId:          key.planId,
			BatchId:         key.batchId,
			PlanTriggerTime: key.planTriggerTime.Format("2006-01-02 15:04:05"),
			Total:           group.total,
			StatusStats:     make([]*trigger.PbExecItemStatusCount, 0),
		}
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
			// 转换为proto响应格式
			stat.StatusStats = append(stat.StatusStats, &trigger.PbExecItemStatusCount{
				Status:     trigger.PbExecItemStatus(status),
				StatusName: model.StatusNames[status],
				Count:      count,
			})
		}
		stats = append(stats, stat)
	}

	start := (pageNum - 1) * pageSize
	end := start + pageSize
	if start >= total {
		return &trigger.ListPlanExecItemStatsRes{
			Stats: make([]*trigger.PbPlanExecItemStat, 0),
			Total: total,
		}, nil
	}
	if end > total {
		end = total
	}
	resp := &trigger.ListPlanExecItemStatsRes{
		Stats: stats[start:end],
		Total: total,
	}
	return resp, nil
}
