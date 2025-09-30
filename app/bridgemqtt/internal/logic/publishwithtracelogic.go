package logic

import (
	"context"
	"zero-service/common/mqttx"

	"zero-service/app/bridgemqtt/bridgemqtt"
	"zero-service/app/bridgemqtt/internal/svc"

	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel"
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
func (l *PublishWithTraceLogic) PublishWithTrace(in *bridgemqtt.PublishReq) (*bridgemqtt.PublishRes, error) {
	msg := mqttx.NewMessage(in.Topic, in.Payload)
	carrier := mqttx.NewMessageCarrier(msg)
	otel.GetTextMapPropagator().Inject(l.ctx, carrier)
	payload, err := jsonx.Marshal(msg)
	if err != nil {
		return nil, err
	}
	err = l.svcCtx.MqttClient.Publish(l.ctx, in.Topic, payload)
	if err != nil {
		return nil, err
	}
	return &bridgemqtt.PublishRes{}, nil
}
