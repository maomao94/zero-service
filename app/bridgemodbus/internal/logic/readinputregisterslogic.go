package logic

import (
	"context"
	"zero-service/common/bytex"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadInputRegistersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReadInputRegistersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadInputRegistersLogic {
	return &ReadInputRegistersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 读取输入寄存器 (Function Code 0x04)
func (l *ReadInputRegistersLogic) ReadInputRegisters(in *bridgemodbus.ReadInputRegistersReq) (*bridgemodbus.ReadInputRegistersRes, error) {
	address, err := validateAddress("address", in.Address)
	if err != nil {
		return nil, err
	}
	quantity, err := validateQuantity("quantity", in.Quantity, maxReadRegisters)
	if err != nil {
		return nil, err
	}

	mbCli, err := l.svcCtx.GetModbusClient(l.ctx, in.ModbusCode)
	if err != nil {
		return nil, err
	}

	results, err := mbCli.ReadInputRegisters(l.ctx, address, quantity)
	if err != nil {
		return nil, wrapModbusError("读取输入寄存器", err)
	}
	bv := bytex.BytesToBinaryValues(results)
	return &bridgemodbus.ReadInputRegistersRes{
		Results:      results,
		UintValues:   bytex.Uint16SliceToUint32Slice(bv.Uint16),
		IntValues:    bytex.Int16SliceToInt32Slice(bv.Int16),
		HexValues:    bv.Hex,
		BinaryValues: bv.Binary,
	}, nil
}
