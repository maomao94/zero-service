package task

import (
	"context"
	"encoding/json"
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel"
	"zero-service/app/trigger/internal/svc"
	"zero-service/common/asynqx"
	"zero-service/common/ctxdata"
)

type DeferTriggerProtoTaskHandler struct {
	svcCtx *svc.ServiceContext
}

func NewDeferTriggerProtoTask(svcCtx *svc.ServiceContext) *DeferTriggerProtoTaskHandler {
	return &DeferTriggerProtoTaskHandler{
		svcCtx: svcCtx,
	}
}

func (l *DeferTriggerProtoTaskHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var msg ctxdata.ProtoMsgBody
	if err := json.Unmarshal([]byte(t.Payload()), &msg); err != nil {
		return err
	} else {
		ctx = otel.GetTextMapPropagator().Extract(ctx, msg.Carrier)
		ctx, span := asynqx.StartAsynqConsumerSpan(ctx, t.Type())
		defer span.End()
		logx.WithContext(ctx).Debugf("defer protoTrigger task,msg:%s", msg)
	}
	return nil
}
