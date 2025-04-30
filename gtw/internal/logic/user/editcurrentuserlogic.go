package user

import (
	"context"
	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"
	"zero-service/zerorpc/zerorpc"

	"github.com/zeromicro/go-zero/core/logx"
)

type EditCurrentUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 修改当前用户信息
func NewEditCurrentUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EditCurrentUserLogic {
	return &EditCurrentUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EditCurrentUserLogic) EditCurrentUser(req *types.EditCurrentUserRequest) (resp *types.EditCurrentUserReply, err error) {
	gL := NewGetCurrentUserLogic(l.ctx, l.svcCtx)
	u, err := gL.GetCurrentUser(&types.GetCurrentUserRequest{})
	if err != nil {
		return nil, err
	}
	l.svcCtx.ZeroRpcCli.EditUserInfo(l.ctx, &zerorpc.EditUserInfoReq{
		Id:       u.User.Id,
		Mobile:   u.User.Mobile,
		Nickname: req.Nickname,
		Sex:      req.Sex,
		Avatar:   req.Avatar,
	})
	return
}
