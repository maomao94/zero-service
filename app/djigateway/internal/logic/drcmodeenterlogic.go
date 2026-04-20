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
	data := &djisdk.DrcModeEnterData{
		MqttBroker: in.MqttBroker,
		ClientID:   in.ClientId,
		Username:   in.Username,
		Password:   in.Password,
		ExpireTime: in.ExpireTime,
		EnableOSD:  in.EnableOsd,
	}
	tid, err := l.svcCtx.DjiClient.DrcModeEnter(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[drc] drc mode enter failed: %v", err)
		return &djigateway.CommonRes{Code: -1, Message: err.Error(), Tid: tid}, nil
	}
	return &djigateway.CommonRes{Code: 0, Message: "success", Tid: tid}, nil
}
