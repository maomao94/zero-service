package logic

import (
	"context"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ResumeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResumeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResumeLogic {
	return &ResumeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ResumeLogic) Resume(in *aisolo.ResumeReq) (*aisolo.ResumeResp, error) {
	sessionID := in.SessionId
	userID := in.UserId
	interruptID := in.InterruptId

	l.Infof("Resume: session=%s, user=%s, interrupt=%s, action=%v",
		sessionID, userID, interruptID, in.Action)

	if interruptID == "" {
		return nil, status.Error(codes.InvalidArgument, "interrupt_id is required")
	}

	// sessionID 可以从 interruptID 关联获取，允许为空
	if userID == "" {
		userID = "anonymous"
	}

	l.Logger.Errorf("Resume not fully implemented: checkpoint storage not available")
	return &aisolo.ResumeResp{
		SessionId: sessionID,
		Success:   false,
		Message:   "checkpoint storage not available",
	}, nil
}
