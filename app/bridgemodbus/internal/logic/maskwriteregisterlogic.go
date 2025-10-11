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

type MaskWriteRegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMaskWriteRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MaskWriteRegisterLogic {
	return &MaskWriteRegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 屏蔽写保持寄存器 (Function Code 0x16)
func (l *MaskWriteRegisterLogic) MaskWriteRegister(in *bridgemodbus.MaskWriteRegisterReq) (*bridgemodbus.MaskWriteRegisterRes, error) {
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

	results, err := mbCli.MaskWriteRegister(l.ctx, uint16(in.Address), uint16(in.AndMask), uint16(in.OrMask))
	if err != nil {
		return nil, err
	}
	return &bridgemodbus.MaskWriteRegisterRes{
		Results: results,
	}, nil
}
