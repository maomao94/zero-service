package logic

import (
	"context"
	"strings"
	"testing"
	"time"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/session"
	"zero-service/aiapp/aisolo/internal/svc"
	"zero-service/common/einox/memory"
)

func TestCreateSessionRequiresRequestAndUserID(t *testing.T) {
	logic := NewCreateSessionLogic(context.Background(), &svc.ServiceContext{})
	_, err := logic.CreateSession(nil)
	assertRequiredIDError(t, err, "create session request is required")
	_, err = logic.CreateSession(&aisolo.CreateSessionReq{UserId: " \t"})
	assertRequiredIDError(t, err, "user_id is required")
}

func TestCreateSessionTrimsUserTitleAndKnowledgeFields(t *testing.T) {
	store := session.NewMemoryStore()
	resp, err := NewCreateSessionLogic(context.Background(), &svc.ServiceContext{Sessions: store}).CreateSession(&aisolo.CreateSessionReq{
		UserId:            " user-1 ",
		Title:             " Title ",
		KnowledgeBaseId:   " kb-1 ",
		KnowledgeBaseName: " Knowledge ",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if resp.GetSession().GetUserId() != "user-1" || resp.GetSession().GetTitle() != "Title" {
		t.Fatalf("session = %#v, want trimmed user/title", resp.GetSession())
	}
	if resp.GetSession().GetKnowledgeBaseId() != "kb-1" || resp.GetSession().GetKnowledgeBaseName() != "Knowledge" {
		t.Fatalf("session = %#v, want trimmed knowledge fields", resp.GetSession())
	}
}

func TestGetSessionRequiresUserAndSessionID(t *testing.T) {
	_, err := NewGetSessionLogic(context.Background(), &svc.ServiceContext{}).GetSession(&aisolo.GetSessionReq{})
	assertRequiredIDError(t, err, "user_id and session_id are required")
	_, err = NewGetSessionLogic(context.Background(), &svc.ServiceContext{}).GetSession(&aisolo.GetSessionReq{UserId: " user-1 ", SessionId: " \t"})
	assertRequiredIDError(t, err, "user_id and session_id are required")
}

func TestBindKnowledgeBaseRequiresRequestUserSessionAndKnowledge(t *testing.T) {
	logic := NewBindKnowledgeBaseLogic(context.Background(), &svc.ServiceContext{})
	_, err := logic.BindKnowledgeBase(nil)
	assertRequiredIDError(t, err, "bind knowledge base request is required")
	_, err = logic.BindKnowledgeBase(&aisolo.BindKnowledgeBaseReq{UserId: " \t", SessionId: "sess-1", KnowledgeBaseId: "kb-1"})
	assertRequiredIDError(t, err, "user_id and session_id are required")
	_, err = logic.BindKnowledgeBase(&aisolo.BindKnowledgeBaseReq{UserId: "user-1", SessionId: " \t", KnowledgeBaseId: "kb-1"})
	assertRequiredIDError(t, err, "user_id and session_id are required")
	_, err = logic.BindKnowledgeBase(&aisolo.BindKnowledgeBaseReq{UserId: "user-1", SessionId: "sess-1", KnowledgeBaseId: " \t"})
	assertRequiredIDError(t, err, "knowledge_base_id is required")
}

func TestBindKnowledgeBaseTrimsLookupIDs(t *testing.T) {
	ctx := context.Background()
	store := session.NewMemoryStore()
	sess := &session.Session{ID: "sess-1", UserID: "user-1", Status: aisolo.SessionStatus_SESSION_STATUS_IDLE, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := store.CreateSession(ctx, sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	resp, err := NewBindKnowledgeBaseLogic(ctx, &svc.ServiceContext{Sessions: store}).BindKnowledgeBase(&aisolo.BindKnowledgeBaseReq{
		UserId:            " user-1 ",
		SessionId:         " sess-1 ",
		KnowledgeBaseId:   " kb-1 ",
		KnowledgeBaseName: " Knowledge ",
	})
	if err != nil {
		t.Fatalf("BindKnowledgeBase() error = %v", err)
	}
	if resp.GetSession().GetKnowledgeBaseId() != "kb-1" || resp.GetSession().GetKnowledgeBaseName() != "Knowledge" {
		t.Fatalf("session = %#v, want trimmed knowledge fields", resp.GetSession())
	}
}

func TestGetInterruptRequiresRequestAndInterruptID(t *testing.T) {
	logic := NewGetInterruptLogic(context.Background(), &svc.ServiceContext{})
	_, err := logic.GetInterrupt(nil)
	assertRequiredIDError(t, err, "get interrupt request is required")
	_, err = logic.GetInterrupt(&aisolo.GetInterruptReq{InterruptId: " \t"})
	assertRequiredIDError(t, err, "interrupt_id is required")
}

func TestGetInterruptTrimsIDAndUserCheck(t *testing.T) {
	ctx := context.Background()
	store := session.NewMemoryStore()
	if err := store.SaveInterrupt(ctx, &session.InterruptRecord{InterruptID: "interrupt-1", UserID: "user-1"}); err != nil {
		t.Fatalf("SaveInterrupt() error = %v", err)
	}
	resp, err := NewGetInterruptLogic(ctx, &svc.ServiceContext{Sessions: store}).GetInterrupt(&aisolo.GetInterruptReq{InterruptId: " interrupt-1 ", UserId: " user-1 "})
	if err != nil {
		t.Fatalf("GetInterrupt() error = %v", err)
	}
	if resp.GetInfo().GetInterruptId() != "interrupt-1" {
		t.Fatalf("interrupt = %#v, want interrupt-1", resp.GetInfo())
	}
	_, err = NewGetInterruptLogic(ctx, &svc.ServiceContext{Sessions: store}).GetInterrupt(&aisolo.GetInterruptReq{InterruptId: "interrupt-1", UserId: "other"})
	assertRequiredIDError(t, err, "interrupt does not belong to current user")
}

func TestListMessagesRequiresUserAndSessionID(t *testing.T) {
	_, err := NewListMessagesLogic(context.Background(), &svc.ServiceContext{}).ListMessages(&aisolo.ListMessagesReq{})
	assertRequiredIDError(t, err, "user_id and session_id are required")
	_, err = NewListMessagesLogic(context.Background(), &svc.ServiceContext{}).ListMessages(&aisolo.ListMessagesReq{UserId: " user-1 ", SessionId: " \t"})
	assertRequiredIDError(t, err, "user_id and session_id are required")
}

func TestListMessagesLimitBoundaries(t *testing.T) {
	ctx := context.Background()
	store := memory.NewMemoryStorage()
	base := time.Unix(100, 0)
	for i, content := range []string{"first", "second", "third"} {
		if err := store.SaveMessage(ctx, &memory.ConversationMessage{
			ID:        content,
			SessionID: "sess-1",
			UserID:    "user-1",
			Role:      "user",
			Content:   content,
			CreatedAt: base.Add(time.Duration(i) * time.Second),
		}); err != nil {
			t.Fatalf("SaveMessage() error = %v", err)
		}
	}

	cases := []struct {
		name  string
		limit int32
		want  []string
	}{
		{name: "negative means all", limit: -1, want: []string{"first", "second", "third"}},
		{name: "zero means all", limit: 0, want: []string{"first", "second", "third"}},
		{name: "positive returns latest ascending", limit: 2, want: []string{"second", "third"}},
		{name: "limit larger than total returns all", limit: 10, want: []string{"first", "second", "third"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := NewListMessagesLogic(ctx, &svc.ServiceContext{Messages: store}).ListMessages(&aisolo.ListMessagesReq{
				UserId:    "user-1",
				SessionId: "sess-1",
				Limit:     tc.limit,
			})
			if err != nil {
				t.Fatalf("ListMessages() error = %v", err)
			}
			if resp.GetTotal() != int32(len(tc.want)) {
				t.Fatalf("total = %d, want %d", resp.GetTotal(), len(tc.want))
			}
			if got := messageContents(resp.GetMessages()); !sameStrings(got, tc.want) {
				t.Fatalf("messages = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestListSessionsRequiresUserID(t *testing.T) {
	_, err := NewListSessionsLogic(context.Background(), &svc.ServiceContext{}).ListSessions(&aisolo.ListSessionsReq{})
	assertRequiredIDError(t, err, "user_id is required")
	_, err = NewListSessionsLogic(context.Background(), &svc.ServiceContext{}).ListSessions(&aisolo.ListSessionsReq{UserId: " \t"})
	assertRequiredIDError(t, err, "user_id is required")
}

func TestDeleteSessionRequiresUserAndSessionID(t *testing.T) {
	resp, err := NewDeleteSessionLogic(context.Background(), &svc.ServiceContext{}).DeleteSession(&aisolo.DeleteSessionReq{})
	assertRequiredIDError(t, err, "user_id and session_id are required")
	if resp == nil || resp.Success {
		t.Fatalf("DeleteSession() resp = %#v, want unsuccessful response", resp)
	}
	resp, err = NewDeleteSessionLogic(context.Background(), &svc.ServiceContext{}).DeleteSession(&aisolo.DeleteSessionReq{UserId: " user-1 ", SessionId: " \t"})
	assertRequiredIDError(t, err, "user_id and session_id are required")
	if resp == nil || resp.Success {
		t.Fatalf("DeleteSession() resp = %#v, want unsuccessful response", resp)
	}
}

func assertRequiredIDError(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func messageContents(msgs []*aisolo.Message) []string {
	out := make([]string, 0, len(msgs))
	for _, msg := range msgs {
		out = append(out, msg.GetContent())
	}
	return out
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
