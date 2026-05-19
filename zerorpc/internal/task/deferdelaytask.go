package task

import (
	"context"
	"encoding/json"
	"zero-service/common/trace"
	"zero-service/zerorpc/internal/svc"
	"zero-service/zerorpc/internal/taskpayload"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
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
	var msg taskpayload.HttpPayload
	if err := json.Unmarshal([]byte(t.Payload()), &msg); err != nil {
		logx.Errorf(" processTask %s 失败 err:%+v", t.Type(), err)
		return err
	} else {
		wireContext := trace.Extract(ctx, msg.Carrier)
		_, span := svc.StartAsynqConsumerSpan(wireContext, t.Type())
		defer span.End()
		// todo: do something
		logx.WithContext(wireContext).Infof("do something")
	}
	return nil
}
