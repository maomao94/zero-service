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
	if in == nil {
		return errors.New("resume request is required")
	}
	sessionID := strings.TrimSpace(in.SessionId)
	userID := strings.TrimSpace(in.UserId)
	interruptID := strings.TrimSpace(in.InterruptId)
	if sessionID == "" || interruptID == "" {
		return errors.New("session_id and interrupt_id are required")
	}
	if userID == "" {
		return errors.New("user_id is required")
	}
	if in.Action != aisolo.ResumeAction_RESUME_ACTION_YES && in.Action != aisolo.ResumeAction_RESUME_ACTION_NO {
		return errors.New("resume action must be yes or no")
	}

	turnID := uuid.NewString()
	sender := &resumeStreamSender{stream: stream, sessionID: sessionID}
	em := protocol.NewEmitter(sender, sessionID, turnID)

	err := l.svcCtx.Executor.Resume(stream.Context(), em, turn.ResumeInput{
		SessionID:   sessionID,
		UserID:      userID,
		InterruptID: interruptID,
		Action:      in.Action,
		Reason:      in.Reason,
		SelectedIDs: in.SelectedIds,
		Text:        in.Text,
		FormValues:  in.FormValues,
	})

	_ = stream.Send(&aisolo.ResumeStreamResp{
		Chunk: &aisolo.ResumeStreamChunk{
			SessionId: sessionID,
			IsFinal:   true,
		},
	})
	return err
}
