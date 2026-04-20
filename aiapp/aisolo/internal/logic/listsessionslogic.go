package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/svc"
)

type ListSessionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListSessionsLogic {
	return &ListSessionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListSessionsLogic) ListSessions(in *aisolo.ListSessionsReq) (*aisolo.ListSessionsResp, error) {
	page := int(in.Page)
	size := int(in.PageSize)
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	sessions, total, err := l.svcCtx.Sessions.ListSessions(l.ctx, in.UserId, page, size)
	if err != nil {
		return nil, err
	}

	out := make([]*aisolo.Session, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, toProtoSession(s))
	}

	totalPages := int32((total + int64(size) - 1) / int64(size))
	return &aisolo.ListSessionsResp{
		Sessions:   out,
		Total:      total,
		Page:       int32(page),
		TotalPages: totalPages,
	}, nil
}
