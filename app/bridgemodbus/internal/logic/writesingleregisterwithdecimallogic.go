package logic

import (
	"context"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type WriteSingleRegisterWithDecimalLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewWriteSingleRegisterWithDecimalLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WriteSingleRegisterWithDecimalLogic {
	return &WriteSingleRegisterWithDecimalLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 写单个保持寄存器（使用十进制数值）
func (l *WriteSingleRegisterWithDecimalLogic) WriteSingleRegisterWithDecimal(in *bridgemodbus.WriteSingleRegisterWithDecimalReq) (*bridgemodbus.WriteSingleRegisterWithDecimalRes, error) {
	convertLogic := NewBatchConvertDecimalToRegisterLogic(l.ctx, l.svcCtx)
	convertReq := &bridgemodbus.BatchConvertDecimalToRegisterReq{
		Values:   []int32{in.Value},
		Unsigned: in.Unsigned,
	}
	convertRes, err := convertLogic.BatchConvertDecimalToRegister(convertReq)
	if err != nil {
		return nil, err
	}
	writeLogic := NewWriteSingleRegisterLogic(l.ctx, l.svcCtx)
	writeReq := &bridgemodbus.WriteSingleRegisterReq{
		ModbusCode: in.ModbusCode,
		Address:    in.Address,
		Value:      convertRes.Uint16Values[0],
	}
	writeRes, err := writeLogic.WriteSingleRegister(writeReq)
	if err != nil {
		return nil, err
	}
	return &bridgemodbus.WriteSingleRegisterWithDecimalRes{
		Results: writeRes.Results,
	}, nil
}
