package logic

import (
	"context"
	"strings"
	"testing"
	"time"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/session"
	"zero-service/aiapp/aisolo/internal/svc"
)

func TestBindKnowledgeBaseUpdatesIdleSession(t *testing.T) {
	ctx := context.Background()
	store := session.NewMemoryStore()
	sess := &session.Session{
		ID:        "session-1",
		UserID:    "user-1",
		Status:    aisolo.SessionStatus_SESSION_STATUS_IDLE,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	resp, err := NewBindKnowledgeBaseLogic(ctx, &svc.ServiceContext{Sessions: store}).BindKnowledgeBase(&aisolo.BindKnowledgeBaseReq{
		UserId:            sess.UserID,
		SessionId:         sess.ID,
		KnowledgeBaseId:   " kb-1 ",
		KnowledgeBaseName: " Knowledge ",
	})
	if err != nil {
		t.Fatalf("BindKnowledgeBase() error = %v", err)
	}
	if resp.Session.KnowledgeBaseId != "kb-1" || resp.Session.KnowledgeBaseName != "Knowledge" {
		t.Fatalf("bound session = %#v, want trimmed knowledge fields", resp.Session)
	}
}

func TestBindKnowledgeBaseRejectsRunningSession(t *testing.T) {
	ctx := context.Background()
	store := session.NewMemoryStore()
	sess := &session.Session{
		ID:        "session-1",
		UserID:    "user-1",
		Status:    aisolo.SessionStatus_SESSION_STATUS_RUNNING,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	_, err := NewBindKnowledgeBaseLogic(ctx, &svc.ServiceContext{Sessions: store}).BindKnowledgeBase(&aisolo.BindKnowledgeBaseReq{
		UserId:          sess.UserID,
		SessionId:       sess.ID,
		KnowledgeBaseId: "kb-1",
	})
	if err == nil || !strings.Contains(err.Error(), "session is running") {
		t.Fatalf("BindKnowledgeBase() error = %v, want running-session rejection", err)
	}
	got, getErr := store.GetSession(ctx, sess.UserID, sess.ID)
	if getErr != nil {
		t.Fatalf("GetSession() error = %v", getErr)
	}
	if got.KnowledgeBaseID != "" {
		t.Fatalf("KnowledgeBaseID = %q, want unchanged", got.KnowledgeBaseID)
	}
}
