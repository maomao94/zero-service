package logic

import (
	"context"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type WriteSingleCoilLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewWriteSingleCoilLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WriteSingleCoilLogic {
	return &WriteSingleCoilLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 写单个线圈 (Function Code 0x05)
func (l *WriteSingleCoilLogic) WriteSingleCoil(in *bridgemodbus.WriteSingleCoilReq) (*bridgemodbus.WriteSingleCoilRes, error) {
	// todo: add your logic here and delete this line

	return &bridgemodbus.WriteSingleCoilRes{}, nil
}
