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

type ReadFIFOQueueLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReadFIFOQueueLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadFIFOQueueLogic {
	return &ReadFIFOQueueLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 读取 FIFO 队列 (Function Code 0x18)
func (l *ReadFIFOQueueLogic) ReadFIFOQueue(in *bridgemodbus.ReadFIFOQueueReq) (*bridgemodbus.ReadFIFOQueueRes, error) {
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

	results, err := mbCli.ReadFIFOQueue(l.ctx, uint16(in.Address))
	if err != nil {
		return nil, err
	}
	return &bridgemodbus.ReadFIFOQueueRes{
		Results: results,
	}, nil
}
