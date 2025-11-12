package logic

import (
	"context"
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
	cli, err := l.svcCtx.ClientManager.GetClientOrNil(in.Host, int(in.Port))
	if err != nil {
		return nil, err
	}
	if cli == nil && l.svcCtx.IsBroadcast() {
		err = l.svcCtx.PushPbBroadcast(ieccaller.IecCaller_SendReadCmd_FullMethodName, in)
		if err != nil {
			return nil, err
		}
	} else if cli != nil {
		if err = cli.SendReadCmd(uint16(in.Coa), uint(in.Ioa)); err != nil {
			return nil, err
		}
	} else {
		logx.Errorf("cli is empty")
	}
	return &ieccaller.SendReadCmdRes{}, nil
}
