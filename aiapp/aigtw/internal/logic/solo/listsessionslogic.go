package solo

import (
	"context"
	"errors"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/aiapp/aisolo/aisolo"
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

func (l *ListSessionsLogic) ListSessions(req *types.SoloListSessionsRequest) (*types.SoloListSessionsResponse, error) {
	userID := ctxdata.GetUserId(l.ctx)
	if userID == "" {
		return nil, errors.New("missing user id in context")
	}
	resp, err := l.svcCtx.AiSoloCli.ListSessions(l.ctx, &aisolo.ListSessionsReq{
		UserId:   userID,
		Page:     int32(req.Page),
		PageSize: int32(req.PageSize),
	})
	if err != nil {
		return nil, err
	}
	out := &types.SoloListSessionsResponse{
		Total:      int(resp.GetTotal()),
		Page:       int(resp.GetPage()),
		TotalPages: int(resp.GetTotalPages()),
	}
	for _, s := range resp.GetSessions() {
		out.Sessions = append(out.Sessions, sessionToType(s))
	}
	return out, nil
}
