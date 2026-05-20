package solo

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/common/ctxdata"
)

func TestSessionLogicsRequireSessionIDBeforeRPC(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxdata.CtxUserIdKey, "user-1")
	svcCtx := &svc.ServiceContext{}

	cases := []struct {
		name string
		run  func() error
	}{
		{name: "get", run: func() error {
			_, err := NewGetSessionLogic(ctx, svcCtx).GetSession(&types.SoloGetSessionRequest{})
			return err
		}},
		{name: "delete", run: func() error {
			_, err := NewDeleteSessionLogic(ctx, svcCtx).DeleteSession(&types.SoloDeleteSessionRequest{})
			return err
		}},
		{name: "messages", run: func() error {
			_, err := NewListMessagesLogic(ctx, svcCtx).ListMessages(&types.SoloListMessagesRequest{})
			return err
		}},
		{name: "bind knowledge", run: func() error {
			_, err := NewBindSessionKnowledgeLogic(ctx, svcCtx).BindSessionKnowledge(&types.SoloBindKnowledgeRequest{KnowledgeBaseId: "kb-1"})
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertValidationError(t, tc.run(), "sessionId is required")
		})
	}
}

func TestSessionLogicsRequireRequestBeforeRPC(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxdata.CtxUserIdKey, "user-1")
	svcCtx := &svc.ServiceContext{}

	cases := []struct {
		name string
		run  func() error
		want string
	}{
		{name: "get", run: func() error {
			_, err := NewGetSessionLogic(ctx, svcCtx).GetSession(nil)
			return err
		}, want: "get session request is required"},
		{name: "delete", run: func() error {
			_, err := NewDeleteSessionLogic(ctx, svcCtx).DeleteSession(nil)
			return err
		}, want: "delete session request is required"},
		{name: "messages", run: func() error {
			_, err := NewListMessagesLogic(ctx, svcCtx).ListMessages(nil)
			return err
		}, want: "list messages request is required"},
		{name: "bind knowledge", run: func() error {
			_, err := NewBindSessionKnowledgeLogic(ctx, svcCtx).BindSessionKnowledge(nil)
			return err
		}, want: "bind session knowledge request is required"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertValidationError(t, tc.run(), tc.want)
		})
	}
}

func TestGetInterruptRequiresInterruptIDBeforeRPC(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxdata.CtxUserIdKey, "user-1")
	logic := NewGetInterruptLogic(ctx, &svc.ServiceContext{})
	_, err := logic.GetInterrupt(nil)
	assertValidationError(t, err, "get interrupt request is required")
	_, err = logic.GetInterrupt(&types.SoloGetInterruptRequest{InterruptId: " \t"})
	assertValidationError(t, err, "interruptId is required")
}

func TestCreateSessionRequiresRequestBeforeRPC(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxdata.CtxUserIdKey, "user-1")
	_, err := NewCreateSessionLogic(ctx, &svc.ServiceContext{}).CreateSession(nil)
	assertValidationError(t, err, "create session request is required")
}

func TestBindSessionKnowledgeRequiresKnowledgeBaseIDBeforeRPC(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxdata.CtxUserIdKey, "user-1")
	_, err := NewBindSessionKnowledgeLogic(ctx, &svc.ServiceContext{}).BindSessionKnowledge(&types.SoloBindKnowledgeRequest{SessionId: "sess-1"})
	assertValidationError(t, err, "knowledgeBaseId is required")
}

func TestSessionLogicsKeepMissingUserPrecedence(t *testing.T) {
	err := func() error {
		_, err := NewGetSessionLogic(context.Background(), &svc.ServiceContext{}).GetSession(&types.SoloGetSessionRequest{})
		return err
	}()
	if err == nil || !strings.Contains(err.Error(), "missing user id in context") {
		t.Fatalf("error = %v, want missing user id", err)
	}
}

