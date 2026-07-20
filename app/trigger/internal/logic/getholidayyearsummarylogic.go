package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetHolidayYearSummaryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetHolidayYearSummaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetHolidayYearSummaryLogic {
	return &GetHolidayYearSummaryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取中国大陆全年节假日汇总
func (l *GetHolidayYearSummaryLogic) GetHolidayYearSummary(in *trigger.GetHolidayYearSummaryReq) (*trigger.GetHolidayYearSummaryRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	info, ok := l.svcCtx.HolidayCalendar.GetYearSummary(int(in.Year))
	if !ok {
		return &trigger.GetHolidayYearSummaryRes{}, nil
	}

	return &trigger.GetHolidayYearSummaryRes{Found: true, Summary: toHolidayYearSummaryPb(info)}, nil
}
