package task

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/hibiken/asynq"
	"go.opentelemetry.io/otel"
	"net/http"
	"time"
	"zero-service/app/trigger/internal/svc"
	"zero-service/common/asynqx"
	"zero-service/common/ctxdata"
)

type DeferTriggerTaskHandler struct {
	svcCtx *svc.ServiceContext
}

func NewDeferTriggerTask(svcCtx *svc.ServiceContext) *DeferTriggerTaskHandler {
	return &DeferTriggerTaskHandler{
		svcCtx: svcCtx,
	}
}

func (l *DeferTriggerTaskHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var msg ctxdata.MsgBody
	if err := json.Unmarshal([]byte(t.Payload()), &msg); err != nil {
		return err
	} else {
		ctx = otel.GetTextMapPropagator().Extract(ctx, msg.Carrier)
		ctx, span := asynqx.StartAsynqConsumerSpan(ctx, t.Type())
		defer span.End()
		if msg.Url != "" {
			type Data struct {
				MsgId string `json:"msgId"`
				Msg   string `json:"Msg"`
			}
			if len(msg.MsgId) == 0 {
				msg.Msg = t.ResultWriter().TaskID()
			}
			var data = Data{
				MsgId: msg.MsgId,
				Msg:   msg.Msg,
			}
			postCtx, _ := context.WithTimeout(ctx, time.Duration(10)*time.Second)
			resp, err := l.svcCtx.Httpc.Do(postCtx, http.MethodPost, msg.Url, data)
			if err != nil {
				t.ResultWriter().Write([]byte("fail,http"))
				return err
			}
			if resp.StatusCode == http.StatusOK {
				t.ResultWriter().Write([]byte("success"))
			} else {
				t.ResultWriter().Write([]byte("fail,httpCode," + resp.Status))
				return errors.New("trigger fail")
			}
		} else {
			t.ResultWriter().Write([]byte("fail,url,empty"))
			return errors.New("trigger fail")
		}
	}
	return nil
}
