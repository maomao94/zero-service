package logic

import (
	"context"
	"zero-service/common/modbusx"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

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
	mbCli := l.svcCtx.ModbusClientPool.Get()
	defer l.svcCtx.ModbusClientPool.Put(mbCli)
	results, err := mbCli.ReadCoils(l.ctx, uint16(in.Address), uint16(in.Quantity))
	if err != nil {
		return nil, err
	}
	return &bridgemodbus.ReadDiscreteInputsRes{
		Results: results,
		Values:  modbusx.BytesToBools(results, int(in.Quantity)),
	}, nil
}
