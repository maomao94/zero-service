package logic

import (
	"context"

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
	// todo: add your logic here and delete this line

	return &bridgemodbus.ReadHoldingRegistersRes{}, nil
}
