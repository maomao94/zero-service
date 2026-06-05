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

type SendStepCommandLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendStepCommandLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendStepCommandLogic {
	return &SendStepCommandLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendStepCommandLogic) SendStepCommand(in *ieccaller.SendStepCommandReq) (*ieccaller.SendStepCommandRes, error) {
	cli, err := l.svcCtx.ClientManager.GetClientOrNil(in.Host, int(in.Port))
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_RPC, err, "获取IEC客户端失败")
	}
	if cli == nil && l.svcCtx.IsBroadcast() {
		var res ieccaller.SendStepCommandRes
		err = l.svcCtx.PushPbBroadcastWithAck(l.ctx, ieccaller.IecCaller_SendStepCommand_FullMethodName, in, &res)
		if err != nil {
			return nil, wrapCommandAckError(err, "集群推送ACK失败")
		}
		return &res, nil
	} else if cli != nil {
		ack, err := cli.SendStepCmd(l.ctx, uint16(in.Coa), asdu.InfoObjAddr(in.Ioa), asdu.StepCommand(in.Value), in.WithTime, client.WithAck())
		if err != nil {
			return nil, wrapCommandAckError(err, "IEC发送档位命令失败")
		}
		value, err := ackStepCommandValue(ack.Value)
		if err != nil {
			return nil, wrapCommandAckError(err, "IEC档位命令ACK解析失败")
		}
		return &ieccaller.SendStepCommandRes{Value: int32(value)}, nil
	} else {
		logx.Errorf("cli is empty")
	}
	return &ieccaller.SendStepCommandRes{}, nil
}
