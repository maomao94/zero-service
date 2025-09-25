package logic

import (
	"context"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadDeviceIdentificationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReadDeviceIdentificationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadDeviceIdentificationLogic {
	return &ReadDeviceIdentificationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 读取设备标识 (Function Code 0x2B / 0x0E)
func (l *ReadDeviceIdentificationLogic) ReadDeviceIdentification(in *bridgemodbus.ReadDeviceIdentificationReq) (*bridgemodbus.ReadDeviceIdentificationRes, error) {
	// todo: add your logic here and delete this line

	return &bridgemodbus.ReadDeviceIdentificationRes{}, nil
}
