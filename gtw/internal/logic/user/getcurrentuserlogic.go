package user

import (
	"context"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"zero-service/common/ctxdata"
	"zero-service/zerorpc/zerorpc"

	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCurrentUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取用户信息
func NewGetCurrentUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCurrentUserLogic {
	return &GetCurrentUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCurrentUserLogic) GetCurrentUser(req *types.GetCurrentUserRequest) (resp *types.GetCurrentUserReply, err error) {
	userId := ctxdata.GetUserIdFromCtx(l.ctx, true)
	if userId > 0 {
		res, err := l.svcCtx.ZeroRpcCli.GetUserInfo(l.ctx, &zerorpc.GetUserInfoReq{
			Id: userId,
		})
		if err != nil {
			return nil, err
		}
		var user types.User
		_ = copier.Copy(&user, res.User)
		return &types.GetCurrentUserReply{
			User: user,
		}, nil
	} else {
		return nil, errors.New("获取用户错误")
	}
	return
}
