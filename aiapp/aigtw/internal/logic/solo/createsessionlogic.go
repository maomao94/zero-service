package solo

import (
	"context"
	"zero-service/aiapp/aisolo/aisolo"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSessionLogic {
	return &CreateSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateSessionLogic) CreateSession(req *types.SoloCreateSessionRequest) (resp *types.SoloCreateSessionResponse, err error) {
	// 从 JWT context 获取用户ID
	userID := ctxdata.GetUserId(l.ctx)
	if userID == "" {
		userID = "anonymous"
	}

	protoReq := &aisolo.CreateSessionReq{
		Title:  req.Title,
		UserId: userID,
	}

	result, err := l.svcCtx.AiSoloCli.CreateSession(l.ctx, protoReq)
	if err != nil {
		l.Logger.Errorf("create session failed: %v", err)
		return nil, err
	}

	return &types.SoloCreateSessionResponse{
		Session: &types.SoloSessionInfo{
			SessionId:    result.Session.SessionId,
			UserId:       result.Session.UserId,
			AgentMode:    result.Session.AgentMode.String(),
			Title:        result.Session.Title,
			CreatedAt:    result.Session.CreatedAt,
			UpdatedAt:    result.Session.UpdatedAt,
			MessageCount: int(result.Session.MessageCount),
			LastMessage:  result.Session.LastMessage,
		},
	}, nil
}
