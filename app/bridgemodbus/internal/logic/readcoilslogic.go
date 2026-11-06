package logic

import (
	"context"
	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"
	"zero-service/common/bytex"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadCoilsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReadCoilsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadCoilsLogic {
	return &ReadCoilsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 读取线圈状态 (Function Code 0x01)
func (l *ReadCoilsLogic) ReadCoils(in *bridgemodbus.ReadCoilsReq) (*bridgemodbus.ReadCoilsRes, error) {
	address, err := validateAddress("address", in.Address)
	if err != nil {
		return nil, err
	}
	quantity, err := validateQuantity("quantity", in.Quantity, maxReadBits)
	if err != nil {
		return nil, err
	}

	mbCli, err := l.svcCtx.GetModbusClient(l.ctx, in.ModbusCode)
	if err != nil {
		return nil, err
	}

	results, err := mbCli.ReadCoils(l.ctx, address, quantity)
	if err != nil {
		return nil, wrapModbusError("读取线圈", err)
	}
	return &bridgemodbus.ReadCoilsRes{
		Results: results,
		Values:  bytex.BytesToBools(results, int(quantity)),
	}, nil
}
