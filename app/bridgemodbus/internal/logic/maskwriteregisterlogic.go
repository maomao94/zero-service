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
	// todo: add your logic here and delete this line

	return &bridgemodbus.MaskWriteRegisterRes{}, nil
}
