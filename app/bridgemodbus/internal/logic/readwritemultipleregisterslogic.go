package logic

import (
	"context"
	"errors"
	"fmt"
	"zero-service/common/modbusx"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadWriteMultipleRegistersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReadWriteMultipleRegistersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadWriteMultipleRegistersLogic {
	return &ReadWriteMultipleRegistersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 读写多个保持寄存器 (Function Code 0x17)
func (l *ReadWriteMultipleRegistersLogic) ReadWriteMultipleRegisters(in *bridgemodbus.ReadWriteMultipleRegistersReq) (*bridgemodbus.ReadWriteMultipleRegistersRes, error) {
	var mdCliPool *modbusx.ModbusClientPool
	var err error
	if len(in.ModbusCode) == 0 {
		mdCliPool = l.svcCtx.ModbusClientPool
	} else {
		var ok bool
		mdCliPool, ok = l.svcCtx.Manager.GetPool(in.ModbusCode) // 关键：用=而不是:=，避免局部变量
		if !ok {
			mdCliPool, err = l.svcCtx.AddPool(l.ctx, in.ModbusCode)
			if err != nil {
				return nil, fmt.Errorf("创建Modbus连接池失败: %w", err)
			}
		}
		if mdCliPool == nil {
			return nil, errors.New("获取的Modbus连接池为空")
		}
	}
	mbCli := mdCliPool.Get()
	defer mdCliPool.Put(mbCli)

	results, err := mbCli.ReadWriteMultipleRegisters(l.ctx, uint16(in.ReadAddress), uint16(in.ReadQuantity), uint16(in.WriteAddress), uint16(in.WriteQuantity), in.Values)
	if err != nil {
		return nil, err
	}
	return &bridgemodbus.ReadWriteMultipleRegistersRes{
		Results: results,
	}, nil
}
