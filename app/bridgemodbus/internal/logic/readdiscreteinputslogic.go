package logic

import (
	"context"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadDiscreteInputsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReadDiscreteInputsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadDiscreteInputsLogic {
	return &ReadDiscreteInputsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 读取离散输入状态 (Function Code 0x02)
func (l *ReadDiscreteInputsLogic) ReadDiscreteInputs(in *bridgemodbus.ReadDiscreteInputsReq) (*bridgemodbus.ReadDiscreteInputsRes, error) {
	// todo: add your logic here and delete this line

	return &bridgemodbus.ReadDiscreteInputsRes{}, nil
}
