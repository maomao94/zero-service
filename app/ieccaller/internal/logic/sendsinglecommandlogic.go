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

type SendSingleCommandLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendSingleCommandLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendSingleCommandLogic {
	return &SendSingleCommandLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendSingleCommandLogic) SendSingleCommand(in *ieccaller.SendSingleCommandReq) (*ieccaller.SendSingleCommandRes, error) {
	cli, err := l.svcCtx.ClientManager.GetClientOrNil(in.Host, int(in.Port))
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_RPC, err, "获取IEC客户端失败")
	}
	if cli == nil && l.svcCtx.IsBroadcast() {
		var res ieccaller.SendSingleCommandRes
		err = l.svcCtx.PushPbBroadcastWithAck(l.ctx, ieccaller.IecCaller_SendSingleCommand_FullMethodName, in, &res)
		if err != nil {
			return nil, wrapCommandAckError(err, "集群推送ACK失败")
		}
		return &res, nil
	} else if cli != nil {
		ack, err := cli.SendSingleCmd(l.ctx, uint16(in.Coa), asdu.InfoObjAddr(in.Ioa), in.Value, in.WithTime, client.WithAck())
		if err != nil {
			return nil, wrapCommandAckError(err, "IEC发送单点命令失败")
		}
		value, err := ackBoolValue(ack.Value)
		if err != nil {
			return nil, wrapCommandAckError(err, "IEC单点命令ACK解析失败")
		}
		return &ieccaller.SendSingleCommandRes{Value: value}, nil
	} else {
		logx.Errorf("cli is empty")
	}
	return &ieccaller.SendSingleCommandRes{}, nil
}
