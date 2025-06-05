package logic

import (
	"context"
	"github.com/songzhibin97/gkit/errors"

	"zero-service/zerorpc/internal/svc"
	"zero-service/zerorpc/zerorpc"

	"github.com/zeromicro/go-zero/core/logx"
)

type MiniProgramLoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMiniProgramLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MiniProgramLoginLogic {
	return &MiniProgramLoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 小程序登录
func (l *MiniProgramLoginLogic) MiniProgramLogin(in *zerorpc.MiniProgramLoginReq) (*zerorpc.MiniProgramLoginRes, error) {
	se, err := l.svcCtx.MiniCli.Auth.Session(l.ctx, in.Code)
	if err != nil {
		return nil, err
	}
	if se.ErrCode != 0 {
		l.WithContext(l.ctx).Errorf("小程序登录失败 errCode:%v", se.ErrCode)
		return nil, errors.BadRequest("9999", "小程序登录失败")
	}
	return &zerorpc.MiniProgramLoginRes{
		OpenId:     se.OpenID,
		UnionId:    se.UnionID,
		SessionKey: se.SessionKey,
	}, nil
}
