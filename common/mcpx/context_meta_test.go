package mcpx

import (
	"context"
	"reflect"
	"testing"

	"zero-service/common/authctx"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestMetaCollectExtractAndRawStorage(t *testing.T) {
	if ctxMetaKey != "_meta" {
		t.Fatalf("ctxMetaKey = %q", ctxMetaKey)
	}
	ctx := context.Background()
	ctx = authctx.WithAuthorization(ctx, "Bearer token")
	ctx = authctx.WithUserID(ctx, "user-1")
	ctx = authctx.WithUserName(ctx, "")
	ctx = context.WithValue(ctx, string("dept-code"), float64(3))

	meta := CollectFromCtx(ctx)
	want := map[string]any{"authorization": "Bearer token", "user-id": "user-1"}
	if !reflect.DeepEqual(meta, want) {
		t.Fatalf("CollectFromCtx() = %#v, want %#v", meta, want)
	}
	if got := CollectFromCtx(context.Background()); got != nil {
		t.Fatalf("empty CollectFromCtx() = %#v", got)
	}

	meta[authctx.CtxDeptCodeKey] = float64(12)
	extracted := ExtractFromMeta(context.Background(), meta)
	if got := authctx.GetAuthorization(extracted); got != "Bearer token" {
		t.Fatalf("authorization = %q", got)
	}
	if got := authctx.GetDeptCode(extracted); got != "12" {
		t.Fatalf("numeric dept code = %q", got)
	}

	stored := WithMeta(context.Background(), meta)
	if got := GetMeta(stored); !reflect.DeepEqual(got, meta) {
		t.Fatalf("GetMeta() = %#v", got)
	}
	if got := stored.Value(string("_meta")); !reflect.DeepEqual(got, meta) {
		t.Fatalf("raw string _meta lookup = %#v", got)
	}
	externalStored := context.WithValue(context.Background(), string("_meta"), meta)
	if got := GetMeta(externalStored); !reflect.DeepEqual(got, meta) {
		t.Fatalf("GetMeta() from external string key = %#v", got)
	}
	if got := GetMeta(context.Background()); got != nil {
		t.Fatalf("empty GetMeta() = %#v", got)
	}
	if got := ExtractFromMeta(context.Background(), nil); got != context.Background() {
		t.Fatal("nil ExtractFromMeta changed the context")
	}
}

func TestExtractFromMetaConvertsClaim(t *testing.T) {
	meta := map[string]any{authctx.CtxDeptCodeKey: float64(1.5)}
	ctx := ExtractFromMeta(context.Background(), meta)
	if got := authctx.GetDeptCode(ctx); got != "1.5" {
		t.Fatalf("dept code = %q, want 1.5", got)
	}
	meta = map[string]any{authctx.CtxUserIdKey: true}
	ctx = ExtractFromMeta(context.Background(), meta)
	if got := authctx.GetUserId(ctx); got != "" {
		t.Fatalf("bool user id = %q, want empty", got)
	}
}

func TestExtractTraceFromMeta(t *testing.T) {
	previous := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(previous) })

	const traceparent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	ctx := ExtractTraceFromMeta(context.Background(), map[string]any{"traceparent": traceparent})
	spanContext := oteltrace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() || spanContext.TraceID().String() != "4bf92f3577b34da6a3ce929d0e0e4736" || spanContext.SpanID().String() != "00f067aa0ba902b7" {
		t.Fatalf("unexpected span context: %v", spanContext)
	}
}
