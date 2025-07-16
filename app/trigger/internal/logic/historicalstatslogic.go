package logic

import (
	"context"
	"github.com/jinzhu/copier"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/copierx"

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
	dailyStatList := []*trigger.PbDailyStats{}
	copier.CopyWithOption(&dailyStatList, dailyStats, copierx.Option)
	return &trigger.HistoricalStatsRes{
		DailyStat: dailyStatList,
	}, nil
}
