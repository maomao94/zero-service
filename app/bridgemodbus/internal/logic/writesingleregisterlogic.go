package logic

import (
	"context"
	"zero-service/common/bytex"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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
	mdCliPool, err := l.svcCtx.GetModbusClientPool(l.ctx, in.ModbusCode)
	if err != nil {
		return nil, err
	}
	mbCli := mdCliPool.Get()
	defer mdCliPool.Put(mbCli)

	if in.Value > 65535 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ, "值超过 16 位寄存器的最大值 (65535)")
	}

	uint16Value := uint16(in.Value)
	// 创建包含单个值的切片
	uint16Values := []uint16{uint16Value}
	binaryValues := bytex.Uint16SliceToBinaryValues(uint16Values)
	l.Infof("写单个保持寄存器: 0x%X, hex=%v, uint16=%v, int16=%v, binary=%v", in.Value, binaryValues.Hex, binaryValues.Uint16, binaryValues.Int16, binaryValues.Binary)
	results, err := mbCli.WriteSingleRegister(l.ctx, uint16(in.Address), uint16(in.Value))
	if err != nil {
		return nil, err
	}
	return &bridgemodbus.WriteSingleRegisterRes{
		Results: results,
	}, nil
}
