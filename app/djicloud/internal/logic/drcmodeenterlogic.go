package logic

import (
	"context"
	"time"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/drc"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

const defaultDrcMaxControlTime = 30 * time.Minute

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
	data := &djisdk.DrcModeEnterData{
		MqttBroker:   toDrcMqttBroker(l.svcCtx.Config.MqttConfig),
		OsdFrequency: int(in.GetOsdFrequency()),
		HsiFrequency: int(in.GetHsiFrequency()),
	}
	tid, err := l.svcCtx.DjiClient.DrcModeEnter(l.ctx, deviceSn, data)
	if err != nil {
		l.Errorf("[drc] mode enter failed device_sn=%s: %v", deviceSn, err)
		return errRes(tid, err), nil
	}
	maxCtrlMs := in.GetMaxControlTimeMillis()
	maxTimeout := defaultDrcMaxControlTime
	if maxCtrlMs > 0 {
		maxTimeout = time.Duration(maxCtrlMs) * time.Millisecond
	}
	if err := l.svcCtx.DrcManager.Enable(l.ctx, deviceSn, drc.WithMaxTimeout(maxTimeout)); err != nil {
		l.Errorf("[drc] manager enable failed device_sn=%s: %v", deviceSn, err)
		return nil, err
	}
	return okRes(tid), nil
}
