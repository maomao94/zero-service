package mqttx

import (
	"context"
	"testing"
)

// asduPayload 模拟 ieccaller 推送的 ASDU 报文（自带 headers），用于衡量跳过注入时的扫描开销。
var asduPayload = []byte(`{"msgId":"01930f2e-7d6a-7c31-9b2a-8f0d1c2b3a4e","host":"192.168.1.100","port":2404,` +
	`"asdu":"M_ME_NC_1","typeId":13,"dataType":6,"coa":1,` +
	`"traceId":"4bf92f3577b34da6a3ce929d0e0e4736",` +
	`"headers":{"traceparent":"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"},` +
	`"metaData":{"stationId":"330KV","arrayId":[1,2,3]},` +
	`"body":{"ioa":16385,"value":12.34,"qds":0,"qdsDesc":"good","ov":false,"bl":false,"sb":false,"nt":false,"iv":false}}`)

// businessPayload 模拟不带 headers 的普通业务 JSON，用于衡量注入（追加字段）开销。
var businessPayload = []byte(`{"deviceId":"1581F5BKD225900D0A1E","sn":"ABC123","time":"2026-11-27 10:00:00.000000",` +
	`"latitude":31.2304,"longitude":121.4737,"altitude":120.5,"battery":86,"status":"online"}`)

func BenchmarkInjectTraceHeadersASDU(b *testing.B) {
	ctx := withTestSpanContext(context.Background())
	b.ReportAllocs()
	for b.Loop() {
		if _, ok := injectTraceHeaders(ctx, asduPayload); ok {
			b.Fatal("expected injection to be skipped for payload with headers")
		}
	}
}

func BenchmarkInjectTraceHeadersBusinessJSON(b *testing.B) {
	ctx := withTestSpanContext(context.Background())
	b.ReportAllocs()
	for b.Loop() {
		if _, ok := injectTraceHeaders(ctx, businessPayload); !ok {
			b.Fatal("expected injection for payload without headers")
		}
	}
}

func BenchmarkInjectTraceHeadersNoTrace(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, ok := injectTraceHeaders(context.Background(), asduPayload); ok {
			b.Fatal("expected no injection without span")
		}
	}
}
