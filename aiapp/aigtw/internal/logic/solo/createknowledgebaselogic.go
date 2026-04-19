package solo

import (
	"context"
	"errors"
	"strings"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateKnowledgeBaseLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateKnowledgeBaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateKnowledgeBaseLogic {
	return &CreateKnowledgeBaseLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *CreateKnowledgeBaseLogic) CreateKnowledgeBase(req *types.KnowledgeCreateBaseRequest) (*types.KnowledgeCreateBaseResponse, error) {
	if l.svcCtx.Knowledge == nil {
		return nil, errors.New("knowledge is disabled (configure knowledge.enabled and embedding in aigtw.yaml)")
	}
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, errors.New("missing user id")
	}
	id, err := l.svcCtx.Knowledge.CreateBase(l.ctx, uid, strings.TrimSpace(req.Name))
	if err != nil {
		return nil, err
	}
	return &types.KnowledgeCreateBaseResponse{Id: id}, nil
}
