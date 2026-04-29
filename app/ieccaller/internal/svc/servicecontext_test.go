package svc

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"zero-service/app/ieccaller/internal/config"
	"zero-service/common/iec104/types"

	"github.com/zeromicro/go-zero/core/logx"
)

func TestASDUPushLogContextFillsUnifiedFields(t *testing.T) {
	var buf bytes.Buffer
	logx.SetWriter(logx.NewWriter(&buf))
	defer logx.Reset()

	msg := &types.MsgBody{
		MsgId:    "msg-1",
		Host:     "127.0.0.1",
		Port:     2404,
		Asdu:     "M_SP_NA_1",
		TypeId:   1,
		DataType: 1,
		Coa:      7,
		Body:     &types.SinglePointInfo{Ioa: 3},
	}

	ctx := asduPushLogContext(context.Background(), msg, 3, "kafka")
	logx.WithContext(ctx).Error("asdu push context test")

	got := buf.String()
	for _, want := range []string{
		"asdu push context test",
		`"msgId":"msg-1"`,
		`"host":"127.0.0.1"`,
		`"port":2404`,
		`"asdu":"M_SP_NA_1"`,
		`"typeId":1`,
		`"dataType":1`,
		`"coa":7`,
		`"ioa":3`,
		`"channel":"kafka"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected log to contain %q, got %q", want, got)
		}
	}
}

func TestPushBroadcastReturnsErrorWhenPusherIsNilInClusterMode(t *testing.T) {
	svcCtx := ServiceContext{
		Config: config.Config{
			DeployMode: "cluster",
		},
	}
	svcCtx.Config.KafkaConfig.BroadcastGroupId = "iec-caller"

	err := svcCtx.PushBroadcast(context.Background(), &types.BroadcastBody{Method: "test"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "kafka broadcast pusher is nil") {
		t.Fatalf("expected kafka broadcast pusher error, got %v", err)
	}
}
