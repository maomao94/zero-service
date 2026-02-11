package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"zero-service/common/tool"
	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

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
		return nil, fmt.Errorf("access token is empty")
	}
	secrets := []string{l.svcCtx.Config.JwtAuth.AccessSecret}
	if len(l.svcCtx.Config.JwtAuth.PrevAccessSecret) > 0 {
		secrets = append(secrets, l.svcCtx.Config.JwtAuth.PrevAccessSecret)
	}
	claims, err := tool.ParseToken(in.AccessToken, secrets...)
	if err != nil {
		return nil, fmt.Errorf("invalid access token: %v", err)
	}
	claimJSON, err := json.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to process token claims: %v", err)
	}
	res := &socketpush.VerifyTokenRes{
		ClaimJson: string(claimJSON),
	}
	return res, nil
}
