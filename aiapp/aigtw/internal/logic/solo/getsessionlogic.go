package solo

import (
	"context"
	"zero-service/aiapp/aisolo/aisolo"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSessionLogic {
	return &GetSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSessionLogic) GetSession(req *types.SoloGetSessionRequest) (resp *types.SoloGetSessionResponse, err error) {
	protoReq := &aisolo.GetSessionReq{
		SessionId: req.SessionId,
	}

	result, err := l.svcCtx.AiSoloCli.GetSession(l.ctx, protoReq)
	if err != nil {
		l.Logger.Errorf("get session failed: %v", err)
		return nil, err
	}

	return &types.SoloGetSessionResponse{
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
