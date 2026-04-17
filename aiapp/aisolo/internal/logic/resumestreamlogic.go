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

type ResumeStreamLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResumeStreamLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResumeStreamLogic {
	return &ResumeStreamLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ResumeStream 流式恢复中断的执行。
func (l *ResumeStreamLogic) ResumeStream(in *aisolo.ResumeReq, stream aisolo.AiSolo_ResumeStreamServer) error {
	if l.svcCtx.Executor == nil {
		return errors.New("executor not ready (chat model may be missing)")
	}
	if in.SessionId == "" || in.InterruptId == "" {
		return errors.New("session_id and interrupt_id are required")
	}

	turnID := uuid.NewString()
	sender := &resumeStreamSender{stream: stream, sessionID: in.SessionId}
	em := protocol.NewEmitter(sender, in.SessionId, turnID)

	err := l.svcCtx.Executor.Resume(stream.Context(), em, turn.ResumeInput{
		SessionID:   in.SessionId,
		UserID:      in.UserId,
		InterruptID: in.InterruptId,
		Action:      in.Action,
		Reason:      in.Reason,
		SelectedIDs: in.SelectedIds,
		Text:        in.Text,
		FormValues:  in.FormValues,
	})

	_ = stream.Send(&aisolo.ResumeStreamResp{
		Chunk: &aisolo.ResumeStreamChunk{
			SessionId: in.SessionId,
			IsFinal:   true,
		},
	})
	return err
}
