package solo

import (
	"context"
	"errors"
	"strings"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type BindSessionKnowledgeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBindSessionKnowledgeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BindSessionKnowledgeLogic {
	return &BindSessionKnowledgeLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *BindSessionKnowledgeLogic) BindSessionKnowledge(req *types.SoloBindKnowledgeRequest) (*types.SoloBindKnowledgeResponse, error) {
	uid := ctxdata.GetUserId(l.ctx)
	if uid == "" {
		return nil, errors.New("missing user id")
	}
	kb := strings.TrimSpace(req.KnowledgeBaseId)
	if kb == "" {
		return nil, errors.New("knowledgeBaseId is required")
	}
	resp, err := l.svcCtx.AiSoloCli.BindKnowledgeBase(l.ctx, &aisolo.BindKnowledgeBaseReq{
		SessionId:         req.SessionId,
		UserId:            uid,
		KnowledgeBaseId:   kb,
		KnowledgeBaseName: strings.TrimSpace(req.KnowledgeBaseName),
	})
	if err != nil {
		return nil, err
	}
	return &types.SoloBindKnowledgeResponse{Session: sessionToType(resp.GetSession())}, nil
}
