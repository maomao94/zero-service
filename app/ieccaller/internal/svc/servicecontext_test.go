package svc

import (
	"context"
	"errors"
	"testing"

	"zero-service/common/iec104/types"
	"zero-service/common/mqttx"
)

func TestDecodeBroadcastAck(t *testing.T) {
	router := mqttx.NewReplyRouter(mqttx.ReplyDecoderFunc[*types.BroadcastAckBody](decodeBroadcastAck))

	resolved, err := router.HandleReply(context.Background(), []byte(`{"tId":"tid-1","method":"method","success":true,"responseBody":"{}"}`), "reply/topic", "reply/+")
	if err != nil {
		t.Fatalf("HandleReply returned error: %v", err)
	}
	if resolved {
		t.Fatal("expected no pending request to resolve")
	}
}

func TestDecodeBroadcastAckRejectsInvalidPayload(t *testing.T) {
	router := mqttx.NewReplyRouter(mqttx.ReplyDecoderFunc[*types.BroadcastAckBody](decodeBroadcastAck))

	_, err := router.HandleReply(context.Background(), []byte(`{`), "reply/topic", "reply/+")
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestDecodeBroadcastAckRejectsEmptyTid(t *testing.T) {
	router := mqttx.NewReplyRouter(mqttx.ReplyDecoderFunc[*types.BroadcastAckBody](decodeBroadcastAck))

	_, err := router.HandleReply(context.Background(), []byte(`{"method":"method","success":true,"responseBody":"{}"}`), "reply/topic", "reply/+")
	if !errors.Is(err, mqttx.ErrEmptyReplyTid) {
		t.Fatalf("expected ErrEmptyReplyTid, got %v", err)
	}
}
