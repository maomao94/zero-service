package logic

import (
	"context"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/holiday"

	"github.com/zeromicro/go-zero/core/logx"
)

type SaveHolidaySourceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSaveHolidaySourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SaveHolidaySourceLogic {
	return &SaveHolidaySourceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 保存中国大陆节假日源配置
func (l *SaveHolidaySourceLogic) SaveHolidaySource(in *trigger.SaveHolidaySourceReq) (*trigger.SaveHolidaySourceRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if err := validateHolidayDate(in.Date); err != nil {
		return nil, err
	}
	if err := l.svcCtx.HolidaySource.Save(l.ctx, holiday.StoredEntry{Date: in.Date, Entry: holiday.Entry{Name: in.Name, Type: holiday.DayType(in.Type), IsFestivalDay: in.IsFestivalDay, Note: in.Note}, Enabled: in.Enabled}); err != nil {
		return nil, err
	}
	if err := l.svcCtx.HolidayCalendar.Reload(l.ctx); err != nil {
		return nil, err
	}

	return &trigger.SaveHolidaySourceRes{}, nil
}
