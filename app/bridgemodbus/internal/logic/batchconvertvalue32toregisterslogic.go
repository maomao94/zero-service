package logic

import (
	"context"
	"fmt"
	"math"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"
	"zero-service/common/bytex"
	"zero-service/common/errx"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchConvertValue32ToRegistersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBatchConvertValue32ToRegistersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchConvertValue32ToRegistersLogic {
	return &BatchConvertValue32ToRegistersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 批量将 32 位数值按字序编码为两个寄存器（纯计算，不访问设备）
func (l *BatchConvertValue32ToRegistersLogic) BatchConvertValue32ToRegisters(in *bridgemodbus.BatchConvertValue32ToRegistersReq) (*bridgemodbus.BatchConvertValue32ToRegistersRes, error) {
	if len(in.Values) == 0 {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID, "values 不能为空")
	}
	if err := validateValue32Type(in.DataType); err != nil {
		return nil, err
	}
	order, err := toWordOrder(in.WordOrder)
	if err != nil {
		return nil, err
	}

	results := make([]*bridgemodbus.Value32EncodeResult, 0, len(in.Values))
	blockRegisters := make([]uint32, 0, len(in.Values)*2)
	for i, value := range in.Values {
		regs, encoded, err := encodeValue32(i, value, in.DataType, order)
		if err != nil {
			return nil, err
		}
		results = append(results, &bridgemodbus.Value32EncodeResult{
			Registers: registersToUint32(regs),
			Bytes:     registersToBytes(regs),
			Hex:       registersToHex(regs),
			Binary:    fmt.Sprintf("%016b %016b", regs[0], regs[1]),
			Value:     encoded,
		})
		blockRegisters = append(blockRegisters, uint32(regs[0]), uint32(regs[1]))
	}
	return &bridgemodbus.BatchConvertValue32ToRegistersRes{
		Results:   results,
		Registers: blockRegisters,
	}, nil
}

// encodeValue32 校验单个数值的范围并按类型编码为两个寄存器。
// 第二个返回值是实际被编码的数值：int32/uint32 原值即整数，float32 会回落为
// float32 精度（如 0.1 → 0.10000000149011612），供响应回显，避免与设备读回值对不上。
func encodeValue32(index int, value float64, dataType bridgemodbus.Value32Type, order bytex.WordOrder) ([2]uint16, float64, error) {
	switch dataType {
	case bridgemodbus.Value32Type_VALUE32_UINT32:
		if value < 0 || value > math.MaxUint32 || value != math.Trunc(value) {
			return [2]uint16{}, 0, errx.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID,
				"第 %d 个无符号 32 位值 %v 超出范围 [0, 4294967295] 或不是整数", index+1, value)
		}
		regs, err := bytex.SplitUint32(uint32(value), order)
		return regs, value, err
	case bridgemodbus.Value32Type_VALUE32_INT32:
		if value < math.MinInt32 || value > math.MaxInt32 || value != math.Trunc(value) {
			return [2]uint16{}, 0, errx.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID,
				"第 %d 个有符号 32 位值 %v 超出范围 [-2147483648, 2147483647] 或不是整数", index+1, value)
		}
		regs, err := bytex.SplitInt32(int32(value), order)
		return regs, value, err
	default: // VALUE32_FLOAT32
		// 上界以转换结果为准：略大于 MaxFloat32 的 double（如文档标称的 3.4028235e38）
		// 仍会落到有限的 MaxFloat32，属于可表示范围，不应拒绝；真正溢出成 ±Inf 才拒绝。
		f32 := float32(value)
		if math.IsNaN(value) || math.IsInf(float64(f32), 0) {
			return [2]uint16{}, 0, errx.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID,
				"第 %d 个浮点值 %v 无法用有限 float32 表示", index+1, value)
		}
		regs, err := bytex.SplitFloat32(f32, order)
		return regs, float64(f32), err
	}
}
