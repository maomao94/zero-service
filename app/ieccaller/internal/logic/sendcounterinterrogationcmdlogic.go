package logic

import (
	"context"

	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendCounterInterrogationCmdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendCounterInterrogationCmdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendCounterInterrogationCmdLogic {
	return &SendCounterInterrogationCmdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 累积量召唤
func (l *SendCounterInterrogationCmdLogic) SendCounterInterrogationCmd(in *ieccaller.SendCounterInterrogationCmdReq) (*ieccaller.SendCounterInterrogationCmdRes, error) {
	cli, err := l.svcCtx.ClientManager.GetClientOrNil(in.Host, int(in.Port))
	if err != nil {
		return nil, err
	}
	if cli == nil && l.svcCtx.IsBroadcast() {
		err = l.svcCtx.PushPbBroadcast(l.ctx, ieccaller.IecCaller_SendCounterInterrogationCmd_FullMethodName, in)
		return nil, err
	} else if cli != nil {
		if err = cli.SendCounterInterrogationCmd(uint16(in.Coa)); err != nil {
			return nil, err
		}
	} else {
		logx.Errorf("cli is empty")
	}
	return &ieccaller.SendCounterInterrogationCmdRes{}, nil
}
