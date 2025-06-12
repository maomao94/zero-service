package logic

import (
	"context"
	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendInterrogationCmdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendInterrogationCmdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendInterrogationCmdLogic {
	return &SendInterrogationCmdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 发送总召唤
func (l *SendInterrogationCmdLogic) SendInterrogationCmd(in *ieccaller.SendInterrogationCmdReq) (*ieccaller.SendInterrogationCmdRes, error) {
	cli, err := l.svcCtx.ClientManager.GetClientOrNil(in.Host, int(in.Port))
	if err != nil {
		return nil, err
	}
	if cli == nil {
		err = l.svcCtx.PushPbBroadcast(ieccaller.IecCaller_SendInterrogationCmd_FullMethodName, in)
		return nil, err
	} else {
		if err = cli.SendInterrogationCmd(uint16(in.Coa)); err != nil {
			return nil, err
		}
	}
	return &ieccaller.SendInterrogationCmdRes{}, nil
}
