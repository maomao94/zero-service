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
	// test cmd
	cli, err := l.svcCtx.ClientManager.GetDefaultSessionClient()
	if err != nil {
		return nil, err
	}
	if err := cli.SendTestCmd(1); err != nil {
		l.Logger.Errorf("send test cmd error %v\n", err)
		return nil, err
	}
	return &ieccaller.SendTestCmdRes{}, nil
}
