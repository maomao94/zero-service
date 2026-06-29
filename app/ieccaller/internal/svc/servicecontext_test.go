package svc

import (
	"context"
	"errors"
	"testing"

	"zero-service/common/iec104/types"
	"zero-service/common/mqttx"
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
