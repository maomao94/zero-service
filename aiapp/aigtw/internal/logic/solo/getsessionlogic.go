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

func (l *GetSessionLogic) GetSession(req *types.SoloGetSessionReq) (resp *types.SoloGetSessionResp, err error) {
	protoReq := &aisolo.SessionRequest{
		SessionId: req.SessionId,
	}

	result, err := l.svcCtx.EinoCli.GetSession(l.ctx, protoReq)
	if err != nil {
		l.Logger.Errorf("get session failed: %v", err)
		return nil, err
	}

	return &types.SoloGetSessionResp{
		Session: &types.SoloSessionInfo{
			SessionId:    result.SessionId,
			UserId:       result.UserId,
			AgentMode:    result.AgentMode.String(),
			Title:        result.Title,
			CreatedAt:    result.CreatedAt,
			UpdatedAt:    result.UpdatedAt,
			MessageCount: int(result.MessageCount),
			LastMessage:  result.LastMessage,
		},
	}, nil
}
