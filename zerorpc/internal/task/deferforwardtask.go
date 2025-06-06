package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dromara/carbon/v2"
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/netx"
	"github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	"net/http"
	"time"
	"zero-service/common/ctxdata"
	"zero-service/zeroalarm/zeroalarm"
	"zero-service/zerorpc/internal/svc"
)

type DeferForwardTaskHandler struct {
	svcCtx *svc.ServiceContext
}

func NewDeferForwardTask(svcCtx *svc.ServiceContext) *DeferForwardTaskHandler {
	return &DeferForwardTaskHandler{
		svcCtx: svcCtx,
	}
}

func (l *DeferForwardTaskHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var msg ctxdata.MsgBody
	if err := json.Unmarshal([]byte(t.Payload()), &msg); err != nil {
		return err
	} else {
		ctx = otel.GetTextMapPropagator().Extract(ctx, msg.Carrier)
		traceID := trace.TraceIDFromContext(ctx)
		ctx, span := svc.StartAsynqConsumerSpan(ctx, t.Type())
		defer span.End()
		if msg.Url != "" {
			type Data struct {
				MsgId string `json:"msgId"`
				Body  string `json:"body"`
			}
			var data = Data{
				MsgId: msg.MsgId,
				Body:  msg.Msg,
			}
			postCtx, _ := context.WithTimeout(ctx, time.Duration(5)*time.Second)
			resp, err := l.svcCtx.Httpc.Do(postCtx, http.MethodPost, msg.Url, data)
			if err != nil {
				_, alarmErr := l.svcCtx.ZeroAlarmCli.Alarm(ctx, &zeroalarm.AlarmReq{
					ChatName:    "服务告警",
					Description: "服务告警",
					Title:       "服务告警 - Zero-Service",
					Project:     "zero.rpc",
					DateTime:    carbon.Now().Format("Y-m-d H:i:s"),
					AlarmId:     msg.MsgId,
					Content:     fmt.Sprintf("%s,转发任务执行失败", traceID),
					Error:       fmt.Sprintf("processTask-%s, err:%+v, msgId:%s", t.Type(), err.Error(), msg.MsgId),
					Ip:          netx.InternalIp(),
				})
				if alarmErr != nil {
					return alarmErr
				}
				t.ResultWriter().Write([]byte("fail"))
				return err
			}
			if resp.StatusCode == http.StatusOK {
				t.ResultWriter().Write([]byte("success"))
			} else {
				t.ResultWriter().Write([]byte("fail"))
				_, alarmErr := l.svcCtx.ZeroAlarmCli.Alarm(ctx, &zeroalarm.AlarmReq{
					ChatName:    "服务告警",
					Description: "服务告警",
					Title:       "服务告警 - Zero-Service",
					Project:     "zero.rpc",
					DateTime:    carbon.Now().Format("Y-m-d H:i:s"),
					AlarmId:     msg.MsgId,
					Content:     fmt.Sprintf("%s,转发任务执行失败", traceID),
					Error:       fmt.Sprintf(fmt.Sprintf("processTask-%s, code:%d, url:%s", t.Type(), resp.StatusCode, msg.Url)),
					Ip:          netx.InternalIp(),
				})
				if alarmErr != nil {
					return alarmErr
				}
				return errors.New("forwardTask fail")
			}
		} else {
			t.ResultWriter().Write([]byte("fail"))
		}
	}
	return nil
}
