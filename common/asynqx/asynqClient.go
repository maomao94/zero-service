package asynqx

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
	trace2 "github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	AsynqTypeKey = attribute.Key("asynq.type")
)

func NewAsynqClient(addr, pass string, db int) *asynq.Client {
	logx.Infow("[asynq] client creating", logx.Field("addr", addr), logx.Field("db", db))
	return asynq.NewClient(asynq.RedisClientOpt{Addr: addr, Password: pass, DB: db})
}

func NewAsynqInspector(addr, pass string, db int) *asynq.Inspector {
	logx.Infow("[asynq] inspector creating", logx.Field("addr", addr), logx.Field("db", db))
	return asynq.NewInspector(asynq.RedisClientOpt{Addr: addr, Password: pass, DB: db})
}

func StartAsynqProducerSpan(ctx context.Context, typename string) (context.Context, trace.Span) {
	trace := otel.Tracer(trace2.TraceName)
	ctx, span := trace.Start(ctx, "asynq-producer", oteltrace.WithSpanKind(oteltrace.SpanKindProducer))
	span.SetAttributes(AsynqTypeKey.String(typename))
	return ctx, span
}

// LogEnqueue 记录入队结果
func LogEnqueue(typename string, info *asynq.TaskInfo, err error) {
	if err != nil {
		logx.Errorw("[asynq] enqueue failed",
			logx.Field("type", typename),
			logx.Field("err", err),
		)
		return
	}
	logx.Infow("[asynq] enqueue success",
		logx.Field("type", typename),
		logx.Field("taskId", info.ID),
		logx.Field("queue", info.Queue),
	)
}

// FormatPayloadSize 格式化 payload 大小
func FormatPayloadSize(payload []byte) string {
	size := len(payload)
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}
	return fmt.Sprintf("%.2fKB", float64(size)/1024)
}
