package logic

import (
	"context"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"
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
	if in == nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "bind knowledge base request is required")
	}
	userID := strings.TrimSpace(in.GetUserId())
	sessionID := strings.TrimSpace(in.GetSessionId())
	if userID == "" || sessionID == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "user_id and session_id are required")
	}
	kb := strings.TrimSpace(in.GetKnowledgeBaseId())
	if kb == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "knowledge_base_id is required")
	}
	sess, err := l.svcCtx.Sessions.GetSession(l.ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.Status == aisolo.SessionStatus_SESSION_STATUS_RUNNING {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "cannot bind knowledge base while session is running")
	}
	sess.KnowledgeBaseID = kb
	sess.KnowledgeBaseName = strings.TrimSpace(in.GetKnowledgeBaseName())
	sess.UpdatedAt = time.Now()
	if err := l.svcCtx.Sessions.UpdateSession(l.ctx, sess); err != nil {
		return nil, err
	}
	return &aisolo.BindKnowledgeBaseResp{Session: toProtoSession(sess)}, nil
}
