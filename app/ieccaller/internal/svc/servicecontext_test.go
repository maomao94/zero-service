package svc

import (
	"context"
	"testing"

	"zero-service/common/iec104"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

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

	headers, traceId := iec104.TraceHeaders(ctx)

	if headers["traceparent"] != "00-0102030405060708090a0b0c0d0e0f10-1112131415161718-01" {
		t.Fatalf("unexpected traceparent: %q", headers["traceparent"])
	}
	if traceId != "0102030405060708090a0b0c0d0e0f10" {
		t.Fatalf("unexpected traceId: %q", traceId)
	}
}
