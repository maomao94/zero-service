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
	address, err := validateAddress("address", in.Address)
	if err != nil {
		return nil, err
	}
	quantity, err := validateQuantity("quantity", in.Quantity, maxWriteRegisters)
	if err != nil {
		return nil, err
	}
	if err := validateValuesCount("quantity", in.Quantity, len(in.Values)); err != nil {
		return nil, err
	}

	uint16Values, errIdx, err := bytex.Uint32SliceToUint16SliceValidate(in.Values)
	if err != nil {
		return nil, errx.NewErrorByPbCode(extproto.Code__1_00_PARAM_INVALID,
			"第 %d 个值超过 16 位寄存器的最大值 (65535)", errIdx+1)
	}

	mbCli, err := l.svcCtx.GetModbusClient(l.ctx, in.ModbusCode)
	if err != nil {
		return nil, err
	}

	binaryValues := bytex.Uint16SliceToBinaryValues(uint16Values)
	l.Infof("写多个保持寄存器: 0x%X, hex=%v, uint16=%v, int16=%v, binary=%v", binaryValues.Bytes, binaryValues.Hex, binaryValues.Uint16, binaryValues.Int16, binaryValues.Binary)
	results, err := mbCli.WriteMultipleRegisters(l.ctx, address, quantity, binaryValues.Bytes)
	if err != nil {
		return nil, wrapModbusWriteError("写多个保持寄存器", err)
	}
	return &bridgemodbus.WriteMultipleRegistersRes{
		Results: results,
	}, nil
}
