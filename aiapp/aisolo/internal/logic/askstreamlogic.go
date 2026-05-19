package logic

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/svc"
	"zero-service/aiapp/aisolo/internal/turn"
	"zero-service/common/einox/protocol"
)

type AskStreamLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAskStreamLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AskStreamLogic {
	return &AskStreamLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// AskStream 流式对话。此 handler 是薄壳, 核心在 turn.Executor。
func (l *AskStreamLogic) AskStream(in *aisolo.AskReq, stream aisolo.AiSolo_AskStreamServer) error {
	if l.svcCtx.Executor == nil {
		return errors.New("executor not ready (chat model may be missing)")
	}
	if in == nil {
		return errors.New("ask request is required")
	}
	sessionID := strings.TrimSpace(in.SessionId)
	userID := strings.TrimSpace(in.UserId)
	message := strings.TrimSpace(in.Message)
	if sessionID == "" {
		return errors.New("session_id is required")
	}
	if userID == "" {
		return errors.New("user_id is required")
	}
	if message == "" {
		return errors.New("message is required")
	}

	turnID := uuid.NewString()
	sender := &askStreamSender{stream: stream, sessionID: sessionID}
	em := protocol.NewEmitter(sender, sessionID, turnID)

	err := l.svcCtx.Executor.Ask(stream.Context(), em, turn.AskInput{
		SessionID: sessionID,
		UserID:    userID,
		Message:   message,
		Mode:      in.Mode,
		UILang:    in.GetUiLang(),
	})

	// 最终一帧: 强制 is_final=true, 让客户端知道流结束
	_ = stream.Send(&aisolo.AskStreamResp{
		Chunk: &aisolo.AskStreamChunk{
			SessionId: sessionID,
			IsFinal:   true,
		},
	})
	return err
}
