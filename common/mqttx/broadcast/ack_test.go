package broadcast

import (
	"context"
	"errors"
	"testing"
	"time"

	"zero-service/common/mqttx"
)

func TestAckReplyRouterConsumeDecodesValidPayload(t *testing.T) {
	router := NewAckReplyRouter(time.Second, "test-router")
	defer router.Close()

	// 解码成功但没有 pending 条目 → ErrReplyNotMatched（证明解码链路 OK）
	err := router.Consume(context.Background(), []byte(`{"tId":"tid-1","method":"m","success":true,"responseBody":"{}"}`), "reply/topic", "reply/+")
	if !errors.Is(err, mqttx.ErrReplyNotMatched) {
		t.Fatalf("expected ErrReplyNotMatched, got %v", err)
	}
}

func TestAckReplyRouterConsumeRejectsInvalidPayload(t *testing.T) {
	router := NewAckReplyRouter(time.Second, "test-router")
	defer router.Close()

	err := router.Consume(context.Background(), []byte(`{`), "reply/topic", "reply/+")
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}
	if errors.Is(err, mqttx.ErrReplyNotMatched) {
		t.Fatalf("invalid payload must not be treated as not-matched: %v", err)
	}
}

func TestAckReplyRouterConsumeRejectsEmptyTid(t *testing.T) {
	router := NewAckReplyRouter(time.Second, "test-router")
	defer router.Close()

	err := router.Consume(context.Background(), []byte(`{"method":"m","success":true}`), "reply/topic", "reply/+")
	if !errors.Is(err, mqttx.ErrEmptyReplyTid) {
		t.Fatalf("expected ErrEmptyReplyTid, got %v", err)
	}
}

func TestNewAckReplyRouterDefaults(t *testing.T) {
	router := NewAckReplyRouter(0, "")
	defer router.Close()

	err := router.Consume(context.Background(), []byte(`{"tId":"t","success":true}`), "reply/topic", "reply/+")
	if !errors.Is(err, mqttx.ErrReplyNotMatched) {
		t.Fatalf("default router should decode and report not-matched, got %v", err)
	}
}
