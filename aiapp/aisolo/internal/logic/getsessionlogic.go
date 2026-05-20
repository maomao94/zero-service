package logic

import (
	"context"
	"errors"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/svc"
)

type GetSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSessionLogic {
	return &GetSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetSessionLogic) GetSession(in *aisolo.GetSessionReq) (*aisolo.GetSessionResp, error) {
	userID := strings.TrimSpace(in.GetUserId())
	sessionID := strings.TrimSpace(in.GetSessionId())
	if userID == "" || sessionID == "" {
		return nil, errors.New("user_id and session_id are required")
	}
	sess, err := l.svcCtx.Sessions.GetSession(l.ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	return &aisolo.GetSessionResp{Session: toProtoSession(sess)}, nil
}
