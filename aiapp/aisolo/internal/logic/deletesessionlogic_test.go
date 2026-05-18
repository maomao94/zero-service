package logic

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/session"
	"zero-service/aiapp/aisolo/internal/svc"
	"zero-service/common/einox/memory"
)

func TestDeleteSessionDeletesMessagesBeforeSession(t *testing.T) {
	ctx := context.Background()
	sessions := session.NewMemoryStore()
	messages := memory.NewMemoryStorage()
	sess := &session.Session{
		ID:        "session-1",
		UserID:    "user-1",
		Status:    aisolo.SessionStatus_SESSION_STATUS_IDLE,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := sessions.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if err := messages.SaveMessage(ctx, &memory.ConversationMessage{UserID: sess.UserID, SessionID: sess.ID, Role: "user", Content: "hello"}); err != nil {
		t.Fatalf("SaveMessage() error = %v", err)
	}

	resp, err := NewDeleteSessionLogic(ctx, &svc.ServiceContext{Sessions: sessions, Messages: messages}).DeleteSession(&aisolo.DeleteSessionReq{
		UserId:    sess.UserID,
		SessionId: sess.ID,
	})
	if err != nil || !resp.Success {
		t.Fatalf("DeleteSession() = (%#v, %v), want success", resp, err)
	}
	if _, err := sessions.GetSession(ctx, sess.UserID, sess.ID); err == nil {
		t.Fatal("session still exists after delete")
	}
	stored, err := messages.GetMessages(ctx, sess.UserID, sess.ID, 0)
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}
	if len(stored) != 0 {
		t.Fatalf("stored messages = %#v, want deleted", stored)
	}
}

func TestDeleteSessionRejectsRunningSession(t *testing.T) {
	ctx := context.Background()
	sessions := session.NewMemoryStore()
	sess := &session.Session{
		ID:        "session-1",
		UserID:    "user-1",
		Status:    aisolo.SessionStatus_SESSION_STATUS_RUNNING,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := sessions.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	resp, err := NewDeleteSessionLogic(ctx, &svc.ServiceContext{Sessions: sessions}).DeleteSession(&aisolo.DeleteSessionReq{
		UserId:    sess.UserID,
		SessionId: sess.ID,
	})
	if err == nil || !strings.Contains(err.Error(), "running session") || resp.Success {
		t.Fatalf("DeleteSession() = (%#v, %v), want running-session rejection", resp, err)
	}
	if _, err := sessions.GetSession(ctx, sess.UserID, sess.ID); err != nil {
		t.Fatalf("session was deleted despite rejection: %v", err)
	}
}

func TestDeleteSessionKeepsSessionWhenMessageDeleteFails(t *testing.T) {
	ctx := context.Background()
	sessions := session.NewMemoryStore()
	sess := &session.Session{
		ID:        "session-1",
		UserID:    "user-1",
		Status:    aisolo.SessionStatus_SESSION_STATUS_IDLE,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := sessions.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	messages := failingDeleteStorage{err: fmt.Errorf("delete messages failed")}
	resp, err := NewDeleteSessionLogic(ctx, &svc.ServiceContext{Sessions: sessions, Messages: messages}).DeleteSession(&aisolo.DeleteSessionReq{
		UserId:    sess.UserID,
		SessionId: sess.ID,
	})
	if err == nil || !strings.Contains(err.Error(), "delete messages failed") || resp.Success {
		t.Fatalf("DeleteSession() = (%#v, %v), want message delete failure", resp, err)
	}
	if _, err := sessions.GetSession(ctx, sess.UserID, sess.ID); err != nil {
		t.Fatalf("session was deleted despite message cleanup failure: %v", err)
	}
}

type failingDeleteStorage struct {
	err error
}

func (s failingDeleteStorage) SaveMessage(context.Context, *memory.ConversationMessage) error {
	return nil
}

func (s failingDeleteStorage) GetMessages(context.Context, string, string, int) ([]*memory.ConversationMessage, error) {
	return nil, nil
}

func (s failingDeleteStorage) DeleteSession(context.Context, string, string) error {
	return s.err
}

func (s failingDeleteStorage) Close() error { return nil }
