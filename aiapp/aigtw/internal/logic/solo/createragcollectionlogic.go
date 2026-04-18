package solo

import (
	"context"
	"errors"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateRagCollectionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateRagCollectionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateRagCollectionLogic {
	return &CreateRagCollectionLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *CreateRagCollectionLogic) CreateRagCollection(req *types.RagCreateCollectionRequest) (*types.RagCreateCollectionResponse, error) {
	if l.svcCtx.Rag == nil {
		return nil, errors.New("rag is disabled (configure rag.enabled and embedding in aigtw.yaml)")
	}
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, errors.New("missing user id")
	}
	id, err := l.svcCtx.Rag.CreateCollection(l.ctx, uid, req.Name)
	if err != nil {
		return nil, err
	}
	return &types.RagCreateCollectionResponse{Id: id}, nil
}
