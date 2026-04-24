package logic

import (
	"context"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/asynqx"
	"zero-service/common/msgbody"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendTriggerLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendTriggerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendTriggerLogic {
	return &SendTriggerLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendTriggerLogic) SendTrigger(in *trigger.SendTriggerReq) (*trigger.SendTriggerRes, error) {
	traceID := trace.TraceIDFromContext(l.ctx)
	spanCtx, span := asynqx.StartAsynqProducerSpan(l.ctx, asynqx.DeferTriggerTask)
	defer span.End()

	carrier := &propagation.HeaderCarrier{}
	otel.GetTextMapPropagator().Inject(spanCtx, carrier)

	msg := &msgbody.MsgBody{
		MsgId:   in.MsgId,
		Carrier: carrier,
		Msg:     in.Body,
		Url:     in.Url,
	}

	opts, payload, err := prepareEnqueue(l.ctx, l.svcCtx, in.MsgId, in.MaxRetry, in.TriggerTime, in.ProcessIn, msg)
	if err != nil {
		return nil, err
	}

	taskInfo, err := l.svcCtx.AsynqClient.Enqueue(asynq.NewTask(asynqx.DeferTriggerTask, payload), opts...)
	if err != nil {
		return nil, err
	}

	return &trigger.SendTriggerRes{
		TraceId: traceID,
		Id:      taskInfo.ID,
		Queue:   taskInfo.Queue,
	}, nil
}
