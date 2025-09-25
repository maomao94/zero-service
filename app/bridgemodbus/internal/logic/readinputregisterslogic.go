package logic

import (
	"context"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadInputRegistersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReadInputRegistersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadInputRegistersLogic {
	return &ReadInputRegistersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 读取输入寄存器 (Function Code 0x04)
func (l *ReadInputRegistersLogic) ReadInputRegisters(in *bridgemodbus.ReadInputRegistersReq) (*bridgemodbus.ReadInputRegistersRes, error) {
	// todo: add your logic here and delete this line

	return &bridgemodbus.ReadInputRegistersRes{}, nil
}
