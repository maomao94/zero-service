package asynqx

import (
	"context"
	"github.com/hibiken/asynq"
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
	return asynq.NewClient(asynq.RedisClientOpt{Addr: addr, Password: pass, DB: db})
}

func NewAsynqInspector(addr, pass string, db int) *asynq.Inspector {
	return asynq.NewInspector(asynq.RedisClientOpt{Addr: addr, Password: pass, DB: db})
}

func StartAsynqProducerSpan(ctx context.Context, typename string) (context.Context, trace.Span) {
	trace := otel.Tracer(trace2.TraceName)
	ctx, span := trace.Start(ctx, "asynq-producer", oteltrace.WithSpanKind(oteltrace.SpanKindProducer))
	span.SetAttributes(AsynqTypeKey.String(typename))
	return ctx, span
}
