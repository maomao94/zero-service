package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
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

func (l *DrcModeEnterLogic) DrcModeEnter(in *djicloud.DrcModeEnterReq) (*djicloud.CommonRes, error) {
	deviceSn := in.GetDeviceSn()
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
	tid, err := l.svcCtx.DjiClient.DrcModeEnter(l.ctx, deviceSn, data)
	if err != nil {
		l.Errorf("[drc] mode enter failed device_sn=%s tid=%s: %v", deviceSn, tid, err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
