package solo

import (
	"context"
	"strings"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/common/ctxdata"

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

func (l *DeleteSessionLogic) DeleteSession(req *types.SoloDeleteSessionRequest) (*types.SoloDeleteSessionResponse, error) {
	userID := ctxdata.GetUserId(l.ctx)
	if userID == "" {
		return nil, unauthenticatedError("missing user id in context")
	}
	if req == nil {
		return nil, invalidRequestError("delete session request is required")
	}
	sessionID := strings.TrimSpace(req.SessionId)
	if sessionID == "" {
		return nil, invalidRequestError("sessionId is required")
	}
	resp, err := l.svcCtx.AiSoloCli.DeleteSession(l.ctx, &aisolo.DeleteSessionReq{
		SessionId: sessionID,
		UserId:    userID,
	})
	if err != nil {
		return nil, err
	}
	return &types.SoloDeleteSessionResponse{Success: resp.GetSuccess()}, nil
}
