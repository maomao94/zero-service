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

type MaskWriteRegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMaskWriteRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MaskWriteRegisterLogic {
	return &MaskWriteRegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 屏蔽写保持寄存器 (Function Code 0x16)
func (l *MaskWriteRegisterLogic) MaskWriteRegister(in *bridgemodbus.MaskWriteRegisterReq) (*bridgemodbus.MaskWriteRegisterRes, error) {
	mdCliPool, err := l.svcCtx.GetModbusClientPool(l.ctx, in.ModbusCode)
	if err != nil {
		return nil, err
	}
	mbCli := mdCliPool.Get()
	defer mdCliPool.Put(mbCli)

	andMask, err := bytex.Uint32ToUint16Validate(in.AndMask)
	if err != nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ, "AND 掩码超过 16 位寄存器的最大值 (65535)")
	}
	orMask, err := bytex.Uint32ToUint16Validate(in.OrMask)
	if err != nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ, "OR 掩码超过 16 位寄存器的最大值 (65535)")
	}

	results, err := mbCli.MaskWriteRegister(l.ctx, uint16(in.Address), andMask, orMask)
	if err != nil {
		return nil, err
	}
	return &bridgemodbus.MaskWriteRegisterRes{
		Results: results,
	}, nil
}
