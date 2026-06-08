package logic

import (
	"context"
	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

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

func (l *SendInterrogationCmdLogic) SendInterrogationCmd(in *ieccaller.SendInterrogationCmdReq) (*ieccaller.SendInterrogationCmdRes, error) {
	cli, err := l.svcCtx.ClientManager.GetClientOrNil(in.Host, int(in.Port))
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_RPC, err, "获取IEC客户端失败")
	}
	if cli == nil && l.svcCtx.IsBroadcast() {
		err = l.svcCtx.PushPbBroadcast(l.ctx, ieccaller.IecCaller_SendInterrogationCmd_FullMethodName, in)
		if err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_RPC, err, "广播推送失败")
		}
	} else if cli != nil {
		if err = cli.SendInterrogationCmd(uint16(in.Coa)); err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "IEC发送总召唤失败")
		}
	}
	return nil, tool.NewErrorByPbCode(extproto.Code__1_06_RPC, "IEC客户端不存在: %s:%d", in.Host, in.Port)
}
