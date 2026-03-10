package logic

import (
	"context"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/doug-martin/goqu/v9"
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
	ds := l.svcCtx.Database.From(goqu.T("plan_exec_item").As("pei")).
		Join(goqu.T("plan_batch").As("pb"), goqu.On(goqu.I("pei.batch_pk").Eq(goqu.I("pb.id")))).
		Join(goqu.T("plan").As("p"), goqu.On(goqu.I("pb.plan_pk").Eq(goqu.I("p.id")))).
		Select(
			goqu.I("p.type").As("planType"),
			goqu.COUNT(goqu.I("pei.id")).As("total"),
			// 已结束任务数
			goqu.COUNT(
				goqu.Case().
					When(
						goqu.Or(
							goqu.I("pei.status").In([]int{200, 300}),
							goqu.I("p.finished_time").IsNotNull(),
							goqu.I("pb.finished_time").IsNotNull(),
						),
						1,
					),
			).As("finishedTotal"),
			// 已终止任务数
			goqu.COUNT(
				goqu.Case().
					When(
						goqu.Or(
							goqu.I("pei.status").Eq(300),
							goqu.I("p.status").Eq(3),
							goqu.I("pb.status").Eq(3),
						),
						1,
					),
			).As("finishedTerminated"),
			// 待完成任务数
			goqu.COUNT(
				goqu.Case().
					When(
						goqu.And(
							goqu.I("pei.status").NotIn([]int{200, 300}),
							goqu.I("p.finished_time").IsNull(),
							goqu.I("pb.finished_time").IsNull(),
						),
						1,
					),
			).As("pendingTotal"),
			// 延期任务数
			goqu.COUNT(
				goqu.Case().
					When(
						goqu.And(
							goqu.I("pei.status").Eq(10),
							goqu.I("p.finished_time").IsNull(),
							goqu.I("pb.finished_time").IsNull(),
						),
						1,
					),
			).As("pendingDelayed"),
		).
		Where(
			goqu.And(
				goqu.I("pei.del_state").Eq(0),
				goqu.I("pb.del_state").Eq(0),
				goqu.I("p.del_state").Eq(0),
			),
		)

	// 可选过滤条件
	if in.DeptCode != "" {
		ds = ds.Where(goqu.I("p.dept_code").Eq(in.DeptCode))
	}
	if in.UserId != "" {
		ds = ds.Where(goqu.I("p.create_user").Eq(in.UserId))
	}
	if in.PlanType != "" {
		ds = ds.Where(goqu.I("p.type").Eq(in.PlanType))
	}
	// 分组和排序
	ds = ds.GroupBy(goqu.I("p.type")).Order(goqu.I("p.type").Asc())

	// 执行查询
	type DashboardStats struct {
		PlanType           string `db:"planType"`
		Total              int64  `db:"total"`
		FinishedTotal      int64  `db:"finishedTotal"`
		FinishedTerminated int64  `db:"finishedTerminated"`
		PendingTotal       int64  `db:"pendingTotal"`
		PendingDelayed     int64  `db:"pendingDelayed"`
	}

	var stats []DashboardStats
	sql, args, err := ds.ToSQL()
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.SqlConn.QueryRowsPartialCtx(l.ctx, &stats, sql, args...); err != nil {
		return nil, err
	}

	// 构建响应
	response := &trigger.GetExecItemDashboardRes{
		Stats: make([]*trigger.ExecItemDashboardItem, 0, len(stats)),
	}
	for _, stat := range stats {
		response.Stats = append(response.Stats, &trigger.ExecItemDashboardItem{
			PlanType: stat.PlanType,
			Total:    stat.Total,
			Finished: &trigger.FinishedItemsStats{
				Total:      stat.FinishedTotal,
				Terminated: stat.FinishedTerminated,
			},
			Pending: &trigger.PendingItemsStats{
				Total:   stat.PendingTotal,
				Delayed: stat.PendingDelayed,
			},
		})
	}
	return response, nil
}
