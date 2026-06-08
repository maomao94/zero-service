package logic

import (
	"context"

	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/common/iec104/client"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/duke-git/lancet/v2/validator"
	"github.com/wendy512/go-iecp5/asdu"
	"github.com/zeromicro/go-zero/core/logx"
)

type SendSetpointFloatLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendSetpointFloatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendSetpointFloatLogic {
	return &SendSetpointFloatLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendSetpointFloatLogic) SendSetpointFloat(in *ieccaller.SendSetpointFloatReq) (*ieccaller.SendSetpointFloatRes, error) {
	if !validator.IsNumberStr(in.Value) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_06_RPC, "浮点设点值格式无效: %s", in.Value)
	}
	fv, err := convertor.ToFloat(in.Value)
	if err != nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_06_RPC, "浮点设点值解析失败: %s", in.Value)
	}
	sigDigits := tool.CountSignificantDigits(in.Value)
	if sigDigits > 7 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_06_RPC, "浮点设点值有效数字(%d位)超出IEEE754单精度上限(7位), 输入: %s", sigDigits, in.Value)
	}

	cli, err := l.svcCtx.ClientManager.GetClientOrNil(in.Host, int(in.Port))
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_RPC, err, "获取IEC客户端失败")
	}
	if cli == nil && l.svcCtx.IsBroadcast() {
		var res ieccaller.SendSetpointFloatRes
		err = l.svcCtx.PushPbBroadcastWithAck(l.ctx, ieccaller.IecCaller_SendSetpointFloat_FullMethodName, in, &res)
		if err != nil {
			return nil, wrapCommandAckError(err, "集群推送ACK失败")
		}
		return &res, nil
	} else if cli != nil {
		ack, err := cli.SendSetpointFloatCmd(l.ctx, uint16(in.Coa), asdu.InfoObjAddr(in.Ioa), float32(fv), in.WithTime, client.WithAck())
		if err != nil {
			return nil, wrapCommandAckError(err, "IEC发送浮点设点命令失败")
		}
		ackValue, err := ackFloat32Value(ack.Value)
		if err != nil {
			return nil, wrapCommandAckError(err, "IEC浮点设点ACK解析失败")
		}
		return &ieccaller.SendSetpointFloatRes{Value: convertor.ToString(ackValue)}, nil
	}
	return nil, tool.NewErrorByPbCode(extproto.Code__1_06_RPC, "IEC客户端不存在: %s:%d", in.Host, in.Port)
}
