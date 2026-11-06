package logic

import (
	"context"
	"zero-service/common/bytex"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type WriteMultipleCoilsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewWriteMultipleCoilsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WriteMultipleCoilsLogic {
	return &WriteMultipleCoilsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 写多个线圈 (Function Code 0x0F)
func (l *WriteMultipleCoilsLogic) WriteMultipleCoils(in *bridgemodbus.WriteMultipleCoilsReq) (*bridgemodbus.WriteMultipleCoilsRes, error) {
	address, err := validateAddress("address", in.Address)
	if err != nil {
		return nil, err
	}
	quantity, err := validateQuantity("quantity", in.Quantity, maxWriteBits)
	if err != nil {
		return nil, err
	}
	if err := validateValuesCount("quantity", in.Quantity, len(in.Values)); err != nil {
		return nil, err
	}

	mbCli, err := l.svcCtx.GetModbusClient(l.ctx, in.ModbusCode)
	if err != nil {
		return nil, err
	}

	binaryValues := bytex.BoolsToBitValues(in.Values)
	l.Infof("写多个线圈: 0x%X, bools=%v, binary=%v", binaryValues.Bytes, binaryValues.Bools, binaryValues.Binary)
	results, err := mbCli.WriteMultipleCoils(l.ctx, address, quantity, binaryValues.Bytes)
	if err != nil {
		return nil, wrapModbusWriteError("写多个线圈", err)
	}
	return &bridgemodbus.WriteMultipleCoilsRes{
		Results: results,
	}, nil
}
