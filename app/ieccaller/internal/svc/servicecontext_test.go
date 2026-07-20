package svc

import (
	"context"
	"errors"
	"testing"

	"zero-service/common/iec104"
	"zero-service/common/iec104/types"
	"zero-service/common/mqttx"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestDecodeBroadcastAck(t *testing.T) {
	decoder := mqttx.ReplyDecoderFunc[*types.BroadcastAckBody](decodeBroadcastAck)

	msg, err := decoder.Decode(context.Background(), []byte(`{"tId":"tid-1","method":"method","success":true,"responseBody":"{}"}`), "reply/topic", "reply/+")
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if msg.Tid != "tid-1" || msg.Value == nil || !msg.Value.Success {
		t.Fatalf("unexpected decoded message: %+v", msg)
	}
}

func TestDecodeBroadcastAckRejectsInvalidPayload(t *testing.T) {
	decoder := mqttx.ReplyDecoderFunc[*types.BroadcastAckBody](decodeBroadcastAck)

	_, err := decoder.Decode(context.Background(), []byte(`{`), "reply/topic", "reply/+")
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestDecodeBroadcastAckRejectsEmptyTid(t *testing.T) {
	decoder := mqttx.ReplyDecoderFunc[*types.BroadcastAckBody](decodeBroadcastAck)

	_, err := decoder.Decode(context.Background(), []byte(`{"method":"method","success":true,"responseBody":"{}"}`), "reply/topic", "reply/+")
	if !errors.Is(err, mqttx.ErrEmptyReplyTid) {
		t.Fatalf("expected ErrEmptyReplyTid, got %v", err)
	}
}

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
