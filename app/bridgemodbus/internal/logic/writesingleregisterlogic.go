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

type WriteSingleRegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewWriteSingleRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WriteSingleRegisterLogic {
	return &WriteSingleRegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 写单个保持寄存器 (Function Code 0x06)
func (l *WriteSingleRegisterLogic) WriteSingleRegister(in *bridgemodbus.WriteSingleRegisterReq) (*bridgemodbus.WriteSingleRegisterRes, error) {
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

	results, err := mbCli.WriteSingleRegister(l.ctx, uint16(in.Address), uint16(in.Value))
	if err != nil {
		return nil, err
	}
	return &bridgemodbus.WriteSingleRegisterRes{
		Results: results,
	}, nil
}
