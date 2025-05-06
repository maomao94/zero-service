package logic

import (
	"context"
	"fmt"
	"zero-service/iec104/iec104client"

	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendReadCmdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendReadCmdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendReadCmdLogic {
	return &SendReadCmdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendReadCmdLogic) SendReadCmd(in *ieccaller.SendReadCmdReq) (*ieccaller.SendReadCmdRes, error) {
	cli := l.svcCtx.ClientManager.GetSession(iec104client.CoaConfig{
		Host: in.Host,
		Port: int(in.Port),
		Coa:  int(in.Coa),
	})
	if cli == nil {
		return nil, fmt.Errorf("cli is empty")
	}
	if err := cli.SendReadCmd(uint16(in.Coa), uint(in.Ioa)); err != nil {
		return nil, err
	}
	return &ieccaller.SendReadCmdRes{}, nil
}
