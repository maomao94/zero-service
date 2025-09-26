package logic

import (
	"context"
	"errors"
	"fmt"
	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"
	"zero-service/common/modbusx"

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

	results, err := mbCli.ReadCoils(l.ctx, uint16(in.Address), uint16(in.Quantity))
	if err != nil {
		return nil, err
	}
	return &bridgemodbus.ReadCoilsRes{
		Results: results,
		Values:  modbusx.BytesToBools(results, int(in.Quantity)),
	}, nil
}
