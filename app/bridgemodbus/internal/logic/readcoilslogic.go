package logic

import (
	"context"
	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"
	"zero-service/common/bytex"
	"zero-service/common/ctxdata"

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
	auth := ctxdata.GetAuthorization(l.ctx)
	l.Infof("token %s", auth)
	mdCliPool, err := l.svcCtx.GetModbusClientPool(l.ctx, in.ModbusCode)
	if err != nil {
		return nil, err
	}
	mbCli := mdCliPool.Get()
	defer mdCliPool.Put(mbCli)

	results, err := mbCli.ReadCoils(l.ctx, uint16(in.Address), uint16(in.Quantity))
	if err != nil {
		return nil, err
	}
	return &bridgemodbus.ReadCoilsRes{
		Results: results,
		Values:  bytex.BytesToBools(results, int(in.Quantity)),
	}, nil
}
