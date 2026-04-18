package solo

import (
	"context"
	"errors"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/modeweb"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewCreateSessionLogic 创建会话 Logic。
func NewCreateSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSessionLogic {
	return &CreateSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// CreateSession 透传到 aisolo gRPC, 用户 ID 从 JWT 解析出的 ctx 拿。
func (l *CreateSessionLogic) CreateSession(req *types.SoloCreateSessionRequest) (*types.SoloCreateSessionResponse, error) {
	userID := ctxdata.GetUserId(l.ctx)
	if userID == "" {
		return nil, errors.New("missing user id in context")
	}
	resp, err := l.svcCtx.AiSoloCli.CreateSession(l.ctx, &aisolo.CreateSessionReq{
		UserId: userID,
		Title:  req.Title,
		Mode:   modeweb.Parse(req.Mode),
		UiLang: req.UiLang,
	})
	if err != nil {
		return nil, err
	}
	return &types.SoloCreateSessionResponse{Session: sessionToType(resp.GetSession())}, nil
}
