package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/logx"
)

type QueryHolidayLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewQueryHolidayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryHolidayLogic {
	return &QueryHolidayLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询中国大陆日期类型
func (l *QueryHolidayLogic) QueryHoliday(in *trigger.QueryHolidayReq) (*trigger.QueryHolidayRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	date, err := parseHolidayDate(in.Date)
	if err != nil {
		return nil, err
	}

	return &trigger.QueryHolidayRes{Day: toHolidayDayPb(l.svcCtx.HolidayCalendar.Lookup(date))}, nil
}
