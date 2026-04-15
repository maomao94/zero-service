package logic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/svc"
	"zero-service/common/einox/a2ui"
	"zero-service/common/einox/memory"

	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ResumeStreamLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	Logger logx.Logger
}

func NewResumeStreamLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResumeStreamLogic {
	return &ResumeStreamLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

type resumeGrpcWriter struct {
	streamSvc aisolo.AiSolo_ResumeStreamServer
}

func (w *resumeGrpcWriter) Write(p []byte) (n int, err error) {
	resp := &aisolo.ResumeStreamResp{
		Chunk: &aisolo.ResumeStreamChunk{
			Data:    string(p),
			IsFinal: false,
		},
	}
	if err := w.streamSvc.Send(resp); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (l *ResumeStreamLogic) ResumeStream(in *aisolo.ResumeReq, streamSvc aisolo.AiSolo_ResumeStreamServer) error {
	startTime := time.Now()

	sessionID := in.SessionId
	userID := in.UserId
	interruptID := in.InterruptId

	l.Logger.Infof("ResumeStream: session=%s, user=%s, interrupt=%s, action=%v",
		sessionID, userID, interruptID, in.Action)

	if sessionID == "" {
		return status.Error(codes.InvalidArgument, "session_id is required")
	}
	if interruptID == "" {
		return status.Error(codes.InvalidArgument, "interrupt_id is required")
	}
	if userID == "" {
		userID = "anonymous"
	}

	writer := &resumeGrpcWriter{streamSvc: streamSvc}

	history, err := l.getHistory(userID, sessionID)
	if err != nil {
		l.Logger.Errorf("get messages: %v", err)
	}

	einoAgent, cleanup, err := l.svcCtx.Router.Route(l.ctx, &aisolo.AskReq{
		UserId:    userID,
		SessionId: sessionID,
		AgentMode: aisolo.AgentMode_AGENT_MODE_FAST,
	})
	if err != nil {
		errMsg := a2ui.Message{
			SurfaceUpdate: &a2ui.SurfaceUpdateMsg{
				SurfaceID: "chat-" + sessionID,
				Components: []a2ui.Component{
					{
						ID: "error-card",
						Component: a2ui.ComponentValue{
							Card: &a2ui.CardComp{
								Children: []string{"error-content"},
							},
						},
					},
					{
						ID: "error-content",
						Component: a2ui.ComponentValue{
							Text: &a2ui.TextComp{
								Value:     fmt.Sprintf("Error: route agent failed: %v", err),
								UsageHint: "body",
							},
						},
					},
				},
			},
		}
		if err := l.emitMessage(writer, errMsg); err != nil {
			l.Logger.Errorf("send error message: %v", err)
		}
		return nil
	}
	defer cleanup()

	if einoAgent == nil || einoAgent.Runner() == nil {
		errMsg := a2ui.Message{
			SurfaceUpdate: &a2ui.SurfaceUpdateMsg{
				SurfaceID: "chat-" + sessionID,
				Components: []a2ui.Component{
					{
						ID: "error-card",
						Component: a2ui.ComponentValue{
							Card: &a2ui.CardComp{
								Children: []string{"error-content"},
							},
						},
					},
					{
						ID: "error-content",
						Component: a2ui.ComponentValue{
							Text: &a2ui.TextComp{
								Value:     "Error: No agent available",
								UsageHint: "body",
							},
						},
					},
				},
			},
		}
		if err := l.emitMessage(writer, errMsg); err != nil {
			l.Logger.Errorf("send error message: %v", err)
		}
		return nil
	}

	runner := einoAgent.Runner()

	l.Logger.Infof("ResumeStream: resuming agent with interruptID=%s, approved=%v", interruptID, in.Action == aisolo.ResumeAction_RESUME_ACTION_APPROVE)

	events, err := runner.Resume(l.ctx, interruptID)
	if err != nil {
		l.Logger.Errorf("resume agent failed: %v", err)

		errMsg := a2ui.Message{
			SurfaceUpdate: &a2ui.SurfaceUpdateMsg{
				SurfaceID: "chat-" + sessionID,
				Components: []a2ui.Component{
					{
						ID: "error-card",
						Component: a2ui.ComponentValue{
							Card: &a2ui.CardComp{
								Children: []string{"error-content"},
							},
						},
					},
					{
						ID: "error-content",
						Component: a2ui.ComponentValue{
							Text: &a2ui.TextComp{
								Value:     fmt.Sprintf("Resume failed: %v", err),
								UsageHint: "body",
							},
						},
					},
				},
			},
		}
		if err := l.emitMessage(writer, errMsg); err != nil {
			l.Logger.Errorf("send error message: %v", err)
		}

		finalResp := &aisolo.ResumeStreamResp{
			Chunk: &aisolo.ResumeStreamChunk{
				SessionId: sessionID,
				IsFinal:   true,
			},
		}
		streamSvc.Send(finalResp)
		return nil
	}

	lastContent, newInterruptID, _, err := a2ui.StreamToWriter(writer, sessionID, history, events)
	if err != nil && !errors.Is(err, io.EOF) {
		l.Logger.Errorf("stream to writer: %v", err)
		return status.Error(codes.Internal, fmt.Sprintf("agent execution failed: %v", err))
	}

	if lastContent != "" {
		l.saveMessages(userID, sessionID, "", lastContent)
	}

	if newInterruptID != "" {
		interruptMsg := a2ui.Message{
			InterruptRequest: &a2ui.InterruptRequestMsg{
				InterruptID: newInterruptID,
				Description: "approval required",
				Type:        a2ui.InterruptTypeApproval,
			},
		}
		if err := l.emitMessage(writer, interruptMsg); err != nil {
			l.Logger.Errorf("send interrupt request: %v", err)
		}
	}

	finalResp := &aisolo.ResumeStreamResp{
		Chunk: &aisolo.ResumeStreamChunk{
			SessionId: sessionID,
			IsFinal:   true,
		},
	}
	if err := streamSvc.Send(finalResp); err != nil {
		l.Logger.Errorf("send final response: %v", err)
	}

	duration := time.Since(startTime).Milliseconds()
	l.Logger.Infof("ResumeStream completed: session=%s, duration=%dms", sessionID, duration)

	return nil
}

func (l *ResumeStreamLogic) getHistory(userID, sessionID string) ([]*schema.Message, error) {
	if l.svcCtx.MemoryStorage == nil {
		return nil, nil
	}
	msgs, err := l.svcCtx.MemoryStorage.GetMessages(l.ctx, userID, sessionID, 20)
	if err != nil {
		return nil, err
	}
	var history []*schema.Message
	for _, msg := range msgs {
		history = append(history, &schema.Message{
			Role:    schema.RoleType(msg.Role),
			Content: msg.Content,
		})
	}
	return history, nil
}

func (l *ResumeStreamLogic) saveMessages(userID, sessionID, userMsg, assistantMsg string) {
	if l.svcCtx.MemoryStorage == nil {
		return
	}
	if userMsg != "" {
		if err := l.svcCtx.MemoryStorage.SaveMessage(l.ctx, &memory.ConversationMessage{
			UserID:    userID,
			SessionID: sessionID,
			Role:      "user",
			Content:   userMsg,
		}); err != nil {
			l.Logger.Errorf("save user message: %v", err)
		}
	}
	if assistantMsg != "" {
		if err := l.svcCtx.MemoryStorage.SaveMessage(l.ctx, &memory.ConversationMessage{
			UserID:    userID,
			SessionID: sessionID,
			Role:      "assistant",
			Content:   assistantMsg,
		}); err != nil {
			l.Logger.Errorf("save assistant message: %v", err)
		}
	}
}

func (l *ResumeStreamLogic) emitMessage(w io.Writer, msg a2ui.Message) error {
	data, err := a2ui.Encode(msg)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
