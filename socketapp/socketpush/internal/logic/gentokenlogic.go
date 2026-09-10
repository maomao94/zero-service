package logic

import (
	"context"
	"zero-service/common/tool"
	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/zeromicro/go-zero/core/logx"
)

type GenTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGenTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GenTokenLogic {
	return &GenTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GenTokenLogic) GenToken(in *socketpush.GenTokenReq) (*socketpush.GenTokenRes, error) {
	accessExpire := l.svcCtx.Config.JwtAuth.AccessExpire

	accessToken, accessExpireTime, refreshAfter, err := tool.GenerateTokenByMap(
		l.svcCtx.Config.JwtAuth.AccessSecret,
		accessExpire,
		in.Payload,
	)
	if err != nil {
		return nil, err
	}

	return &socketpush.GenTokenRes{
		AccessToken:  accessToken,
		AccessExpire: accessExpireTime,
		RefreshAfter: refreshAfter,
	}, nil
}
