package logic

import (
	"context"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type WriteMultipleRegistersWithDecimalLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewWriteMultipleRegistersWithDecimalLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WriteMultipleRegistersWithDecimalLogic {
	return &WriteMultipleRegistersWithDecimalLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 写多个保持寄存器（使用十进制数值）
func (l *WriteMultipleRegistersWithDecimalLogic) WriteMultipleRegistersWithDecimal(in *bridgemodbus.WriteMultipleRegistersWithDecimalReq) (*bridgemodbus.WriteMultipleRegistersWithDecimalRes, error) {
	convertLogic := NewBatchConvertDecimalToRegisterLogic(l.ctx, l.svcCtx)
	convertReq := &bridgemodbus.BatchConvertDecimalToRegisterReq{
		Values:   in.Values,
		Unsigned: in.Unsigned,
	}
	convertRes, err := convertLogic.BatchConvertDecimalToRegister(convertReq)
	if err != nil {
		return nil, err
	}
	writeLogic := NewWriteMultipleRegistersLogic(l.ctx, l.svcCtx)
	writeReq := &bridgemodbus.WriteMultipleRegistersReq{
		ModbusCode: in.ModbusCode,
		Address:    in.Address,
		Quantity:   in.Quantity,
		Values:     convertRes.Uint16Values,
	}
	writeRes, err := writeLogic.WriteMultipleRegisters(writeReq)
	if err != nil {
		return nil, err
	}
	return &bridgemodbus.WriteMultipleRegistersWithDecimalRes{
		Results: writeRes.Results,
	}, nil
}
