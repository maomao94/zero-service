package logic

import (
	"context"
	"zero-service/common/bytex"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadHoldingRegistersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReadHoldingRegistersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadHoldingRegistersLogic {
	return &ReadHoldingRegistersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 读取保持寄存器 (Function Code 0x03)
func (l *ReadHoldingRegistersLogic) ReadHoldingRegisters(in *bridgemodbus.ReadHoldingRegistersReq) (*bridgemodbus.ReadHoldingRegistersRes, error) {
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

	results, err := mbCli.ReadHoldingRegisters(l.ctx, address, quantity)
	if err != nil {
		return nil, wrapModbusError("读取保持寄存器", err)
	}
	bv := bytex.BytesToBinaryValues(results)
	return &bridgemodbus.ReadHoldingRegistersRes{
		Results:      results,
		UintValues:   bytex.Uint16SliceToUint32Slice(bv.Uint16),
		IntValues:    bytex.Int16SliceToInt32Slice(bv.Int16),
		HexValues:    bv.Hex,
		BinaryValues: bv.Binary,
	}, nil
}
