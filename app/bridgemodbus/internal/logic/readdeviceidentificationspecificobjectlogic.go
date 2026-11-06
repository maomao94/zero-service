package logic

import (
	"context"
	"zero-service/common/errx"
	"zero-service/third_party/extproto"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadDeviceIdentificationSpecificObjectLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReadDeviceIdentificationSpecificObjectLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadDeviceIdentificationSpecificObjectLogic {
	return &ReadDeviceIdentificationSpecificObjectLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 读取特定 Object ID 的设备标识 (Function Code 0x2B / 0x0E)
func (l *ReadDeviceIdentificationSpecificObjectLogic) ReadDeviceIdentificationSpecificObject(in *bridgemodbus.ReadDeviceIdentificationSpecificObjectReq) (*bridgemodbus.ReadDeviceIdentificationSpecificObjectRes, error) {
	if in.ObjectId > 0xFF {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID,
			"objectId 超出范围 [0, 255]（当前 %d）", in.ObjectId)
	}

	mbCli, err := l.svcCtx.GetModbusClient(l.ctx, in.ModbusCode)
	if err != nil {
		return nil, err
	}

	results, err := mbCli.ReadDeviceIdentificationSpecificObject(l.ctx, byte(in.ObjectId))
	if err != nil {
		return nil, wrapModbusError("读取指定设备标识对象", err)
	}
	return &bridgemodbus.ReadDeviceIdentificationSpecificObjectRes{
		Results:         toDeviceIDDec(results),
		HexResults:      toDeviceIDHex(results),
		SemanticResults: toDeviceIDSemantic(results),
	}, nil
}
