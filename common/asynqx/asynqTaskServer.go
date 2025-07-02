package asynqx

import (
	"context"
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/timex"
	trace2 "github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type TaskServer struct {
	asynqServer *asynq.Server
	mux         *asynq.ServeMux
}

func NewTaskServer(server *asynq.Server, mux *asynq.ServeMux) *TaskServer {
	return &TaskServer{asynqServer: server, mux: mux}
}

func (q *TaskServer) Start() {
	if err := q.asynqServer.Run(q.mux); err != nil {
		logx.Errorf("asynq taskServer run err:%+v", err)
		panic(err)
	}
}

func (q *TaskServer) Stop() {
	q.asynqServer.Stop()
}

func NewAsynqServer(addr, pass string, db int) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{Addr: addr, Password: pass, DB: db},
		asynq.Config{
			IsFailure: func(err error) bool {
				//logx.Infof("asynq server exec task err:%+v", err)
				return true
			},
			Concurrency: 20, //max concurrent process task task num
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)
}

func StartAsynqConsumerSpan(ctx context.Context, typename string) (context.Context, trace.Span) {
	trace := otel.Tracer(trace2.TraceName)
	ctx, span := trace.Start(ctx, "asynq-cosumer", oteltrace.WithSpanKind(oteltrace.SpanKindConsumer))
	span.SetAttributes(AsynqTypeKey.String(typename))
	return ctx, span
}

func LoggingMiddleware(h asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
		startTime := timex.Now()
		ctx = logx.ContextWithFields(ctx, logx.Field("type", t.Type()), logx.Field("taskId", t.ResultWriter().TaskID()))
		err := h.ProcessTask(ctx, t)
		duration := timex.Since(startTime)
		if err != nil {
			logx.WithContext(ctx).WithDuration(duration).Errorf("asynq processing error %+v", err)
			return err
		}
		logx.WithContext(ctx).WithDuration(duration).Infof("asynq processing success")
		return nil
	})
}
