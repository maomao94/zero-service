package solo

import (
	"context"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListKnowledgeBasesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListKnowledgeBasesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListKnowledgeBasesLogic {
	return &ListKnowledgeBasesLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ListKnowledgeBasesLogic) ListKnowledgeBases() (*types.KnowledgeListBasesResponse, error) {
	if l.svcCtx.Knowledge == nil {
		return nil, invalidRequestError("knowledge is disabled")
	}
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, unauthenticatedError("missing user id")
	}
	list, err := l.svcCtx.Knowledge.ListBases(l.ctx, uid)
	if err != nil {
		return nil, err
	}
	out := make([]types.KnowledgeBaseInfo, 0, len(list))
	for _, c := range list {
		out = append(out, types.KnowledgeBaseInfo{Id: c.ID, Name: c.Name, CreatedAt: c.CreatedAt.Unix()})
	}
	return &types.KnowledgeListBasesResponse{Bases: out}, nil
}
