package session

import (
	"testing"
	"time"

	"zero-service/aiapp/aisolo/aisolo"
)

func TestGormxSessionRowConversionKeepsKnowledgeFields(t *testing.T) {
	now := time.Now()
	sess := &Session{
		ID:                "sess-1",
		UserID:            "user-1",
		Title:             "title",
		Mode:              aisolo.AgentMode_AGENT_MODE_AGENT,
		Status:            aisolo.SessionStatus_SESSION_STATUS_IDLE,
		KnowledgeBaseID:   "kb-1",
		KnowledgeBaseName: "Knowledge Base",
		RunOwner:          "worker-1",
		RunLeaseUntil:     now.Add(time.Minute),
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	row, err := sessionToRow(sess)
	if err != nil {
		t.Fatalf("sessionToRow() error = %v", err)
	}
	if row.KnowledgeBaseID != sess.KnowledgeBaseID {
		t.Fatalf("KnowledgeBaseID not written: got %q want %q", row.KnowledgeBaseID, sess.KnowledgeBaseID)
	}
	if row.KnowledgeBaseName != sess.KnowledgeBaseName {
		t.Fatalf("KnowledgeBaseName not written: got %q want %q", row.KnowledgeBaseName, sess.KnowledgeBaseName)
	}

	got, err := rowToSession(row)
	if err != nil {
		t.Fatalf("rowToSession() error = %v", err)
	}
	if got.KnowledgeBaseID != sess.KnowledgeBaseID {
		t.Fatalf("KnowledgeBaseID not read: got %q want %q", got.KnowledgeBaseID, sess.KnowledgeBaseID)
	}
	if got.KnowledgeBaseName != sess.KnowledgeBaseName {
		t.Fatalf("KnowledgeBaseName not read: got %q want %q", got.KnowledgeBaseName, sess.KnowledgeBaseName)
	}
}
