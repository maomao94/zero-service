package logic

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/svc"
)

type BindKnowledgeBaseLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBindKnowledgeBaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BindKnowledgeBaseLogic {
	return &BindKnowledgeBaseLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *BindKnowledgeBaseLogic) BindKnowledgeBase(in *aisolo.BindKnowledgeBaseReq) (*aisolo.BindKnowledgeBaseResp, error) {
	if in.GetUserId() == "" || in.GetSessionId() == "" {
		return nil, errors.New("user_id and session_id are required")
	}
	kb := strings.TrimSpace(in.GetKnowledgeBaseId())
	if kb == "" {
		return nil, errors.New("knowledge_base_id is required")
	}
	sess, err := l.svcCtx.Sessions.GetSession(l.ctx, in.GetUserId(), in.GetSessionId())
	if err != nil {
		return nil, err
	}
	sess.KnowledgeBaseID = kb
	sess.KnowledgeBaseName = strings.TrimSpace(in.GetKnowledgeBaseName())
	sess.UpdatedAt = time.Now()
	if err := l.svcCtx.Sessions.UpdateSession(l.ctx, sess); err != nil {
		return nil, err
	}
	return &aisolo.BindKnowledgeBaseResp{Session: toProtoSession(sess)}, nil
}
