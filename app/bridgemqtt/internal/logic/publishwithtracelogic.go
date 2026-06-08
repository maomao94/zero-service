package logic

import (
	"context"

	"zero-service/app/bridgemqtt/bridgemqtt"
	"zero-service/app/bridgemqtt/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type PublishWithTraceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPublishWithTraceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublishWithTraceLogic {
	return &PublishWithTraceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 带 traceId 的发布消息 内部服务链路追踪
func (l *PublishWithTraceLogic) PublishWithTrace(in *bridgemqtt.PublishWithTraceReq) (*bridgemqtt.PublishWithTraceRes, error) {
	traceID, err := l.svcCtx.MqttClient.PublishWithTrace(l.ctx, in.Topic, in.Payload)
	if err != nil {
		return nil, err
	}
	return &bridgemqtt.PublishWithTraceRes{
		TraceId: traceID,
	}, nil
}
