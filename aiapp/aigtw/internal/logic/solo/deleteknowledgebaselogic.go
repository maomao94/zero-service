package solo

import (
	"context"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteKnowledgeBaseLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteKnowledgeBaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteKnowledgeBaseLogic {
	return &DeleteKnowledgeBaseLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *DeleteKnowledgeBaseLogic) DeleteKnowledgeBase(req *types.KnowledgeDeleteBaseRequest) (*types.KnowledgeDeleteBaseResponse, error) {
	if l.svcCtx.Knowledge == nil {
		return nil, invalidRequestError("knowledge is disabled")
	}
	if req == nil {
		return nil, invalidRequestError("delete knowledge base request is required")
	}
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, unauthenticatedError("missing user id")
	}
	baseID, err := requireKnowledgeBaseID(req.BaseId)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.Knowledge.DeleteBase(l.ctx, uid, baseID); err != nil {
		return nil, err
	}
	return &types.KnowledgeDeleteBaseResponse{Success: true}, nil
}
