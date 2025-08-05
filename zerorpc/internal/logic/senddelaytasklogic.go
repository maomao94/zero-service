package logic

import (
	"context"
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/jsonx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"time"
	"zero-service/common/asynqx"
	"zero-service/common/ctxdata"
	"zero-service/zerorpc/internal/svc"
	"zero-service/zerorpc/zerorpc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendDelayTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendDelayTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendDelayTaskLogic {
	return &SendDelayTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 发送延迟任务
func (l *SendDelayTaskLogic) SendDelayTask(in *zerorpc.SendDelayTaskReq) (*zerorpc.SendDelayTaskRes, error) {
	spanCtx, span := svc.StartAsynqProducerSpan(l.ctx, asynqx.DeferDelayTask)
	defer span.End()
	carrier := &propagation.HeaderCarrier{}
	otel.GetTextMapPropagator().Inject(spanCtx, carrier)
	msg := &ctxdata.MsgBody{
		MsgId:   in.MsgId,
		Carrier: carrier,
		Msg:     in.Body,
	}
	payload, err := jsonx.Marshal(msg)
	if err != nil {
		return nil, err
	}
	_, err = l.svcCtx.AsynqClient.Enqueue(asynq.NewTask(asynqx.DeferDelayTask, []byte(payload)), asynq.TaskID(in.GetMsgId()), asynq.ProcessIn(time.Duration(in.ProcessIn)*time.Minute), asynq.Retention(7*24*time.Hour))
	if err != nil {
		return nil, err
	}
	return &zerorpc.SendDelayTaskRes{}, nil
}
