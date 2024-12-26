package logic

import (
	"context"

	"github.com/golang-module/carbon/v2"
	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"time"
	"zero-service/common/asynqx"
	"zero-service/common/ctxdata"
	"zero-service/zerorpc/tasktype"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

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
	spanCtx, span := asynqx.StartAsynqProducerSpan(l.ctx, tasktype.DeferTriggerTask)
	defer span.End()
	carrier := &propagation.HeaderCarrier{}
	otel.GetTextMapPropagator().Inject(spanCtx, carrier)
	msg := &ctxdata.MsgBody{
		MsgId:   in.MsgId,
		Carrier: carrier,
		Msg:     in.Body,
		Url:     in.Url,
	}
	payload, err := jsonx.Marshal(msg)
	if err != nil {
		return nil, err
	}
	var d time.Duration
	if len(in.TriggerTime) > 0 {
		triggerTime := carbon.Parse(in.TriggerTime)
		if triggerTime.Error != nil {
			return nil, triggerTime.Error
		}
		internal := carbon.Now().DiffInSeconds(triggerTime)
		if internal < 0 {
			return nil, errors.New("triggerTime is invalid")
		}
		d = time.Duration(internal) * time.Second
	} else {
		d = time.Duration(in.ProcessIn) * time.Second
	}
	opts := []asynq.Option{}
	if len(in.GetMsgId()) != 0 {
		opts = append(opts, asynq.TaskID(in.GetMsgId()))
	}
	if len(in.Group) != 0 {
		opts = append(opts, asynq.Group(in.GetGroup()))
	}
	opts = append(opts, asynq.Queue("critical"), asynq.ProcessIn(d), asynq.Retention(24*time.Hour))
	taskInfo, err := l.svcCtx.AsynqClient.Enqueue(asynq.NewTask(tasktype.DeferTriggerTask, []byte(payload)), opts...)
	if err != nil {
		return nil, err
	}
	return &trigger.SendTriggerRes{
		TraceId:  traceID,
		Id:       taskInfo.ID,
		Queue:    taskInfo.Queue,
		MaxRetry: int64(taskInfo.MaxRetry),
		Retried:  int64(taskInfo.Retried),
		Group:    taskInfo.Group,
	}, nil
}
