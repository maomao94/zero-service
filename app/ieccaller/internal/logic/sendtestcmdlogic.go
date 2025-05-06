package logic

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/iec104/iec104client"
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
	cli := l.svcCtx.ClientManager.GetSession(iec104client.CoaConfig{
		Host: in.Host,
		Port: int(in.Port),
		Coa:  int(in.Coa),
	})
	if cli == nil {
		return nil, fmt.Errorf("cli is empty")
	}
	if err := cli.SendTestCmd(uint16(in.Coa)); err != nil {
		return nil, err
	}
	return &ieccaller.SendTestCmdRes{}, nil
}
