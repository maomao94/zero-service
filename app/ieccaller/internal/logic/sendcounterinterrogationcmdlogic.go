package logic

import (
	"context"

	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_RPC, err, "获取IEC客户端失败")
	}
	if cli == nil && l.svcCtx.IsBroadcast() {
		err = l.svcCtx.PushPbBroadcast(l.ctx, ieccaller.IecCaller_SendCounterInterrogationCmd_FullMethodName, in)
		if err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_RPC, err, "广播推送失败")
		}
	} else if cli != nil {
		if err = cli.SendCounterInterrogationCmd(uint16(in.Coa)); err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "IEC发送累积量召唤失败")
		}
	}
	return nil, tool.NewErrorByPbCode(extproto.Code__1_06_RPC, "IEC客户端不存在: %s:%d", in.Host, in.Port)
}
