package logic

import (
	"context"

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
	// todo: add your logic here and delete this line

	return &bridgemodbus.WriteMultipleRegistersRes{}, nil
}
