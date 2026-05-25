package logic

import (
	"context"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"
)

type DeleteSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSessionLogic {
	return &DeleteSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteSessionLogic) DeleteSession(in *aisolo.DeleteSessionReq) (*aisolo.DeleteSessionResp, error) {
	userID := strings.TrimSpace(in.GetUserId())
	sessionID := strings.TrimSpace(in.GetSessionId())
	if userID == "" || sessionID == "" {
		return &aisolo.DeleteSessionResp{Success: false}, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "user_id and session_id are required")
	}
	sess, err := l.svcCtx.Sessions.GetSession(l.ctx, userID, sessionID)
	if err != nil {
		return &aisolo.DeleteSessionResp{Success: false}, err
	}
	if sess.Status == aisolo.SessionStatus_SESSION_STATUS_RUNNING {
		return &aisolo.DeleteSessionResp{Success: false}, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ_STATE, "cannot delete running session")
	}
	if l.svcCtx.Messages != nil {
		if err := l.svcCtx.Messages.DeleteSession(l.ctx, userID, sessionID); err != nil {
			return &aisolo.DeleteSessionResp{Success: false}, err
		}
	}
	if err := l.svcCtx.Sessions.DeleteSession(l.ctx, userID, sessionID); err != nil {
		return &aisolo.DeleteSessionResp{Success: false}, err
	}
	return &aisolo.DeleteSessionResp{Success: true}, nil
}
