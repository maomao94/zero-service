package logic

import (
	"context"
	"encoding/json"
	"zero-service/common/tool"
	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type VerifyTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewVerifyTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VerifyTokenLogic {
	return &VerifyTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 验证token
func (l *VerifyTokenLogic) VerifyToken(in *socketpush.VerifyTokenReq) (*socketpush.VerifyTokenRes, error) {
	if len(in.AccessToken) == 0 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "access token is empty")
	}
	secrets := []string{l.svcCtx.Config.JwtAuth.AccessSecret}
	if len(l.svcCtx.Config.JwtAuth.PrevAccessSecret) > 0 {
		secrets = append(secrets, l.svcCtx.Config.JwtAuth.PrevAccessSecret)
	}
	claims, err := tool.ParseToken(in.AccessToken, secrets...)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_03_UNAUTHORIZED, err, "invalid access token")
	}
	claimJSON, err := json.Marshal(claims)
	if err != nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_00_INTERNAL, "failed to process token claims")
	}
	res := &socketpush.VerifyTokenRes{
		ClaimJson: string(claimJSON),
	}
	return res, nil
}
