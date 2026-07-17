package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetExecItemDashboardLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetExecItemDashboardLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetExecItemDashboardLogic {
	return &GetExecItemDashboardLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取执行项仪表板统计信息
func (l *GetExecItemDashboardLogic) GetExecItemDashboard(in *trigger.GetExecItemDashboardReq) (*trigger.GetExecItemDashboardRes, error) {
	db := l.svcCtx.DB.WithContext(l.ctx).Table("plan_exec_item AS pei").
		Joins("JOIN plan_batch pb ON pei.batch_pk = pb.id").
		Joins("JOIN plan p ON pb.plan_pk = p.id").
		Select(`p.type AS planType,
			COUNT(pei.id) AS total,
			COUNT(CASE WHEN (pei.status IN (200, 300) OR p.finished_time IS NOT NULL OR pb.finished_time IS NOT NULL) THEN 1 END) AS finishedTotal,
			COUNT(CASE WHEN (pei.status = 300 OR p.status = 3 OR pb.status = 3) THEN 1 END) AS finishedTerminated,
			COUNT(CASE WHEN (pei.status NOT IN (200, 300) AND p.finished_time IS NULL AND pb.finished_time IS NULL) THEN 1 END) AS pendingTotal,
			COUNT(CASE WHEN (pei.status = 10 AND p.finished_time IS NULL AND pb.finished_time IS NULL) THEN 1 END) AS pendingDelayed`).
		Where("pei.is_deleted = 0 AND pb.is_deleted = 0 AND p.is_deleted = 0")

	if in.DeptCode != "" {
		db = db.Where("p.dept_code = ?", in.DeptCode)
	}
	if in.UserId != "" {
		db = db.Where("p.create_user = ?", in.UserId)
	}
	if in.PlanType != "" {
		db = db.Where("p.type = ?", in.PlanType)
	}

	db = db.Group("p.type").Order("p.type ASC")

	type DashboardStats struct {
		PlanType           string `gorm:"column:planType"`
		Total              int64  `gorm:"column:total"`
		FinishedTotal      int64  `gorm:"column:finishedTotal"`
		FinishedTerminated int64  `gorm:"column:finishedTerminated"`
		PendingTotal       int64  `gorm:"column:pendingTotal"`
		PendingDelayed     int64  `gorm:"column:pendingDelayed"`
	}

	var stats []DashboardStats
	if err := db.Scan(&stats).Error; err != nil {
		return nil, err
	}

	response := &trigger.GetExecItemDashboardRes{
		Stats: make([]*trigger.ExecItemDashboardItemPb, 0, len(stats)),
	}
	for _, stat := range stats {
		response.Stats = append(response.Stats, &trigger.ExecItemDashboardItemPb{
			PlanType: stat.PlanType,
			Total:    stat.Total,
			Finished: &trigger.FinishedItemsStatsPb{
				Total:      stat.FinishedTotal,
				Terminated: stat.FinishedTerminated,
			},
			Pending: &trigger.PendingItemsStatsPb{
				Total:   stat.PendingTotal,
				Delayed: stat.PendingDelayed,
			},
		})
	}
	return response, nil
}
