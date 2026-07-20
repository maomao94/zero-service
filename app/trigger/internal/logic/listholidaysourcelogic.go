package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListHolidaySourceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListHolidaySourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListHolidaySourceLogic {
	return &ListHolidaySourceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取中国大陆节假日源配置
func (l *ListHolidaySourceLogic) ListHolidaySource(in *trigger.ListHolidaySourceReq) (*trigger.ListHolidaySourceRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	items, err := l.svcCtx.HolidaySource.List(l.ctx, int(in.Year), in.IncludeDisabled)
	if err != nil {
		return nil, err
	}
	res := &trigger.ListHolidaySourceRes{Items: make([]*trigger.HolidaySourcePb, 0, len(items))}
	for _, item := range items {
		res.Items = append(res.Items, toHolidaySourcePb(item))
	}

	return res, nil
}
