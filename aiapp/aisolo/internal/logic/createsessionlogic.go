package logic

import (
	"context"

	"zero-service/aiapp/aisolo/aisolo"

	"zero-service/aiapp/aisolo/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// CreateSession 创建会话
func (l *CreateSessionLogic) CreateSession(in *aisolo.CreateSessionReq) (*aisolo.CreateSessionResp, error) {
	// 验证必填参数
	userID := in.UserId
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// 创建会话
	session, err := GlobalSessionStore.Create(l.ctx, userID)
	if err != nil {
		l.Errorf("create session failed: %v", err)
		return nil, err
	}

	// 返回包装响应
	return &aisolo.CreateSessionResp{
		Session: &aisolo.Session{
			SessionId:    session.SessionId,
			UserId:       session.UserId,
			Title:        in.Title,
			AgentMode:    in.AgentMode,
			CreatedAt:    session.CreatedAt,
			UpdatedAt:    session.UpdatedAt,
			MessageCount: session.MessageCount,
		},
	}, nil
}
