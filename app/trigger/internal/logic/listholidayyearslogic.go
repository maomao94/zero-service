package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListHolidayYearsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListHolidayYearsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListHolidayYearsLogic {
	return &ListHolidayYearsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取已配置中国大陆节假日年份
func (l *ListHolidayYearsLogic) ListHolidayYears(in *trigger.ListHolidayYearsReq) (*trigger.ListHolidayYearsRes, error) {
	years := l.svcCtx.HolidayCalendar.SupportedYears()
	res := &trigger.ListHolidayYearsRes{Years: make([]int32, 0, len(years))}
	for _, year := range years {
		res.Years = append(res.Years, int32(year))
	}

	return res, nil
}
