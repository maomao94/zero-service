package logic

import (
	"context"
	"zero-service/common/bytex"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadHoldingRegistersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReadHoldingRegistersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadHoldingRegistersLogic {
	return &ReadHoldingRegistersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 读取保持寄存器 (Function Code 0x03)
func (l *ReadHoldingRegistersLogic) ReadHoldingRegisters(in *bridgemodbus.ReadHoldingRegistersReq) (*bridgemodbus.ReadHoldingRegistersRes, error) {
	mdCliPool, err := l.svcCtx.GetModbusClientPool(l.ctx, in.ModbusCode)
	if err != nil {
		return nil, err
	}
	mbCli := mdCliPool.Get()
	defer mdCliPool.Put(mbCli)
	results, err := mbCli.ReadHoldingRegisters(l.ctx, uint16(in.Address), uint16(in.Quantity))
	if err != nil {
		return nil, err
	}
	// 每个寄存器占 2 个字节，所以寄存器数量 = bytes 长度 / 2
	//n := len(results) / 2
	//hexValues := make([]string, 0, n)
	//for i := 0; i < len(results); i += 2 {
	//	hi := results[i]   // 高字节
	//	lo := results[i+1] // 低字节
	//
	//	val := uint16(hi)<<8 | uint16(lo)                         // 拼成 16 位寄存器值
	//	hexValues = append(hexValues, fmt.Sprintf("0x%04X", val)) // 转 16 进制字符串
	//}
	bv := bytex.BytesToBinaryValues(results)
	return &bridgemodbus.ReadHoldingRegistersRes{
		Results:      results,
		UintValues:   bytex.Uint16SliceToUint32Slice(bv.Uint16),
		IntValues:    bytex.Int16SliceToInt32Slice(bv.Int16),
		HexValues:    bv.Hex,
		BinaryValues: bv.Binary,
	}, nil
}
