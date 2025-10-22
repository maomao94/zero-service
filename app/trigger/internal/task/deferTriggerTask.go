package task

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
	"zero-service/app/trigger/internal/svc"
	"zero-service/common/asynqx"
	"zero-service/common/ctxdata"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/core/timex"
	"go.opentelemetry.io/otel"
)

type DeferTriggerTaskHandler struct {
	svcCtx  *svc.ServiceContext
	metrics *stat.Metrics
}

func NewDeferTriggerTask(svcCtx *svc.ServiceContext) *DeferTriggerTaskHandler {
	return &DeferTriggerTaskHandler{
		svcCtx:  svcCtx,
		metrics: stat.NewMetrics("http-task"),
	}
}

func (l *DeferTriggerTaskHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	startTime := timex.Now()
	defer l.metrics.Add(stat.Task{
		Duration: timex.Since(startTime),
	})
	var msg ctxdata.MsgBody
	if err := json.Unmarshal([]byte(t.Payload()), &msg); err != nil {
		return err
	} else {
		ctx = otel.GetTextMapPropagator().Extract(ctx, msg.Carrier)
		ctx, span := asynqx.StartAsynqConsumerSpan(ctx, t.Type())
		defer span.End()
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
		logx.WithContext(ctx).Infof("http invoke - %s", msg.Url)
		if err != nil {
			t.ResultWriter().Write([]byte("fail,http error"))
			return err
		}
		if resp.StatusCode == http.StatusOK {
			t.ResultWriter().Write([]byte("success"))
		} else {
			t.ResultWriter().Write([]byte("fail,httpCode error: " + resp.Status))
			return errors.New("trigger fail")
		}

	}
	return nil
}
