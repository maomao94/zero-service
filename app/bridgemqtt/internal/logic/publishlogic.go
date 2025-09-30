package logic

import (
	"context"

	"zero-service/app/bridgemqtt/bridgemqtt"
	"zero-service/app/bridgemqtt/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type PublishLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPublishLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishLogic {
	return &PublishLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 发布消息
func (l *PublishLogic) Publish(in *bridgemqtt.PublishReq) (*bridgemqtt.PublishRes, error) {
	err := l.svcCtx.MqttClient.Publish(l.ctx, in.Topic, in.Payload)
	if err != nil {
		return nil, err
	}
	return &bridgemqtt.PublishRes{}, nil
}
