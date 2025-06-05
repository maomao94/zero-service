package logic

import (
	"context"
	"zero-service/model"

	"zero-service/zerorpc/internal/svc"
	"zero-service/zerorpc/zerorpc"

	"github.com/zeromicro/go-zero/core/logx"
)

type EditUserInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewEditUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EditUserInfoLogic {
	return &EditUserInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 编辑用户
func (l *EditUserInfoLogic) EditUserInfo(in *zerorpc.EditUserInfoReq) (*zerorpc.EditUserInfoRes, error) {
	uId := in.GetId()
	u, err := l.svcCtx.UserModel.FindOne(l.ctx, uId)
	if err != nil {
		return nil, err
	}
	mU := model.User{
		Id:       u.Id,
		Mobile:   in.Mobile,
		Password: u.Password,
		Nickname: in.Nickname,
		Sex:      in.Sex,
		Avatar:   in.Avatar,
		Info:     u.Info,
	}
	_, err = l.svcCtx.UserModel.Update(l.ctx, nil, &mU)
	if err != nil {
		return nil, err
	}
	return &zerorpc.EditUserInfoRes{}, nil
}
