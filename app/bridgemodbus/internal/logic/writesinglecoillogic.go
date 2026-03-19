package logic

import (
	"context"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type WriteSingleCoilLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewWriteSingleCoilLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WriteSingleCoilLogic {
	return &WriteSingleCoilLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 写单个线圈 (Function Code 0x05)
func (l *WriteSingleCoilLogic) WriteSingleCoil(in *bridgemodbus.WriteSingleCoilReq) (*bridgemodbus.WriteSingleCoilRes, error) {
	mdCliPool, err := l.svcCtx.GetModbusClientPool(l.ctx, in.ModbusCode)
	if err != nil {
		return nil, err
	}
	mbCli := mdCliPool.Get()
	defer mdCliPool.Put(mbCli)

	// 将 bool 值转换为 uint16: false=0x0000, true=0xFF00
	var value uint16
	if in.Value {
		value = 0xFF00
	} else {
		value = 0x0000
	}
	l.Infof("写单个线圈: 0x%X", value)
	results, err := mbCli.WriteSingleCoil(l.ctx, uint16(in.Address), value)
	if err != nil {
		return nil, err
	}
	return &bridgemodbus.WriteSingleCoilRes{
		Results: results,
	}, nil
}
