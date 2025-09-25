package logic

import (
	"context"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadWriteMultipleRegistersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReadWriteMultipleRegistersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadWriteMultipleRegistersLogic {
	return &ReadWriteMultipleRegistersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 读写多个保持寄存器 (Function Code 0x17)
func (l *ReadWriteMultipleRegistersLogic) ReadWriteMultipleRegisters(in *bridgemodbus.ReadWriteMultipleRegistersReq) (*bridgemodbus.ReadWriteMultipleRegistersRes, error) {
	// todo: add your logic here and delete this line

	return &bridgemodbus.ReadWriteMultipleRegistersRes{}, nil
}
