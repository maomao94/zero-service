package logic

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/sessionworkdir"
	"zero-service/aiapp/aisolo/internal/svc"
	"zero-service/aiapp/aisolo/internal/uilang"
)

type CreateSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSessionLogic {
	return &CreateSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// CreateSession 新建一个会话 (用户只挑 mode, 无需再指定 agent)。
func (l *CreateSessionLogic) CreateSession(in *aisolo.CreateSessionReq) (*aisolo.CreateSessionResp, error) {
	if in.UserId == "" {
		return nil, errors.New("user_id is required")
	}
	sess := newSession(in.UserId, in.Title, in.Mode)
	if v := uilang.Normalize(in.GetUiLang()); v != "" {
		sess.UILang = v
	}
	if err := l.svcCtx.Sessions.CreateSession(l.ctx, sess); err != nil {
		return nil, err
	}
	if err := sessionworkdir.EnsureSession(l.svcCtx.Config, sess.ID); err != nil {
		l.Errorf("session workspace: %v", err)
	}
	return &aisolo.CreateSessionResp{Session: toProtoSession(sess)}, nil
}
