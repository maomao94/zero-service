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

	// 获取历史消息
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

	// 获取角色
	roleID := l.getRoleID(in)

	// 为角色创建 Agent
	einoAgent, err := l.svcCtx.RoleManager.CreateAgent(l.ctx, roleID)
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
								Value:     fmt.Sprintf("Error: create agent failed: %v", err),
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

	// 构建消息
	messages := append(history, &schema.Message{
		Role:    schema.User,
		Content: in.Message,
	})

	// 使用 Agent 的 Runner（已包含 EnableStreaming 配置）
	runner := einoAgent.Runner()
	events := runner.Run(l.ctx, messages)

	lastContent, interruptID, err := a2ui.StreamToWriter(writer, sessionID, history, events)
	if err != nil {
		l.Logger.Errorf("stream to writer: %v", err)
		if !errors.Is(err, io.EOF) {
			return status.Error(codes.Internal, fmt.Sprintf("agent execution failed: %v", err))
		}
	}

	// 保存消息
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

	// 发送中断请求
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
	l.Logger.Infof("AskStream completed: session=%s, role=%s, duration=%dms", sessionID, roleID, duration)
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

// getRoleID 根据 AgentMode 获取角色 ID
func (l *AskStreamLogic) getRoleID(in *aisolo.AskReq) string {
	switch in.AgentMode {
	case aisolo.AgentMode_AGENT_MODE_FAST:
		return "assistant"
	case aisolo.AgentMode_AGENT_MODE_DEEP:
		// Deep 模式使用 deep agent
		return "assistant" // 后续支持 deep agent
	case aisolo.AgentMode_AGENT_MODE_AUTO:
		fallthrough
	default:
		return "assistant"
	}
}
