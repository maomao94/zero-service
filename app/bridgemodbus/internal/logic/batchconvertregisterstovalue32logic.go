package logic

import (
	"context"
	"math"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"
	"zero-service/common/bytex"
	"zero-service/common/errx"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchConvertRegistersToValue32Logic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchConvertRegistersToValue32Logic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchConvertRegistersToValue32Logic {
	return &BatchConvertRegistersToValue32Logic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 批量将连续寄存器按字序解码为 32 位数值（纯计算，不访问设备）
func (l *BatchConvertRegistersToValue32Logic) BatchConvertRegistersToValue32(in *bridgemodbus.BatchConvertRegistersToValue32Req) (*bridgemodbus.BatchConvertRegistersToValue32Res, error) {
	if len(in.Registers) == 0 {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID, "registers 不能为空")
	}
	if len(in.Registers)%2 != 0 {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID,
			"registers 数量必须为偶数（每 2 个寄存器对应一个 32 位值），当前为 %d 个", len(in.Registers))
	}
	if err := validateValue32Type(in.DataType); err != nil {
		return nil, err
	}
	order, err := toWordOrder(in.WordOrder)
	if err != nil {
		return nil, err
	}

	results := make([]*bridgemodbus.Value32DecodeResult, 0, len(in.Registers)/2)
	for i := 0; i < len(in.Registers); i += 2 {
		regs, err := decodeValue32Pair(i/2, in.Registers[i:i+2])
		if err != nil {
			return nil, err
		}
		// order 已在入口校验、regs 已在 decodeValue32Pair 校验，CombineUint32 不会失败；
		// 这里仍不忽略 error，保持错误可追溯而不是静默吞掉。
		bits, err := bytex.CombineUint32(regs, order)
		if err != nil {
			return nil, err
		}
		result := &bridgemodbus.Value32DecodeResult{
			Registers:   registersToUint32(regs),
			Bytes:       registersToBytes(regs),
			Hex:         registersToHex(regs),
			Value:       value32FromBits(bits, in.DataType),
			Uint32Value: bits,
			Int32Value:  int32(bits),
		}
		// 浮点视图只在按 float32 解码时给出，避免整数比特被当作浮点后输出 NaN/Infinity 字符串。
		if in.DataType == bridgemodbus.Value32Type_VALUE32_FLOAT32 {
			result.Float32Value = math.Float32frombits(bits)
		}
		results = append(results, result)
	}
	return &bridgemodbus.BatchConvertRegistersToValue32Res{Results: results}, nil
}

// decodeValue32Pair 校验一组寄存器（2 个）并转换为两个 uint16。
func decodeValue32Pair(index int, values []uint32) ([2]uint16, error) {
	var regs [2]uint16
	for j, v := range values {
		if v > maxRegisterValue {
			return [2]uint16{}, errx.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID,
				"第 %d 组第 %d 个寄存器值 %d 超出范围 [0, 65535]", index+1, j+1, v)
		}
		regs[j] = uint16(v)
	}
	return regs, nil
}

// value32FromBits 按数值类型把 32 位比特解释为主结果。
func value32FromBits(bits uint32, dataType bridgemodbus.Value32Type) float64 {
	switch dataType {
	case bridgemodbus.Value32Type_VALUE32_UINT32:
		return float64(bits)
	case bridgemodbus.Value32Type_VALUE32_INT32:
		return float64(int32(bits))
	default: // VALUE32_FLOAT32
		return float64(math.Float32frombits(bits))
	}
}
