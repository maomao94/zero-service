package logic

import (
	"context"
	"time"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/holiday"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListHolidayFestivalsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListHolidayFestivalsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListHolidayFestivalsLogic {
	return &ListHolidayFestivalsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询中国大陆节日列表
func (l *ListHolidayFestivalsLogic) ListHolidayFestivals(in *trigger.ListHolidayFestivalsReq) (*trigger.ListHolidayFestivalsRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	startYear, endYear := int(in.StartYear), int(in.EndYear)
	if startYear == 0 {
		startYear = time.Now().Year()
	}
	if endYear == 0 {
		endYear = startYear
	}
	items, err := l.svcCtx.HolidayCalendar.ListFestivals(l.ctx, holiday.ListFestivalsReq{StartYear: startYear, EndYear: endYear, Name: in.Name})
	if err != nil {
		return nil, err
	}
	res := &trigger.ListHolidayFestivalsRes{Items: make([]*trigger.HolidayFestivalPb, 0, len(items))}
	for _, item := range items {
		res.Items = append(res.Items, toHolidayFestivalPb(item))
	}

	return res, nil
}
