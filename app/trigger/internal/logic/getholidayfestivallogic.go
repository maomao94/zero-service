package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetHolidayFestivalLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetHolidayFestivalLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetHolidayFestivalLogic {
	return &GetHolidayFestivalLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取中国大陆节日详情
func (l *GetHolidayFestivalLogic) GetHolidayFestival(in *trigger.GetHolidayFestivalReq) (*trigger.GetHolidayFestivalRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	info, ok := l.svcCtx.HolidayCalendar.GetFestival(int(in.Year), in.Name)
	if !ok {
		return &trigger.GetHolidayFestivalRes{}, nil
	}

	return &trigger.GetHolidayFestivalRes{Found: true, Festival: toHolidayFestivalPb(info)}, nil
}
