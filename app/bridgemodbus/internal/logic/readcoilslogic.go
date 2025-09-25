package logic

import (
	"context"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadCoilsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReadCoilsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadCoilsLogic {
	return &ReadCoilsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 读取线圈状态 (Function Code 0x01)
func (l *ReadCoilsLogic) ReadCoils(in *bridgemodbus.ReadCoilsReq) (*bridgemodbus.ReadCoilsRes, error) {
	// todo: add your logic here and delete this line

	return &bridgemodbus.ReadCoilsRes{}, nil
}
