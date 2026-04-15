package solo

import (
	"context"
	"zero-service/aiapp/aisolo/aisolo"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListSessionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListSessionsLogic {
	return &ListSessionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListSessionsLogic) ListSessions(req *types.SoloListSessionsRequest) (resp *types.SoloListSessionsResponse, err error) {
	// 从 JWT context 获取用户ID
	userID := ctxdata.GetUserId(l.ctx)
	if userID == "" {
		userID = "anonymous"
	}

	protoReq := &aisolo.ListSessionsReq{
		UserId:   userID,
		Page:     int32(req.Page),
		PageSize: int32(req.PageSize),
	}

	result, err := l.svcCtx.EinoCli.ListSessions(l.ctx, protoReq)
	if err != nil {
		l.Logger.Errorf("list sessions failed: %v", err)
		return nil, err
	}

	sessions := make([]*types.SoloSessionInfo, len(result.Sessions))
	for i, s := range result.Sessions {
		sessions[i] = &types.SoloSessionInfo{
			SessionId:    s.SessionId,
			UserId:       s.UserId,
			AgentMode:    s.AgentMode.String(),
			Title:        s.Title,
			CreatedAt:    s.CreatedAt,
			UpdatedAt:    s.UpdatedAt,
			MessageCount: int(s.MessageCount),
			LastMessage:  s.LastMessage,
		}
	}

	return &types.SoloListSessionsResponse{
		Sessions: sessions,
		Total:    int(result.Total),
	}, nil
}
