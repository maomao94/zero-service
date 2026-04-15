package solo

import (
	"context"
	"zero-service/aiapp/aisolo/aisolo"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSessionLogic {
	return &DeleteSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteSessionLogic) DeleteSession(req *types.SoloDeleteSessionRequest) (resp *types.SoloDeleteSessionResponse, err error) {
	protoReq := &aisolo.DeleteSessionReq{
		SessionId: req.SessionId,
	}

	result, err := l.svcCtx.EinoCli.DeleteSession(l.ctx, protoReq)
	if err != nil {
		l.Logger.Errorf("delete session failed: %v", err)
		return nil, err
	}

	return &types.SoloDeleteSessionResponse{Success: result.Success}, nil
}
