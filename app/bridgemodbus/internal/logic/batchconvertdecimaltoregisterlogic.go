package logic

import (
	"context"
	"fmt"
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
	// 检查输入值是否在 16 位寄存器的范围内
	for i, v := range in.Values {
		if in.Unsigned {
			// 无符号整数范围：0-65535
			if v < 0 || v > 65535 {
				return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, fmt.Errorf("第 %d 个无符号值 %d 超出 16 位寄存器范围 [0, 65535]", i+1, v))
			}
		} else {
			// 有符号整数范围：-32768-32767
			if v > 32767 || v < -32768 {
				return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, fmt.Errorf("第 %d 个有符号值 %d 超出 16 位寄存器范围 [-32768, 32767]", i+1, v))
			}
		}
	}

	uint16Values := make([]uint16, len(in.Values))
	for i, v := range in.Values {
		if in.Unsigned {
			// 无符号整数直接转换
			uint16Values[i] = uint16(v)
		} else {
			// 有符号整数先转换为 int16，再转换为 uint16
			int16Val := int16(v)
			uint16Values[i] = uint16(int16Val)
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
