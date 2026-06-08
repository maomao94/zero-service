package logic

import (
	"context"

	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/common/iec104/client"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/wendy512/go-iecp5/asdu"
	"github.com/zeromicro/go-zero/core/logx"
)

type SendDoubleCommandLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendDoubleCommandLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendDoubleCommandLogic {
	return &SendDoubleCommandLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendDoubleCommandLogic) SendDoubleCommand(in *ieccaller.SendDoubleCommandReq) (*ieccaller.SendDoubleCommandRes, error) {
	cli, err := l.svcCtx.ClientManager.GetClientOrNil(in.Host, int(in.Port))
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_RPC, err, "获取IEC客户端失败")
	}
	if cli == nil && l.svcCtx.IsBroadcast() {
		var res ieccaller.SendDoubleCommandRes
		err = l.svcCtx.PushPbBroadcastWithAck(l.ctx, ieccaller.IecCaller_SendDoubleCommand_FullMethodName, in, &res)
		if err != nil {
			return nil, wrapCommandAckError(err, "集群推送ACK失败")
		}
		return &res, nil
	} else if cli != nil {
		ack, err := cli.SendDoubleCmd(l.ctx, uint16(in.Coa), asdu.InfoObjAddr(in.Ioa), asdu.DoubleCommand(in.Value), in.WithTime, client.WithAck())
		if err != nil {
			return nil, wrapCommandAckError(err, "IEC发送双点命令失败")
		}
		value, err := ackDoubleCommandValue(ack.Value)
		if err != nil {
			return nil, wrapCommandAckError(err, "IEC双点命令ACK解析失败")
		}
		return &ieccaller.SendDoubleCommandRes{Value: ieccaller.DoubleCommandValue(int32(value))}, nil
	}
	return nil, tool.NewErrorByPbCode(extproto.Code__1_06_RPC, "IEC客户端不存在: %s:%d", in.Host, in.Port)
}
