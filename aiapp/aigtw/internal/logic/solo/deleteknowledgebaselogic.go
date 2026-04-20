package solo

import (
	"context"
	"errors"

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
		return nil, errors.New("knowledge is disabled")
	}
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, errors.New("missing user id")
	}
	if err := l.svcCtx.Knowledge.DeleteBase(l.ctx, uid, req.BaseId); err != nil {
		return nil, err
	}
	return &types.KnowledgeDeleteBaseResponse{Success: true}, nil
}
