package kafka

import (
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/net/context"
	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/iec104/types"
)

type Broadcast struct {
	svcCtx *svc.ServiceContext
}

func NewBroadcast(svcCtx *svc.ServiceContext) *Broadcast {
	return &Broadcast{
		svcCtx: svcCtx,
	}
}

func (l Broadcast) Consume(ctx context.Context, key, value string) error {
	logx.Infof("broadcast, msg:%+v", value)
	if !l.svcCtx.IsBroadcast() {
		logx.Error("not setting cluster")
		return nil
	}
	broadcastBody := &types.BroadcastBody{}
	err := jsonx.Unmarshal([]byte(value), broadcastBody)
	if err != nil {
		return err
	}
	if broadcastBody.BroadcastGroupId == l.svcCtx.Config.KafkaConfig.BroadcastGroupId {
		logx.Debug("ignore broadcast")
	}
	switch broadcastBody.Method {
	case ieccaller.IecCaller_SendInterrogationCmd_FullMethodName:
		in := &ieccaller.SendInterrogationCmdReq{}
		err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
		if err != nil {
			return err
		}
		cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
		if err != nil {
			return err
		}
		if err = cli.SendInterrogationCmd(uint16(in.Coa)); err != nil {
			return err
		}
	case ieccaller.IecCaller_SendReadCmd_FullMethodName:
		in := &ieccaller.SendReadCmdReq{}
		err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
		if err != nil {
			return err
		}
		cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
		if err != nil {
			return err
		}
		if err = cli.SendReadCmd(uint16(in.Coa), uint(in.Ioa)); err != nil {
			return err
		}
	default:
		logx.Errorf("unknown method:%s", broadcastBody.Method)
	}
	return nil
}
