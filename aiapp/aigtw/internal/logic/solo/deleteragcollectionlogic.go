package solo

import (
	"context"
	"errors"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteRagCollectionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteRagCollectionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteRagCollectionLogic {
	return &DeleteRagCollectionLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *DeleteRagCollectionLogic) DeleteRagCollection(req *types.RagDeleteCollectionRequest) (*types.RagDeleteCollectionResponse, error) {
	if l.svcCtx.Rag == nil {
		return nil, errors.New("rag is disabled")
	}
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, errors.New("missing user id")
	}
	if err := l.svcCtx.Rag.DeleteCollection(l.ctx, uid, req.CollectionId); err != nil {
		return nil, err
	}
	return &types.RagDeleteCollectionResponse{Success: true}, nil
}
