package logic

import (
	"context"
	"regexp"
	"time"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/asynqx"
	"zero-service/common/ctxdata"

	"github.com/dromara/carbon/v2"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/zeromicro/go-zero/core/logx"
)

var GrpcServerRegexp = regexp.MustCompile(`^(?:(?:[a-zA-Z0-9\-_.]+\.)*[a-zA-Z0-9\-_.]+:\d+|direct://[^/]*/(?:[a-zA-Z0-9\-_.]+:\d+(?:,[a-zA-Z0-9\-_.]+:\d+)*)|nacos://(.+)@([a-zA-Z0-9\-_.]+:\d+)(/[^?\s]*)?(?:\?[^#\s]*)?)$`)

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
	otel.GetTextMapPropagator().Inject(spanCtx, carrier)
	matsh := GrpcServerRegexp.MatchString(in.GrpcServer)
	if !matsh {
		return nil, errors.New("grpcServer is invalid")
	}
	msg := &ctxdata.ProtoMsgBody{
		MsgId:          in.MsgId,
		Carrier:        carrier,
		GrpcServer:     in.GrpcServer,
		Method:         in.Method,
		Payload:        string(in.Payload),
		RequestTimeout: in.RequestTimeout,
	}
	opts := []asynq.Option{}
	if len(in.GetMsgId()) == 0 {
		in.MsgId = uuid.NewString()
		msg.MsgId = in.MsgId
	}
	opts = append(opts, asynq.TaskID(in.MsgId))
	payload, err := jsonx.Marshal(msg)
	if err != nil {
		return nil, err
	}
	err = l.svcCtx.Validate.Struct(msg)
	if err != nil {
		return nil, err
	}
	if in.GetMaxRetry() > 0 {
		opts = append(opts, asynq.MaxRetry(int(in.GetMaxRetry())))
	}
	opts = append(opts, asynq.Queue("critical"), asynq.Retention(7*24*time.Hour))
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
	opts = append(opts, asynq.ProcessIn(d))
	taskInfo, err := l.svcCtx.AsynqClient.Enqueue(asynq.NewTask(asynqx.DeferTriggerProtoTask, []byte(payload)), opts...)
	if err != nil {
		return nil, err
	}
	return &trigger.SendProtoTriggerRes{
		TraceId: traceID,
		Id:      taskInfo.ID,
		Queue:   taskInfo.Queue,
	}, nil
}
