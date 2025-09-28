package logic

import (
	"context"

	"zero-service/facade/mqttstream/internal/svc"
	"zero-service/facade/mqttstream/mqttstream"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReceiveMessageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReceiveMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReceiveMessageLogic {
	return &ReceiveMessageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 接收消息
func (l *ReceiveMessageLogic) ReceiveMessage(in *mqttstream.ReceiveMessageReq) (*mqttstream.ReceiveMessageRes, error) {
	// todo: add your logic here and delete this line

	return &mqttstream.ReceiveMessageRes{}, nil
}
