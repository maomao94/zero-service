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

type BatchConvertDecimalToRegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchConvertDecimalToRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchConvertDecimalToRegisterLogic {
	return &BatchConvertDecimalToRegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 批量转换十进制数值为Modbus寄存器格式
func (l *BatchConvertDecimalToRegisterLogic) BatchConvertDecimalToRegister(in *bridgemodbus.BatchConvertDecimalToRegisterReq) (*bridgemodbus.BatchConvertDecimalToRegisterRes, error) {
	var uint16Values []uint16
	if in.Unsigned {
		// int32 → uint32 → uint16（带范围校验 [0, 65535]）
		uint32s := make([]uint32, len(in.Values))
		for i, v := range in.Values {
			uint32s[i] = uint32(v)
		}
		uint16s, errIdx, err := bytex.Uint32SliceToUint16SliceValidate(uint32s)
		if err != nil {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "第 %d 个无符号值 %d 超出 16 位寄存器范围 [0, 65535]", errIdx+1, in.Values[errIdx])
		}
		uint16Values = uint16s
	} else {
		// int32 → int16 → uint16（带范围校验 [-32768, 32767]）
		int16s, errIdx, err := bytex.Int32SliceToInt16SliceValidate(in.Values)
		if err != nil {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "第 %d 个有符号值 %d 超出 16 位寄存器范围 [-32768, 32767]", errIdx+1, in.Values[errIdx])
		}
		uint16Values = make([]uint16, len(int16s))
		for i, v := range int16s {
			uint16Values[i] = uint16(v)
		}
	}
	binaryValues := bytex.Uint16SliceToBinaryValues(uint16Values)
	response := &bridgemodbus.BatchConvertDecimalToRegisterRes{
		Uint16Values: bytex.Uint16SliceToUint32Slice(binaryValues.Uint16),
		Int16Values:  bytex.Int16SliceToInt32Slice(binaryValues.Int16),
		HexValues:    binaryValues.Hex,
		BinaryValues: binaryValues.Binary,
		Bytes:        binaryValues.Bytes,
	}
	return response, nil
}
