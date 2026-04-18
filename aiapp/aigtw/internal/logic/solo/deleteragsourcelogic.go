package solo

import (
	"context"
	"errors"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteRagSourceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteRagSourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteRagSourceLogic {
	return &DeleteRagSourceLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *DeleteRagSourceLogic) DeleteRagSource(req *types.RagDeleteSourceRequest) (*types.RagDeleteSourceResponse, error) {
	if l.svcCtx.Rag == nil {
		return nil, errors.New("rag is disabled")
	}
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, errors.New("missing user id")
	}
	if err := l.svcCtx.Rag.DeleteSource(l.ctx, uid, req.CollectionId, req.SourceId); err != nil {
		return nil, err
	}
	return &types.RagDeleteSourceResponse{Success: true}, nil
}
