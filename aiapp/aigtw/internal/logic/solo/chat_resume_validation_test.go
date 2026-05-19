package solo

import (
	"context"
	"io"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/common/ctxdata"
)

func TestValidateChatRequest(t *testing.T) {
	cases := []struct {
		name string
		req  *types.SoloChatRequest
		want string
	}{
		{name: "nil", req: nil, want: "chat request is required"},
		{name: "session", req: &types.SoloChatRequest{Message: "hello"}, want: "sessionId is required"},
		{name: "message", req: &types.SoloChatRequest{SessionId: "sess-1", Message: " \t"}, want: "message is required"},
		{name: "ok", req: &types.SoloChatRequest{SessionId: " sess-1 ", Message: " hello "}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateChatRequest(tc.req)
			assertValidationError(t, err, tc.want)
		})
	}
}

func TestValidateResumeRequest(t *testing.T) {
	cases := []struct {
		name string
		req  *types.SoloInterruptRequest
		want string
	}{
		{name: "nil", req: nil, want: "resume request is required"},
		{name: "session", req: &types.SoloInterruptRequest{InterruptId: "interrupt-1", Action: "yes"}, want: "sessionId is required"},
		{name: "interrupt", req: &types.SoloInterruptRequest{SessionId: "sess-1", Action: "yes"}, want: "interruptId is required"},
		{name: "action", req: &types.SoloInterruptRequest{SessionId: "sess-1", InterruptId: "interrupt-1"}, want: "action must be yes or no"},
		{name: "ok yes", req: &types.SoloInterruptRequest{SessionId: " sess-1 ", InterruptId: " interrupt-1 ", Action: " yes "}},
		{name: "ok no", req: &types.SoloInterruptRequest{SessionId: "sess-1", InterruptId: "interrupt-1", Action: "no"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateResumeRequest(tc.req)
			assertValidationError(t, err, tc.want)
		})
	}
}

func TestChatValidatesRequestBeforeOpeningStream(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxdata.CtxUserIdKey, "user-1")
	err := NewChatLogic(ctx, &svc.ServiceContext{}).Chat(&types.SoloChatRequest{SessionId: "sess-1"}, &strings.Builder{})
	assertValidationError(t, err, "message is required")
}

func TestResumeValidatesRequestBeforeOpeningStream(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxdata.CtxUserIdKey, "user-1")
	err := NewResumeLogic(ctx, &svc.ServiceContext{}).Resume(&types.SoloInterruptRequest{SessionId: "sess-1", InterruptId: "interrupt-1"}, &strings.Builder{})
	assertValidationError(t, err, "action must be yes or no")
}

func TestChatForwardsAiSoloStreamAsSSE(t *testing.T) {
	fake := &streamFakeAiSoloClient{
		askStream: &fakeServerStream[aisolo.AskStreamResp]{responses: []*aisolo.AskStreamResp{
			{Chunk: &aisolo.AskStreamChunk{Data: "", IsFinal: false}},
			{Chunk: &aisolo.AskStreamChunk{Data: "{\"event\":\"turn.start\"}\n", IsFinal: false}},
			{Chunk: &aisolo.AskStreamChunk{Data: "{\"event\":\"turn.end\"}\r\n", IsFinal: true}},
			{Chunk: &aisolo.AskStreamChunk{Data: "{\"event\":\"after.final\"}\n", IsFinal: false}},
		}},
	}
	ctx := context.WithValue(context.Background(), ctxdata.CtxUserIdKey, "user-1")
	var out strings.Builder

	err := NewChatLogic(ctx, &svc.ServiceContext{AiSoloCli: fake}).Chat(&types.SoloChatRequest{
		SessionId: " sess-1 ",
		Message:   " hello ",
		Mode:      "agent",
		UiLang:    " zh ",
	}, &out)
	if err != nil {
		t.Fatalf("Chat() error = %v, want nil", err)
	}
	if got, want := out.String(), "data: {\"event\":\"turn.start\"}\n\ndata: {\"event\":\"turn.end\"}\n\n"; got != want {
		t.Fatalf("SSE output = %q, want %q", got, want)
	}
	if fake.askReq == nil {
		t.Fatal("AskStream request was not captured")
	}
	if fake.askReq.SessionId != "sess-1" || fake.askReq.UserId != "user-1" || fake.askReq.Message != "hello" || fake.askReq.UiLang != "zh" {
		t.Fatalf("AskStream request = %#v, want trimmed request", fake.askReq)
	}
	if fake.askReq.Mode != aisolo.AgentMode_AGENT_MODE_AGENT {
		t.Fatalf("AskStream mode = %v, want AGENT_MODE_AGENT", fake.askReq.Mode)
	}
}

