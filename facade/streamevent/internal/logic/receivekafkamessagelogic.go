package logic

import (
	"context"

	"zero-service/facade/streamevent/internal/svc"
	"zero-service/facade/streamevent/streamevent"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReceiveKafkaMessageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReceiveKafkaMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReceiveKafkaMessageLogic {
	return &ReceiveKafkaMessageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 接收kafka消息
func (l *ReceiveKafkaMessageLogic) ReceiveKafkaMessage(in *streamevent.ReceiveKafkaMessageReq) (*streamevent.ReceiveKafkaMessageRes, error) {
	// todo: add your logic here and delete this line

	return &streamevent.ReceiveKafkaMessageRes{}, nil
}
