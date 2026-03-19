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
	mdCliPool, err := l.svcCtx.GetModbusClientPool(l.ctx, in.ModbusCode)
	if err != nil {
		return nil, err
	}
	mbCli := mdCliPool.Get()
	defer mdCliPool.Put(mbCli)
	results, err := mbCli.ReadDiscreteInputs(l.ctx, uint16(in.Address), uint16(in.Quantity))
	if err != nil {
		return nil, err
	}
	return &bridgemodbus.ReadDiscreteInputsRes{
		Results: results,
		Values:  bytex.BytesToBools(results, int(in.Quantity)),
	}, nil
}
