package logic

import (
	"context"

	"zero-service/zerorpc/internal/svc"
	"zero-service/zerorpc/zerorpc"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserInfoLogic {
	return &GetUserInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 用户详情
func (l *GetUserInfoLogic) GetUserInfo(in *zerorpc.GetUserInfoReq) (*zerorpc.GetUserInfoRes, error) {
	userId, err := convertor.ToInt(in.GetId())
	if err != nil {
		return nil, err
	}
	u, err := l.svcCtx.UserModel.FindOne(l.ctx, userId)
	if err != nil {
		return nil, err
	}
	return &zerorpc.GetUserInfoRes{
		User: &zerorpc.User{
			Id:       u.Id,
			Mobile:   u.Mobile,
			Nickname: u.Nickname,
			Sex:      u.Sex,
			Avatar:   u.Avatar,
		},
	}, nil
}
