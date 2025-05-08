package logic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/svc"
)

type SendTestCmdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendTestCmdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendTestCmdLogic {
	return &SendTestCmdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendTestCmdLogic) SendTestCmd(in *ieccaller.SendTestCmdReq) (*ieccaller.SendTestCmdRes, error) {
	cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
	if err != nil {
		return nil, err
	}
	if err = cli.SendTestCmd(uint16(in.Coa)); err != nil {
		return nil, err
	}
	return &ieccaller.SendTestCmdRes{}, nil
}
