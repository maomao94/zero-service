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

func (l *CreateSessionLogic) CreateSession(req *types.SoloCreateSessionReq) (resp *types.SoloCreateSessionResp, err error) {
	// 从 JWT context 获取用户ID
	userID := ctxdata.GetUserId(l.ctx)
	if userID == "" {
		userID = "anonymous"
	}

	protoReq := &aisolo.CreateSessionRequest{
		Title:  req.Title,
		UserId: userID,
	}

	result, err := l.svcCtx.EinoCli.CreateSession(l.ctx, protoReq)
	if err != nil {
		l.Logger.Errorf("create session failed: %v", err)
		return nil, err
	}

	return &types.SoloCreateSessionResp{
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
