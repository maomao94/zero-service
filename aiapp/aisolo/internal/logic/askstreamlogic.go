package logic

import (
	"context"
	"errors"

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
	if in.SessionId == "" {
		return errors.New("session_id is required")
	}

	turnID := uuid.NewString()
	sender := &askStreamSender{stream: stream, sessionID: in.SessionId}
	em := protocol.NewEmitter(sender, in.SessionId, turnID)

	err := l.svcCtx.Executor.Ask(stream.Context(), em, turn.AskInput{
		SessionID: in.SessionId,
		UserID:    in.UserId,
		Message:   in.Message,
		Mode:      in.Mode,
		UILang:    in.GetUiLang(),
	})

	// 最终一帧: 强制 is_final=true, 让客户端知道流结束
	_ = stream.Send(&aisolo.AskStreamResp{
		Chunk: &aisolo.AskStreamChunk{
			SessionId: in.SessionId,
			IsFinal:   true,
		},
	})
	return err
}
