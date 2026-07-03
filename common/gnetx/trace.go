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

func startServerSpan(tracer oteltrace.Tracer, s *session, msg any) (context.Context, oteltrace.Span) {
	return startSpanWithKind(tracer, s, msg, "gnetx-server", oteltrace.SpanKindServer)
}

func startClientSpan(tracer oteltrace.Tracer, s *session, msg any) (context.Context, oteltrace.Span) {
	return startSpanWithKind(tracer, s, msg, "gnetx-client", oteltrace.SpanKindClient)
}

func startSpanWithKind(tracer oteltrace.Tracer, s *session, msg any, name string, kind oteltrace.SpanKind) (context.Context, oteltrace.Span) {
	attrs := spanAttrs(s, msg)
	return tracer.Start(context.Background(), name,
		oteltrace.WithSpanKind(kind),
		oteltrace.WithAttributes(attrs...),
	)
}

func spanAttrs(s *session, msg any) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("gnetx.session.id", s.ID()),
		attribute.String("gnetx.remote", s.RemoteAddr().String()),
	}
	if a := s.Alias(); a != "" {
		attrs = append(attrs, attribute.String("gnetx.session.alias", a))
	}
	if id, ok := messageIDOf(msg); ok {
		attrs = append(attrs, attribute.Int("gnetx.message.id", id))
	}
	if r, ok := msg.(Response); ok {
		attrs = append(attrs, attribute.String("gnetx.response.tid", r.ResponseTID()))
	}
	return attrs
}
