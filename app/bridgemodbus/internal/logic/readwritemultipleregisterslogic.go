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
	l.Infof("读写多个保持寄存器: 写值=0x%X, hex=%v, uint16=%v, int16=%v, binary=%v", binaryValues.Bytes, binaryValues.Hex, binaryValues.Uint16, binaryValues.Int16, binaryValues.Binary)
	results, err := mbCli.ReadWriteMultipleRegisters(l.ctx, uint16(in.ReadAddress), uint16(in.ReadQuantity), uint16(in.WriteAddress), uint16(in.WriteQuantity), binaryValues.Bytes)
	if err != nil {
		return nil, err
	}
	binaryValues = bytex.BytesToBinaryValues(results)
	return &bridgemodbus.ReadWriteMultipleRegistersRes{
		Results:      results,
		UintValues:   bytex.Uint16SliceToUint32Slice(binaryValues.Uint16),
		IntValues:    bytex.Int16SliceToInt32Slice(binaryValues.Int16),
		HexValues:    binaryValues.Hex,
		BinaryValues: binaryValues.Binary,
	}, nil
}
