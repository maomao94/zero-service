package task

import (
	"context"
	"encoding/json"
	"zero-service/common/msgbody"
	"zero-service/zerorpc/internal/svc"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel"
)

type DeferDelayTaskHandler struct {
	svcCtx *svc.ServiceContext
}

func NewDeferDelayTask(svcCtx *svc.ServiceContext) *DeferDelayTaskHandler {
	return &DeferDelayTaskHandler{
		svcCtx: svcCtx,
	}
}

func (l *DeferDelayTaskHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var msg msgbody.MsgBody
	if err := json.Unmarshal([]byte(t.Payload()), &msg); err != nil {
		logx.Errorf(" processTask %s 失败 err:%+v", t.Type(), err)
		return err
	} else {
		wireContext := otel.GetTextMapPropagator().Extract(ctx, msg.Carrier)
		_, span := svc.StartAsynqConsumerSpan(ctx, t.Type())
		defer span.End()
		// todo: do something
		logx.WithContext(wireContext).Infof("do something")
	}
	return nil
}
