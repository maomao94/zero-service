package iec104

import (
	"context"
	"encoding/json"

	tracex "zero-service/common/trace"

	"github.com/wendy512/go-iecp5/asdu"
	ztrace "github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer(ztrace.TraceName)

type FrameTraceOptions struct {
	Host      string
	Port      int
	StationId string
}

// StartRecvSpan 从站推送帧到主站客户端，spanKind=Consumer。
func StartRecvSpan(ctx context.Context, packet *asdu.ASDU, opts FrameTraceOptions) (context.Context, oteltrace.Span) {
	ctx, span := tracer.Start(ctx, "iec104-recv-frame", oteltrace.WithSpanKind(oteltrace.SpanKindConsumer))
	span.SetAttributes(
		attribute.String("iec.station_id", opts.StationId),
		attribute.String("iec.host", opts.Host),
		attribute.Int("iec.port", opts.Port),
		attribute.Int("iec.type_id", int(packet.Type)),
		attribute.Int("iec.coa", int(packet.CommonAddr)),
	)
	return ctx, span
}

// StartForwardSpan 主站转发到下游客端，spanKind=Producer。
func StartForwardSpan(ctx context.Context) (context.Context, oteltrace.Span) {
	return tracer.Start(ctx, "iec104-forward-chunk", oteltrace.WithSpanKind(oteltrace.SpanKindProducer))
}

// TraceHeaders 从 ctx 提取 OTel propagation headers 和 traceId。
func TraceHeaders(ctx context.Context) (map[string]string, string) {
	headers := make(map[string]string)
	tracex.Inject(ctx, tracex.NewCarrier(headers))
	return headers, TraceIdFromContext(ctx)
}

func TraceIdFromContext(ctx context.Context) string {
	return ztrace.TraceIDFromContext(ctx)
}

// ExtractTraceHeaders 从 JSON payload 顶层 "headers" 字段恢复 OTel propagation context。
func ExtractTraceHeaders(ctx context.Context, payload string) context.Context {
	var obj struct {
		Headers map[string]string `json:"headers"`
	}
	if err := json.Unmarshal([]byte(payload), &obj); err != nil || len(obj.Headers) == 0 {
		return ctx
	}
	return tracex.Extract(ctx, tracex.NewCarrier(obj.Headers))
}