func TestResumeForwardsAiSoloStreamAsSSE(t *testing.T) {
	fake := &streamFakeAiSoloClient{
		resumeStream: &fakeServerStream[aisolo.ResumeStreamResp]{responses: []*aisolo.ResumeStreamResp{
			{Chunk: nil},
			{Chunk: &aisolo.ResumeStreamChunk{Data: "{\"event\":\"resume.start\"}\n", IsFinal: false}},
			{Chunk: &aisolo.ResumeStreamChunk{Data: "{\"event\":\"resume.end\"}\n", IsFinal: true}},
		}},
	}
	ctx := context.WithValue(context.Background(), ctxdata.CtxUserIdKey, "user-1")
	var out strings.Builder

	err := NewResumeLogic(ctx, &svc.ServiceContext{AiSoloCli: fake}).Resume(&types.SoloInterruptRequest{
		SessionId:   " sess-1 ",
		InterruptId: " interrupt-1 ",
		Action:      " no ",
		Reason:      "not safe",
		SelectedIds: []string{"a", "b"},
		Text:        "answer",
		FormValues:  map[string]string{"k": "v"},
	}, &out)
	if err != nil {
		t.Fatalf("Resume() error = %v, want nil", err)
	}
	if got, want := out.String(), "data: {\"event\":\"resume.start\"}\n\ndata: {\"event\":\"resume.end\"}\n\n"; got != want {
		t.Fatalf("SSE output = %q, want %q", got, want)
	}
	if fake.resumeReq == nil {
		t.Fatal("ResumeStream request was not captured")
	}
	if fake.resumeReq.SessionId != "sess-1" || fake.resumeReq.UserId != "user-1" || fake.resumeReq.InterruptId != "interrupt-1" {
		t.Fatalf("ResumeStream request = %#v, want trimmed ids", fake.resumeReq)
	}
	if fake.resumeReq.Action != aisolo.ResumeAction_RESUME_ACTION_NO {
		t.Fatalf("ResumeStream action = %v, want RESUME_ACTION_NO", fake.resumeReq.Action)
	}
	if fake.resumeReq.Reason != "not safe" || fake.resumeReq.Text != "answer" || fake.resumeReq.FormValues["k"] != "v" {
		t.Fatalf("ResumeStream payload = %#v, want original payload", fake.resumeReq)
	}
}

type streamFakeAiSoloClient struct {
	aisolo.AiSoloClient
	askReq       *aisolo.AskReq
	askStream    grpc.ServerStreamingClient[aisolo.AskStreamResp]
	resumeReq    *aisolo.ResumeReq
	resumeStream grpc.ServerStreamingClient[aisolo.ResumeStreamResp]
}

func (c *streamFakeAiSoloClient) AskStream(ctx context.Context, in *aisolo.AskReq, opts ...grpc.CallOption) (grpc.ServerStreamingClient[aisolo.AskStreamResp], error) {
	c.askReq = in
	return c.askStream, nil
}

func (c *streamFakeAiSoloClient) ResumeStream(ctx context.Context, in *aisolo.ResumeReq, opts ...grpc.CallOption) (grpc.ServerStreamingClient[aisolo.ResumeStreamResp], error) {
	c.resumeReq = in
	return c.resumeStream, nil
}

type fakeServerStream[T any] struct {
	grpc.ClientStream
	responses []*T
	idx       int
}

func (s *fakeServerStream[T]) Recv() (*T, error) {
	if s.idx >= len(s.responses) {
		return nil, io.EOF
	}
	resp := s.responses[s.idx]
	s.idx++
	return resp, nil
}

func (s *fakeServerStream[T]) Header() (metadata.MD, error) { return nil, nil }
func (s *fakeServerStream[T]) Trailer() metadata.MD         { return nil }
func (s *fakeServerStream[T]) CloseSend() error             { return nil }
func (s *fakeServerStream[T]) Context() context.Context     { return context.Background() }
func (s *fakeServerStream[T]) SendMsg(m any) error          { return nil }
func (s *fakeServerStream[T]) RecvMsg(m any) error          { return nil }

func assertValidationError(t *testing.T, err error, want string) {
	t.Helper()
	if want == "" {
		if err != nil {
			t.Fatalf("error = %v, want nil", err)
		}
		return
	}
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
}
