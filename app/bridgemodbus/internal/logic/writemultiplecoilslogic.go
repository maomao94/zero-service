package logic

import (
	"context"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type WriteMultipleCoilsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewWriteMultipleCoilsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WriteMultipleCoilsLogic {
	return &WriteMultipleCoilsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 写多个线圈 (Function Code 0x0F)
func (l *WriteMultipleCoilsLogic) WriteMultipleCoils(in *bridgemodbus.WriteMultipleCoilsReq) (*bridgemodbus.WriteMultipleCoilsRes, error) {
	// todo: add your logic here and delete this line

	return &bridgemodbus.WriteMultipleCoilsRes{}, nil
}
