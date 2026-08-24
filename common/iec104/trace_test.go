package iec104

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// TestTraceHeadersAddsTraceparentAndTraceId 校验 TraceHeaders 从 ctx 提取 OTel 传播头与 traceId
// （原 app/ieccaller/internal/svc/servicecontext_test.go 中用例，随 mqtt broadcast 解码器迁移到
// common/mqttx/broadcast 后保留于此，避免 common/iec104 传播契约失去测试覆盖）。
func TestTraceHeadersAddsTraceparentAndTraceId(t *testing.T) {
	oldPropagator := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTextMapPropagator(oldPropagator)
	})

	traceID := oteltrace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	spanID := oteltrace.SpanID{17, 18, 19, 20, 21, 22, 23, 24}
	ctx := oteltrace.ContextWithSpanContext(context.Background(), oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: oteltrace.FlagsSampled,
	}))

	headers, traceId := TraceHeaders(ctx)

	if headers["traceparent"] != "00-0102030405060708090a0b0c0d0e0f10-1112131415161718-01" {
		t.Fatalf("unexpected traceparent: %q", headers["traceparent"])
	}
	if traceId != "0102030405060708090a0b0c0d0e0f10" {
		t.Fatalf("unexpected traceId: %q", traceId)
	}
}
