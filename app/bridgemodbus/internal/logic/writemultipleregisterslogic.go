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

type WriteMultipleRegistersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewWriteMultipleRegistersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WriteMultipleRegistersLogic {
	return &WriteMultipleRegistersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 写多个保持寄存器 (Function Code 0x10)
func (l *WriteMultipleRegistersLogic) WriteMultipleRegisters(in *bridgemodbus.WriteMultipleRegistersReq) (*bridgemodbus.WriteMultipleRegistersRes, error) {
	if int(in.Quantity) != len(in.Values) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "数量与值数量不一致")
	}

	mdCliPool, err := l.svcCtx.GetModbusClientPool(l.ctx, in.ModbusCode)
	if err != nil {
		return nil, err
	}
	mbCli := mdCliPool.Get()
	defer mdCliPool.Put(mbCli)

	for i, v := range in.Values {
		if v > 65535 {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "第 %d 个值超过 16 位寄存器的最大值 (65535)", i+1)
		}
	}

	uint16Values := make([]uint16, len(in.Values))
	for i, v := range in.Values {
		uint16Values[i] = uint16(v)
	}
	binaryValues := bytex.Uint16SliceToBinaryValues(uint16Values)
	l.Infof("写多个保持寄存器: 0x%X, hex=%v, uint16=%v, int16=%v, binary=%v", binaryValues.Bytes, binaryValues.Hex, binaryValues.Uint16, binaryValues.Int16, binaryValues.Binary)
	results, err := mbCli.WriteMultipleRegisters(l.ctx, uint16(in.Address), uint16(in.Quantity), binaryValues.Bytes)
	if err != nil {
		return nil, err
	}
	return &bridgemodbus.WriteMultipleRegistersRes{
		Results: results,
	}, nil
}
