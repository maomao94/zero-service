package logic

import (
	"context"

	ispclient "zero-service/app/ispagent/internal/isp"
	"zero-service/app/ispagent/internal/svc"
	"zero-service/app/ispagent/ispagent"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListReportIntervalsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListReportIntervalsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListReportIntervalsLogic {
	return &ListReportIntervalsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListReportIntervalsLogic) ListReportIntervals(in *ispagent.ListReportIntervalsReq) (*ispagent.ListReportIntervalsRes, error) {
	reportIntervals := l.svcCtx.IspClient.ReportIntervals()
	reserved := l.svcCtx.IspClient.ReservedIntervals()

	intervals := make([]*ispagent.ReportIntervalEntry, 0, len(reportIntervals))
	for cat, d := range reportIntervals {
		intervals = append(intervals, &ispagent.ReportIntervalEntry{
			Category:        int32(cat),
			IntervalSeconds: int64(d.Seconds()),
			Name:            ispclient.CategoryMessageName(cat),
		})
	}

	reservedEntries := make([]*ispagent.ReservedIntervalEntry, 0, len(reserved))
	for k, d := range reserved {
		reservedEntries = append(reservedEntries, &ispagent.ReservedIntervalEntry{
			Key:             k,
			IntervalSeconds: int64(d.Seconds()),
		})
	}

	return &ispagent.ListReportIntervalsRes{
		Intervals: intervals,
		Reserved:  reservedEntries,
	}, nil
}
