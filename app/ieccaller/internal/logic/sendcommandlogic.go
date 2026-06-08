package logic

import (
	"context"

	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/wendy512/go-iecp5/asdu"
	"github.com/zeromicro/go-zero/core/logx"
)

type SendCommandLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendCommandLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendCommandLogic {
	return &SendCommandLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 发送命令
func (l *SendCommandLogic) SendCommand(in *ieccaller.SendCommandReq) (*ieccaller.SendCommandRes, error) {
	cli, err := l.svcCtx.ClientManager.GetClientOrNil(in.Host, int(in.Port))
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_RPC, err, "获取IEC客户端失败")
	}
	if cli == nil && l.svcCtx.IsBroadcast() {
		err = l.svcCtx.PushPbBroadcast(l.ctx, ieccaller.IecCaller_SendCommand_FullMethodName, in)
		if err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_RPC, err, "广播推送失败")
		}
	} else if cli != nil {
		if err = cli.SendCmd(uint16(in.Coa), asdu.TypeID(in.TypeId), asdu.InfoObjAddr(in.Ioa), in.Value); err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "IEC发送命令失败")
		}
	}
	return nil, tool.NewErrorByPbCode(extproto.Code__1_06_RPC, "IEC客户端不存在: %s:%d", in.Host, in.Port)
}
