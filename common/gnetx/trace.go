package gnetx

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	ztrace "github.com/zeromicro/go-zero/core/trace"
)

func gnetxTracer() oteltrace.Tracer {
	return otel.Tracer(ztrace.TraceName)
}

func startServerSpan(tracer oteltrace.Tracer, parentCtx context.Context, s *session, msg any) (context.Context, oteltrace.Span) {
	return startSpanWithKind(tracer, parentCtx, s, msg, "gnetx-server", oteltrace.SpanKindInternal)
}

func startClientSpan(tracer oteltrace.Tracer, parentCtx context.Context, s *session, msg any) (context.Context, oteltrace.Span) {
	return startSpanWithKind(tracer, parentCtx, s, msg, "gnetx-client", oteltrace.SpanKindInternal)
}

func startSpanWithKind(tracer oteltrace.Tracer, parentCtx context.Context, s *session, msg any, name string, kind oteltrace.SpanKind) (context.Context, oteltrace.Span) {
	attrs := spanAttrs(s, msg)
	return tracer.Start(parentCtx, name,
		oteltrace.WithSpanKind(kind),
		oteltrace.WithAttributes(attrs...),
	)
}

func spanAttrs(s *session, msg any) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("gnetx.session.id", s.SessionID()),
		attribute.String("gnetx.remote", s.RemoteAddr().String()),
	}
	if clientID := s.ClientID(); clientID != "" {
		attrs = append(attrs, attribute.String("gnetx.client.id", clientID))
	}
	if id, ok := messageIDOf(msg); ok {
		attrs = append(attrs, attribute.Int("gnetx.message.id", id))
	}
	if r, ok := msg.(Response); ok {
		attrs = append(attrs, attribute.String("gnetx.response.tid", r.ResponseTID()))
	}
	return attrs
}
