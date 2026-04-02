package logic

import (
	"context"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/copierx"

	"github.com/jinzhu/copier"

	"github.com/zeromicro/go-zero/core/logx"
)

type HistoricalStatsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewHistoricalStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *HistoricalStatsLogic {
	return &HistoricalStatsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取任务历史统计
func (l *HistoricalStatsLogic) HistoricalStats(in *trigger.HistoricalStatsReq) (*trigger.HistoricalStatsRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	dailyStats, err := l.svcCtx.AsynqInspector.History(in.Queue, int(in.N))
	if err != nil {
		return nil, err
	}
	dailyStatList := []*trigger.DailyStatsPb{}
	copier.CopyWithOption(&dailyStatList, dailyStats, copierx.Option)
	return &trigger.HistoricalStatsRes{
		DailyStat: dailyStatList,
	}, nil
}
