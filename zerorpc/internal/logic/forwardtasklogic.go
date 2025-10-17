package logic

import (
	"context"
	"errors"
	"fmt"
	"time"
	"zero-service/app/alarm/alarm"
	"zero-service/common/asynqx"
	"zero-service/common/ctxdata"
	"zero-service/zerorpc/internal/svc"
	"zero-service/zerorpc/zerorpc"

	"github.com/dromara/carbon/v2"
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/netx"
	"github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/zeromicro/go-zero/core/logx"
)

type ForwardTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewForwardTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ForwardTaskLogic {
	return &ForwardTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 转发任务
func (l *ForwardTaskLogic) ForwardTask(in *zerorpc.ForwardTaskReq) (*zerorpc.ForwardTaskRes, error) {
	traceID := trace.TraceIDFromContext(l.ctx)
	spanCtx, span := svc.StartAsynqProducerSpan(l.ctx, asynqx.DeferTriggerTask)
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
	_, err = l.svcCtx.AsynqClient.Enqueue(asynq.NewTask(asynqx.DeferTriggerTask, []byte(payload)), asynq.Queue("critical"), asynq.TaskID(in.GetMsgId()), asynq.ProcessIn(d), asynq.Retention(24*time.Hour))
	if err != nil {
		_, alarmErr := l.svcCtx.AlarmCli.Alarm(l.ctx, &alarm.AlarmReq{
			ChatName:    "服务告警",
			Description: "服务告警",
			Title:       "服务告警 - Zero-Service",
			Project:     "zero.rpc",
			DateTime:    carbon.Now().Format("Y-m-d H:i:s"),
			AlarmId:     in.MsgId,
			Content:     fmt.Sprintf("%s, 转发任务下发失败, msg:%s, url:%s", traceID, msg.Msg, msg.Url),
			Error:       fmt.Sprintf("err:%+v", err),
			Ip:          netx.InternalIp(),
		})
		if alarmErr != nil {
			return nil, alarmErr
		}
		return nil, err
	}
	return &zerorpc.ForwardTaskRes{}, nil
}
