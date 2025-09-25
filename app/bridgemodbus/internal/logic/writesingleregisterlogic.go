package logic

import (
	"context"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type WriteSingleRegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewWriteSingleRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WriteSingleRegisterLogic {
	return &WriteSingleRegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 写单个保持寄存器 (Function Code 0x06)
func (l *WriteSingleRegisterLogic) WriteSingleRegister(in *bridgemodbus.WriteSingleRegisterReq) (*bridgemodbus.WriteSingleRegisterRes, error) {
	// todo: add your logic here and delete this line

	return &bridgemodbus.WriteSingleRegisterRes{}, nil
}