func TestListMessagesTrimsSessionAndPassesLimit(t *testing.T) {
	fake := &listMessagesFakeAiSoloClient{
		resp: &aisolo.ListMessagesResp{
			Total: 2,
			Messages: []*aisolo.Message{
				{Id: "m-1", SessionId: "sess-1", UserId: "user-1", Role: "user", Content: "hello", CreatedAt: 10},
				{Id: "m-2", SessionId: "sess-1", UserId: "user-1", Role: "assistant", Content: "world", CreatedAt: 11, ToolCallId: "tool-call", ToolName: "tool"},
			},
		},
	}
	ctx := context.WithValue(context.Background(), ctxdata.CtxUserIdKey, "user-1")

	resp, err := NewListMessagesLogic(ctx, &svc.ServiceContext{AiSoloCli: fake}).ListMessages(&types.SoloListMessagesRequest{
		SessionId: " sess-1 ",
		Limit:     2,
	})
	if err != nil {
		t.Fatalf("ListMessages() error = %v", err)
	}
	if fake.req == nil {
		t.Fatal("ListMessages RPC request was not captured")
	}
	if fake.req.SessionId != "sess-1" || fake.req.UserId != "user-1" || fake.req.Limit != 2 {
		t.Fatalf("ListMessages request = %#v, want trimmed session/user/limit", fake.req)
	}
	if resp.Total != 2 || len(resp.Messages) != 2 {
		t.Fatalf("response = %#v, want two messages", resp)
	}
	if resp.Messages[1].ToolCallId != "tool-call" || resp.Messages[1].ToolName != "tool" {
		t.Fatalf("message = %#v, want tool fields mapped", resp.Messages[1])
	}
}

func TestListMessagesPassesNegativeLimitToRPC(t *testing.T) {
	fake := &listMessagesFakeAiSoloClient{resp: &aisolo.ListMessagesResp{}}
	ctx := context.WithValue(context.Background(), ctxdata.CtxUserIdKey, "user-1")

	_, err := NewListMessagesLogic(ctx, &svc.ServiceContext{AiSoloCli: fake}).ListMessages(&types.SoloListMessagesRequest{
		SessionId: "sess-1",
		Limit:     -1,
	})
	if err != nil {
		t.Fatalf("ListMessages() error = %v", err)
	}
	if fake.req == nil || fake.req.Limit != -1 {
		t.Fatalf("ListMessages request = %#v, want negative limit passed through", fake.req)
	}
}

func TestListSessionsAllowsNilRequest(t *testing.T) {
	fake := &listSessionsFakeAiSoloClient{resp: &aisolo.ListSessionsResp{Page: 1}}
	ctx := context.WithValue(context.Background(), ctxdata.CtxUserIdKey, "user-1")

	_, err := NewListSessionsLogic(ctx, &svc.ServiceContext{AiSoloCli: fake}).ListSessions(nil)
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if fake.req == nil || fake.req.UserId != "user-1" || fake.req.Page != 0 || fake.req.PageSize != 0 {
		t.Fatalf("ListSessions request = %#v, want default nil-request payload", fake.req)
	}
}

type listMessagesFakeAiSoloClient struct {
	aisolo.AiSoloClient
	req  *aisolo.ListMessagesReq
	resp *aisolo.ListMessagesResp
}

func (c *listMessagesFakeAiSoloClient) ListMessages(ctx context.Context, in *aisolo.ListMessagesReq, opts ...grpc.CallOption) (*aisolo.ListMessagesResp, error) {
	c.req = in
	return c.resp, nil
}

type listSessionsFakeAiSoloClient struct {
	aisolo.AiSoloClient
	req  *aisolo.ListSessionsReq
	resp *aisolo.ListSessionsResp
}

func (c *listSessionsFakeAiSoloClient) ListSessions(ctx context.Context, in *aisolo.ListSessionsReq, opts ...grpc.CallOption) (*aisolo.ListSessionsResp, error) {
	c.req = in
	return c.resp, nil
}
