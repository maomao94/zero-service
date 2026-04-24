package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type DrcModeEnterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDrcModeEnterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DrcModeEnterLogic {
	return &DrcModeEnterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DrcModeEnterLogic) DrcModeEnter(in *djigateway.DrcModeEnterReq) (*djigateway.CommonRes, error) {
	broker := in.GetMqttBroker()
	data := &djisdk.DrcModeEnterData{
		MqttBroker: djisdk.DrcMqttBroker{
			Address:    broker.GetAddress(),
			ClientID:   broker.GetClientId(),
			Username:   broker.GetUsername(),
			Password:   broker.GetPassword(),
			ExpireTime: broker.GetExpireTime(),
			EnableTLS:  broker.GetEnableTls(),
		},
		OsdFrequency: int(in.GetOsdFrequency()),
		HsiFrequency: int(in.GetHsiFrequency()),
	}
	tid, err := l.svcCtx.DjiClient.DrcModeEnter(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[drc] drc mode enter failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
