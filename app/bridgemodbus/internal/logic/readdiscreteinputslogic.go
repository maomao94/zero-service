package logic

import (
	"context"
	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"
	"zero-service/common/bytex"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadDiscreteInputsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReadDiscreteInputsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadDiscreteInputsLogic {
	return &ReadDiscreteInputsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 读取离散输入状态 (Function Code 0x02)
func (l *ReadDiscreteInputsLogic) ReadDiscreteInputs(in *bridgemodbus.ReadDiscreteInputsReq) (*bridgemodbus.ReadDiscreteInputsRes, error) {
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

	results, err := mbCli.ReadDiscreteInputs(l.ctx, address, quantity)
	if err != nil {
		return nil, wrapModbusError("读取离散输入", err)
	}
	return &bridgemodbus.ReadDiscreteInputsRes{
		Results: results,
		Values:  bytex.BytesToBools(results, int(quantity)),
	}, nil
}
