package user

import (
	"context"
	"zero-service/zerorpc/zerorpc"

	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 登录
func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginRequest) (resp *types.LoginReply, err error) {
	res, err := l.svcCtx.ZeroRpcCli.Login(l.ctx, &zerorpc.LoginReq{
		AuthType: req.AuthType,
		AuthKey:  req.AuthKey,
		Password: req.Password,
	})
	if err != nil {
		return nil, err
	}
	return &types.LoginReply{
		AccessToken:  res.AccessToken,
		AccessExpire: res.AccessExpire,
		RefreshAfter: res.RefreshAfter,
	}, nil
}
