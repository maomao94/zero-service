package logic

import (
	"context"
	"errors"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type SetHolidaySourceEnabledLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSetHolidaySourceEnabledLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetHolidaySourceEnabledLogic {
	return &SetHolidaySourceEnabledLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 设置中国大陆节假日源配置启用状态
func (l *SetHolidaySourceEnabledLogic) SetHolidaySourceEnabled(in *trigger.SetHolidaySourceEnabledReq) (*trigger.SetHolidaySourceEnabledRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	if err := validateHolidayDate(in.Date); err != nil {
		return nil, err
	}
	if err := l.svcCtx.HolidaySource.SetEnabled(l.ctx, in.Date, in.Enabled); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
		}
		return nil, err
	}
	if err := l.svcCtx.HolidayCalendar.Reload(l.ctx); err != nil {
		return nil, err
	}

	return &trigger.SetHolidaySourceEnabledRes{}, nil
}
