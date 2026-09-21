package svc

import (
	"context"
	"github.com/hibiken/asynq"
	trace2 "github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
	"zero-service/zerorpc/internal/config"
)

const (
	AsynqTypeKey = attribute.Key("asynq.type")
)

func newAsynqClient(c config.Config) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{Addr: c.Redis.Host, Password: c.Redis.Pass})
}

func StartAsynqProducerSpan(ctx context.Context, typename string) (context.Context, oteltrace.Span) {
	tracer := otel.Tracer(trace2.TraceName)
	ctx, span := tracer.Start(ctx, "asynq-producer", oteltrace.WithSpanKind(oteltrace.SpanKindProducer))
	span.SetAttributes(AsynqTypeKey.String(typename))
	return ctx, span
}
