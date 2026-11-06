package logic

import (
	"context"

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
	address, err := validateAddress("address", in.Address)
	if err != nil {
		return nil, err
	}
	andMask, err := validate16BitValue("andMask", in.AndMask)
	if err != nil {
		return nil, err
	}
	orMask, err := validate16BitValue("orMask", in.OrMask)
	if err != nil {
		return nil, err
	}

	mbCli, err := l.svcCtx.GetModbusClient(l.ctx, in.ModbusCode)
	if err != nil {
		return nil, err
	}

	results, err := mbCli.MaskWriteRegister(l.ctx, address, andMask, orMask)
	if err != nil {
		return nil, wrapModbusWriteError("掩码写寄存器", err)
	}
	return &bridgemodbus.MaskWriteRegisterRes{
		Results: results,
	}, nil
}
