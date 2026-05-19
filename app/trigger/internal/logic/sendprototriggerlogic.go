package logic

import (
	"context"
	"regexp"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/internal/taskpayload"
	"zero-service/app/trigger/trigger"
	"zero-service/common/asynqx"

	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel/propagation"
	tracex "zero-service/common/trace"

	"github.com/zeromicro/go-zero/core/logx"
)

var GrpcServerRegexp = regexp.MustCompile(`^(?:(?:[a-zA-Z0-9\-_.]+\.)*[a-zA-Z0-9\-_.]+:\d+|direct://[^/]*/(?:[a-zA-Z0-9\-_.]+:\d+(?:,[a-zA-Z0-9\-_.]+:\d+)*)|nacos://(.+)@([a-zA-Z0-9\-_.]+:\d+)(/[^?\s]*)?(?:\?[^#\s]*)?|etcd://\S+|consul://\S+)$`)

type SendProtoTriggerLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendProtoTriggerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendProtoTriggerLogic {
	return &SendProtoTriggerLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendProtoTriggerLogic) SendProtoTrigger(in *trigger.SendProtoTriggerReq) (*trigger.SendProtoTriggerRes, error) {
	traceID := trace.TraceIDFromContext(l.ctx)
	spanCtx, span := asynqx.StartAsynqProducerSpan(l.ctx, asynqx.DeferTriggerProtoTask)
	defer span.End()

	carrier := &propagation.HeaderCarrier{}
	tracex.Inject(spanCtx, carrier)

	match := GrpcServerRegexp.MatchString(in.GrpcServer)
	if !match {
		return nil, errors.New("grpcServer is invalid")
	}

	msg := &taskpayload.GrpcPayload{
		MsgId:          in.MsgId,
		Carrier:        carrier,
		GrpcServer:     in.GrpcServer,
		Method:         in.Method,
		Payload:        string(in.Payload),
		RequestTimeout: in.RequestTimeout,
	}

	opts, payload, err := prepareEnqueue(l.ctx, l.svcCtx, in.MsgId, in.MaxRetry, in.TriggerTime, in.ProcessIn, msg)
	if err != nil {
		return nil, err
	}

	taskInfo, err := l.svcCtx.AsynqClient.Enqueue(asynq.NewTask(asynqx.DeferTriggerProtoTask, payload), opts...)
	if err != nil {
		return nil, err
	}

	return &trigger.SendProtoTriggerRes{
		TraceId: traceID,
		Id:      taskInfo.ID,
		Queue:   taskInfo.Queue,
	}, nil
}
