package logic

import (
	"context"

	"zero-service/facade/streamevent/internal/svc"
	"zero-service/facade/streamevent/streamevent"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReceiveMQTTMessageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReceiveMQTTMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReceiveMQTTMessageLogic {
	return &ReceiveMQTTMessageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 接收MQTT消息
func (l *ReceiveMQTTMessageLogic) ReceiveMQTTMessage(in *streamevent.ReceiveMQTTMessageReq) (*streamevent.ReceiveMQTTMessageRes, error) {
	// todo: add your logic here and delete this line

	return &streamevent.ReceiveMQTTMessageRes{}, nil
}
