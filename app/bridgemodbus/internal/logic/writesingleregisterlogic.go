package logic

import (
	"context"
	"zero-service/common/bytex"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type WriteSingleRegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewWriteSingleRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WriteSingleRegisterLogic {
	return &WriteSingleRegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 写单个保持寄存器 (Function Code 0x06)
func (l *WriteSingleRegisterLogic) WriteSingleRegister(in *bridgemodbus.WriteSingleRegisterReq) (*bridgemodbus.WriteSingleRegisterRes, error) {
	address, err := validateAddress("address", in.Address)
	if err != nil {
		return nil, err
	}
	uint16Value, err := validate16BitValue("value", in.Value)
	if err != nil {
		return nil, err
	}

	mbCli, err := l.svcCtx.GetModbusClient(l.ctx, in.ModbusCode)
	if err != nil {
		return nil, err
	}

	binaryValues := bytex.Uint16SliceToBinaryValues([]uint16{uint16Value})
	l.Infof("写单个保持寄存器: address=%d, value=0x%X, hex=%v, uint16=%v, int16=%v, binary=%v",
		address, in.Value, binaryValues.Hex, binaryValues.Uint16, binaryValues.Int16, binaryValues.Binary)
	results, err := mbCli.WriteSingleRegister(l.ctx, address, uint16Value)
	if err != nil {
		return nil, wrapModbusWriteError("写单个保持寄存器", err)
	}
	return &bridgemodbus.WriteSingleRegisterRes{
		Results: results,
	}, nil
}
