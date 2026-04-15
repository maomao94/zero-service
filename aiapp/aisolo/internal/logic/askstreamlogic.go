package logic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/svc"
	"zero-service/common/a2ui"
	"zero-service/common/einox/memory"

	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AskStreamLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	Logger logx.Logger
}

func NewAskStreamLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AskStreamLogic {
	return &AskStreamLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

type grpcWriter struct {
	streamSvc aisolo.AiSolo_AskStreamServer
}

func (w *grpcWriter) Write(p []byte) (n int, err error) {
	resp := &aisolo.AskStreamResp{
		Chunk: &aisolo.AskStreamChunk{
			SessionId: "",
			Data:      string(p),
			IsFinal:   false,
		},
	}
	if err := w.streamSvc.Send(resp); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (l *AskStreamLogic) AskStream(in *aisolo.AskReq, streamSvc aisolo.AiSolo_AskStreamServer) error {
	startTime := time.Now()

	sessionID := in.SessionId
	userID := in.UserId
	if userID == "" {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}
	if sessionID == "" {
		return status.Error(codes.InvalidArgument, "session_id is required")
	}
	if in.Message == "" {
		return status.Error(codes.InvalidArgument, "message is required")
	}

	writer := &grpcWriter{streamSvc: streamSvc}

	history, err := l.getHistory(userID, sessionID)
	if err != nil {
		l.Logger.Errorf("get messages: %v", err)
	}

	einoAgent, cleanup, err := l.svcCtx.Router.Route(l.ctx, in)
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

	messages := append(history, &schema.Message{
		Role:    schema.User,
		Content: in.Message,
	})

	runner := einoAgent.Runner()
	events := runner.Run(l.ctx, messages)

	lastContent, interruptID, err := a2ui.StreamToWriter(writer, sessionID, history, events)
	if err != nil && !errors.Is(err, io.EOF) {
		l.Logger.Errorf("stream to writer: %v", err)
		return status.Error(codes.Internal, fmt.Sprintf("agent execution failed: %v", err))
	}

	l.saveMessages(userID, sessionID, in.Message, lastContent)

	if interruptID != "" {
		interruptMsg := a2ui.Message{
			InterruptRequest: &a2ui.InterruptRequestMsg{
				InterruptID: interruptID,
				Description: "approval required",
				Type:        a2ui.InterruptTypeApproval,
				Required:    true,
			},
		}
		if err := l.emitMessage(writer, interruptMsg); err != nil {
			l.Logger.Errorf("send interrupt request: %v", err)
		}
	}

	duration := time.Since(startTime).Milliseconds()
	l.Logger.Infof("AskStream completed: session=%s, mode=%s, duration=%dms", sessionID, in.AgentMode.String(), duration)
	return nil
}

func (l *AskStreamLogic) getHistory(userID, sessionID string) ([]*schema.Message, error) {
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

func (l *AskStreamLogic) saveMessages(userID, sessionID, userMsg, assistantMsg string) {
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

func (l *AskStreamLogic) emitMessage(w io.Writer, msg a2ui.Message) error {
	data, err := a2ui.Encode(msg)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
