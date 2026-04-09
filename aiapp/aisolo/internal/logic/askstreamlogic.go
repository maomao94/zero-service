package logic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/router"
	"zero-service/aiapp/aisolo/internal/svc"
	"zero-service/common/a2ui"
	"zero-service/common/einox/memory"

	"github.com/cloudwego/eino/adk"
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
	chunk := &aisolo.StreamChunk{
		Data: string(p),
	}
	if err := w.streamSvc.Send(chunk); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (l *AskStreamLogic) AskStream(in *aisolo.AskRequest, streamSvc aisolo.AiSolo_AskStreamServer) error {
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

	var history []*schema.Message
	if l.svcCtx.MemoryStorage != nil {
		msgs, err := l.svcCtx.MemoryStorage.GetMessages(l.ctx, userID, sessionID, 20)
		if err != nil {
			l.Logger.Errorf("get messages: %v", err)
		} else {
			for _, msg := range msgs {
				history = append(history, &schema.Message{
					Role:    schema.RoleType(msg.Role),
					Content: msg.Content,
				})
			}
		}
	}

	agentMode := l.selectAgentMode(in)
	agentName := l.getAgentName(agentMode, in.Message)

	agent := l.svcCtx.GetAgent(agentName)
	if agent == nil {
		agent = l.svcCtx.GetAgent("chat_model")
	}

	if agent == nil {
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

	runner := adk.NewRunner(l.ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	events := runner.Run(l.ctx, messages)

	lastContent, interruptID, err := a2ui.StreamToWriter(writer, sessionID, history, events)
	if err != nil {
		l.Logger.Errorf("stream to writer: %v", err)
		if !errors.Is(err, io.EOF) {
			return status.Error(codes.Internal, fmt.Sprintf("agent execution failed: %v", err))
		}
	}

	if l.svcCtx.MemoryStorage != nil {
		if err := l.svcCtx.MemoryStorage.SaveMessage(l.ctx, &memory.ConversationMessage{
			UserID:    userID,
			SessionID: sessionID,
			Role:      "user",
			Content:   in.Message,
		}); err != nil {
			l.Logger.Errorf("save user message: %v", err)
		}
		if lastContent != "" {
			if err := l.svcCtx.MemoryStorage.SaveMessage(l.ctx, &memory.ConversationMessage{
				UserID:    userID,
				SessionID: sessionID,
				Role:      "assistant",
				Content:   lastContent,
			}); err != nil {
				l.Logger.Errorf("save assistant message: %v", err)
			}
		}
	}

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
	l.Logger.Infof("AskStream completed: session=%s, agent=%s, duration=%dms", sessionID, agentName, duration)
	return nil
}

func (l *AskStreamLogic) emitMessage(w io.Writer, msg a2ui.Message) error {
	data, err := a2ui.Encode(msg)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (l *AskStreamLogic) selectAgentMode(in *aisolo.AskRequest) aisolo.AgentMode {
	if in.AgentMode != aisolo.AgentMode_AGENT_MODE_UNSPECIFIED {
		return in.AgentMode
	}
	return aisolo.AgentMode_AGENT_MODE_AUTO
}

func (l *AskStreamLogic) getAgentName(mode aisolo.AgentMode, message string) string {
	switch mode {
	case aisolo.AgentMode_AGENT_MODE_FAST:
		return "chat_model"
	case aisolo.AgentMode_AGENT_MODE_DEEP:
		return "deep"
	case aisolo.AgentMode_AGENT_MODE_AUTO:
		fallthrough
	default:
		if l.svcCtx.Router != nil && message != "" {
			decision, err := l.svcCtx.Router.Route(l.ctx, message)
			if err == nil && decision != nil {
				return l.decisionToAgentName(decision.SelectedAgent)
			}
		}
		return "chat_model"
	}
}

func (l *AskStreamLogic) decisionToAgentName(agentType router.AgentType) string {
	switch agentType {
	case router.AgentTypeDeep:
		return "deep"
	case router.AgentTypeSequential:
		return "sequential"
	case router.AgentTypeParallel:
		return "parallel"
	case router.AgentTypeSupervisor:
		return "supervisor"
	case router.AgentTypePlanExecute:
		return "plan_execute"
	default:
		return "chat_model"
	}
}
