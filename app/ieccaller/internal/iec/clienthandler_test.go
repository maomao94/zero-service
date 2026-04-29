package iec

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"zero-service/app/ieccaller/internal/config"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/common/iec104/client"
	"zero-service/common/iec104/types"

	"github.com/wendy512/go-iecp5/asdu"
	"github.com/zeromicro/go-zero/core/logx"
)

func TestClientCallASDULogContextFillsUnifiedFields(t *testing.T) {
	var buf bytes.Buffer
	logx.SetWriter(logx.NewWriter(&buf))
	defer logx.Reset()

	clientCall := NewClientCall(&svc.ServiceContext{}, config.IecServerConfig{
		ClientConfig: client.ClientConfig{
			Host: "127.0.0.1",
			Port: 2404,
			MetaData: map[string]any{
				"stationId": "station-1",
			},
		},
	})
	packet := &asdu.ASDU{
		Identifier: asdu.Identifier{
			Type:       asdu.M_SP_NA_1,
			CommonAddr: 7,
		},
	}

	ctx := clientCall.asduLogContext(context.Background(), packet)
	logx.WithContext(ctx).Error("asdu log context test")

	got := buf.String()
	for _, want := range []string{
		"asdu log context test",
		`"host":"127.0.0.1"`,
		`"port":2404`,
		`"stationId":"station-1"`,
		`"asdu":"M_SP_NA_1"`,
		`"typeId":1`,
		`"dataType":0`,
		`"coa":7`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected log to contain %q, got %q", want, got)
		}
	}
}

func TestClientCallNewMsgBodyFillsCommonFields(t *testing.T) {
	meta := map[string]any{"stationId": "station-1"}
	clientCall := NewClientCall(&svc.ServiceContext{}, config.IecServerConfig{
		ClientConfig: client.ClientConfig{
			Host:     "127.0.0.1",
			Port:     2404,
			MetaData: meta,
		},
	})
	packet := &asdu.ASDU{
		Identifier: asdu.Identifier{
			Type:       asdu.M_SP_NA_1,
			CommonAddr: 7,
		},
	}
	body := &types.SinglePointInfo{Ioa: 1}

	got := clientCall.newMsgBody(packet, "msg-1", packet.CommonAddr, body)

	if got.MsgId != "msg-1" {
		t.Fatalf("expected msgId msg-1, got %s", got.MsgId)
	}
	if got.Host != "127.0.0.1" {
		t.Fatalf("expected host 127.0.0.1, got %s", got.Host)
	}
	if got.Port != 2404 {
		t.Fatalf("expected port 2404, got %d", got.Port)
	}
	if got.Asdu != genASDUName(packet.Type) {
		t.Fatalf("expected asdu %s, got %s", genASDUName(packet.Type), got.Asdu)
	}
	if got.TypeId != int(packet.Type) {
		t.Fatalf("expected typeId %d, got %d", int(packet.Type), got.TypeId)
	}
	if got.DataType != int(client.GetDataType(packet.Type)) {
		t.Fatalf("expected dataType %d, got %d", int(client.GetDataType(packet.Type)), got.DataType)
	}
	if got.Coa != uint(packet.CommonAddr) {
		t.Fatalf("expected coa %d, got %d", packet.CommonAddr, got.Coa)
	}
	if got.Body != body {
		t.Fatalf("expected body pointer to be preserved")
	}
	if got.MetaData["stationId"] != "station-1" {
		t.Fatalf("expected metadata stationId station-1, got %v", got.MetaData["stationId"])
	}
}

func TestClientCallPushASDUKeepsNonBlockingSemantics(t *testing.T) {
	clientCall := NewClientCall(&svc.ServiceContext{}, config.IecServerConfig{})

	err := clientCall.pushASDU(context.Background(), &types.MsgBody{
		MsgId: "msg-1",
		Host:  "127.0.0.1",
		Port:  2404,
		Body:  &types.SinglePointInfo{Ioa: 1},
	}, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
