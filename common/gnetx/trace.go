package gnetx

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	ztrace "github.com/zeromicro/go-zero/core/trace"
)

// gnetxTracer 返回 gnetx 使用的 OTel Tracer（全局 provider 下按 go-zero TraceName 命名）。
// otel.Tracer 永不返回 nil（未配置 OTel 时返回 noop tracer，Start 出的 span 为 noop，零开销）。
// Server/Client 在构造时缓存一次结果，避免每报文都走全局 provider 的 map+mutex 查找。
func gnetxTracer() oteltrace.Tracer {
	return otel.Tracer(ztrace.TraceName)
}

// startServerSpan 为 server 端每条入站报文创建 SpanKindServer span。
// server 侧作为"服务端处理入站请求"，span 名 gnetx-server。
func startServerSpan(tracer oteltrace.Tracer, sess *Session, msg any) (context.Context, oteltrace.Span) {
	return startSpanWithKind(tracer, sess, msg, "gnetx-server", oteltrace.SpanKindServer)
}

// startClientSpan 为 client 端每条入站报文创建 SpanKindClient span。
// client 侧作为"客户端接收服务端回包/推送"，span 名 gnetx-client。
func startClientSpan(tracer oteltrace.Tracer, sess *Session, msg any) (context.Context, oteltrace.Span) {
	return startSpanWithKind(tracer, sess, msg, "gnetx-client", oteltrace.SpanKindClient)
}

func startSpanWithKind(tracer oteltrace.Tracer, sess *Session, msg any, name string, kind oteltrace.SpanKind) (context.Context, oteltrace.Span) {
	attrs := spanAttrs(sess, msg)
	return tracer.Start(context.Background(), name,
		oteltrace.WithSpanKind(kind),
		oteltrace.WithAttributes(attrs...),
	)
}

func spanAttrs(sess *Session, msg any) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("gnetx.session.id", sess.ID()),
		attribute.String("gnetx.remote", sess.RemoteAddr().String()),
	}
	if s := sess.Alias(); s != "" {
		attrs = append(attrs, attribute.String("gnetx.session.alias", s))
	}
	if id, ok := messageIDOf(msg); ok {
		attrs = append(attrs, attribute.Int("gnetx.message.id", id))
	}
	if r, ok := msg.(Response); ok {
		attrs = append(attrs, attribute.String("gnetx.response.tid", r.ResponseTID()))
	}
	return attrs
}
