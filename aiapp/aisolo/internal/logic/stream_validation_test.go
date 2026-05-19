package logic

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/grpc/metadata"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/session"
	"zero-service/aiapp/aisolo/internal/svc"
	"zero-service/aiapp/aisolo/internal/turn"
	"zero-service/common/einox/memory"
)

func TestAskStreamRequiresUserSessionAndMessage(t *testing.T) {
	logic := NewAskStreamLogic(context.Background(), streamTestSvc())

	cases := []struct {
		name string
		req  *aisolo.AskReq
		want string
	}{
		{name: "nil request", req: nil, want: "ask request is required"},
		{name: "session", req: &aisolo.AskReq{UserId: "user-1", Message: "hello"}, want: "session_id is required"},
		{name: "user", req: &aisolo.AskReq{SessionId: "sess-1", Message: "hello"}, want: "user_id is required"},
		{name: "message", req: &aisolo.AskReq{SessionId: "sess-1", UserId: "user-1", Message: " \t\n"}, want: "message is required"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stream := &askStreamRecorder{ctx: context.Background()}
			err := logic.AskStream(tc.req, stream)
			assertRequiredIDError(t, err, tc.want)
			if len(stream.sent) != 0 {
				t.Fatalf("sent frames = %d, want 0 before request validation passes", len(stream.sent))
			}
		})
	}
}

func TestResumeStreamRequiresUserSessionInterruptAndAction(t *testing.T) {
	logic := NewResumeStreamLogic(context.Background(), streamTestSvc())

	cases := []struct {
		name string
		req  *aisolo.ResumeReq
		want string
	}{
		{name: "nil request", req: nil, want: "resume request is required"},
		{name: "session", req: &aisolo.ResumeReq{UserId: "user-1", InterruptId: "interrupt-1", Action: aisolo.ResumeAction_RESUME_ACTION_YES}, want: "session_id and interrupt_id are required"},
		{name: "interrupt", req: &aisolo.ResumeReq{SessionId: "sess-1", UserId: "user-1", Action: aisolo.ResumeAction_RESUME_ACTION_YES}, want: "session_id and interrupt_id are required"},
		{name: "user", req: &aisolo.ResumeReq{SessionId: "sess-1", InterruptId: "interrupt-1", Action: aisolo.ResumeAction_RESUME_ACTION_YES}, want: "user_id is required"},
		{name: "action", req: &aisolo.ResumeReq{SessionId: "sess-1", UserId: "user-1", InterruptId: "interrupt-1"}, want: "resume action must be yes or no"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stream := &resumeStreamRecorder{ctx: context.Background()}
			err := logic.ResumeStream(tc.req, stream)
			assertRequiredIDError(t, err, tc.want)
			if len(stream.sent) != 0 {
				t.Fatalf("sent frames = %d, want 0 before request validation passes", len(stream.sent))
			}
		})
	}
}

func TestAskStreamSendsFinalFrameAfterExecutorError(t *testing.T) {
	logic := NewAskStreamLogic(context.Background(), streamTestSvc())
	stream := &askStreamRecorder{ctx: context.Background()}

	err := logic.AskStream(&aisolo.AskReq{SessionId: " missing-session ", UserId: " user-1 ", Message: " hello "}, stream)
	if err == nil || !strings.Contains(err.Error(), "session: not found") {
		t.Fatalf("AskStream() error = %v, want missing session error", err)
	}
	if len(stream.sent) == 0 || !stream.sent[len(stream.sent)-1].GetChunk().GetIsFinal() {
		t.Fatalf("last frame = %#v, want final frame after executor returns", stream.sent)
	}
	if got := stream.sent[len(stream.sent)-1].GetChunk().GetSessionId(); got != "missing-session" {
		t.Fatalf("final session_id = %q, want trimmed session id", got)
	}
}

func TestResumeStreamSendsFinalFrameAfterExecutorError(t *testing.T) {
	logic := NewResumeStreamLogic(context.Background(), streamTestSvc())
	stream := &resumeStreamRecorder{ctx: context.Background()}

	err := logic.ResumeStream(&aisolo.ResumeReq{
		SessionId:   " missing-session ",
		UserId:      " user-1 ",
		InterruptId: " interrupt-1 ",
		Action:      aisolo.ResumeAction_RESUME_ACTION_YES,
	}, stream)
	if err == nil || !strings.Contains(err.Error(), "session: not found") {
		t.Fatalf("ResumeStream() error = %v, want missing session error", err)
	}
	if len(stream.sent) == 0 || !stream.sent[len(stream.sent)-1].GetChunk().GetIsFinal() {
		t.Fatalf("last frame = %#v, want final frame after executor returns", stream.sent)
	}
	if got := stream.sent[len(stream.sent)-1].GetChunk().GetSessionId(); got != "missing-session" {
		t.Fatalf("final session_id = %q, want trimmed session id", got)
	}
}

func streamTestSvc() *svc.ServiceContext {
	return &svc.ServiceContext{
		Executor: turn.New(turn.Config{
			Messages: memory.NewMemoryStorage(),
			Sessions: session.NewMemoryStore(),
		}),
	}
}

type askStreamRecorder struct {
	ctx  context.Context
	sent []*aisolo.AskStreamResp
}

func (s *askStreamRecorder) Send(resp *aisolo.AskStreamResp) error {
	s.sent = append(s.sent, resp)
	return nil
}

func (s *askStreamRecorder) SetHeader(metadata.MD) error  { return nil }
func (s *askStreamRecorder) SendHeader(metadata.MD) error { return nil }
func (s *askStreamRecorder) SetTrailer(metadata.MD)       {}
func (s *askStreamRecorder) Context() context.Context     { return s.ctx }
func (s *askStreamRecorder) SendMsg(any) error            { return nil }
func (s *askStreamRecorder) RecvMsg(any) error            { return nil }

type resumeStreamRecorder struct {
	ctx  context.Context
	sent []*aisolo.ResumeStreamResp
}

func (s *resumeStreamRecorder) Send(resp *aisolo.ResumeStreamResp) error {
	s.sent = append(s.sent, resp)
	return nil
}

func (s *resumeStreamRecorder) SetHeader(metadata.MD) error  { return nil }
func (s *resumeStreamRecorder) SendHeader(metadata.MD) error { return nil }
func (s *resumeStreamRecorder) SetTrailer(metadata.MD)       {}
func (s *resumeStreamRecorder) Context() context.Context     { return s.ctx }
func (s *resumeStreamRecorder) SendMsg(any) error            { return nil }
func (s *resumeStreamRecorder) RecvMsg(any) error            { return nil }
