package logic

import (
	"context"
	"zero-service/common/modbusx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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
				return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ, "创建Modbus连接池失败")
			}
		}
		if mdCliPool == nil {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ, "获取的Modbus连接池为空")
		}
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
