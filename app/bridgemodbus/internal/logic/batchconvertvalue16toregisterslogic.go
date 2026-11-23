package logic

import (
	"context"
	"zero-service/common/bytex"
	"zero-service/common/errx"
	"zero-service/third_party/extproto"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchConvertValue16ToRegistersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchConvertValue16ToRegistersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchConvertValue16ToRegistersLogic {
	return &BatchConvertValue16ToRegistersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 批量转换十进制数值为Modbus寄存器格式
func (l *BatchConvertValue16ToRegistersLogic) BatchConvertValue16ToRegisters(in *bridgemodbus.BatchConvertValue16ToRegistersReq) (*bridgemodbus.BatchConvertValue16ToRegistersRes, error) {
	registers, err := toRegisters(in.Values, in.Unsigned)
	if err != nil {
		return nil, err
	}
	bv := bytex.Uint16SliceToBinaryValues(registers)
	return &bridgemodbus.BatchConvertValue16ToRegistersRes{
		Uint16Values: bytex.Uint16SliceToUint32Slice(bv.Uint16),
		Int16Values:  bytex.Int16SliceToInt32Slice(bv.Int16),
		HexValues:    bv.Hex,
		BinaryValues: bv.Binary,
		Bytes:        bv.Bytes,
	}, nil
}

// toRegisters 按 signed 语义把 int32 数值校验并转换为 16 位寄存器值。
func toRegisters(values []int32, unsigned bool) ([]uint16, error) {
	if unsigned {
		// 负数按原始比特拓宽成超出 65535 的大值，由 [0, 65535] 校验统一拦截
		registers, errIdx, err := bytex.Uint32SliceToUint16SliceValidate(bytex.Int32SliceToUint32Slice(values))
		if err != nil {
			return nil, errx.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID,
				"第 %d 个无符号值 %d 超出 16 位寄存器范围 [0, 65535]", errIdx+1, values[errIdx])
		}
		return registers, nil
	}
	// 有符号值按 int16 补码落到寄存器，负数在寄存器里就是高位全 1
	registers, errIdx, err := bytex.Int32SliceToUint16SliceValidate(values)
	if err != nil {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID,
			"第 %d 个有符号值 %d 超出 16 位寄存器范围 [-32768, 32767]", errIdx+1, values[errIdx])
	}
	return registers, nil
}
