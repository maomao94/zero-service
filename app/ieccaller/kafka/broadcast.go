package kafka

import (
	"zero-service/app/ieccaller/ieccaller"
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/common/iec104/types"

	"github.com/wendy512/go-iecp5/asdu"
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/net/context"
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
	logx.WithContext(ctx).Infof("broadcast, msg:%+v", value)
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
		logx.WithContext(ctx).Debug("ignore broadcast")
		return nil
	}
	switch broadcastBody.Method {
	case ieccaller.IecCaller_SendCounterInterrogationCmd_FullMethodName:
		in := &ieccaller.SendCounterInterrogationCmdReq{}
		err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
		if err != nil {
			return err
		}
		cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
		if err != nil {
			logx.WithContext(ctx).Errorf("get client error: %v", err)
			return nil
		}
		if err = cli.SendCounterInterrogationCmd(uint16(in.Coa)); err != nil {
			return err
		}
	case ieccaller.IecCaller_SendInterrogationCmd_FullMethodName:
		in := &ieccaller.SendInterrogationCmdReq{}
		err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
		if err != nil {
			return err
		}
		cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
		if err != nil {
			logx.WithContext(ctx).Errorf("get client error: %v", err)
			return nil
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
			logx.WithContext(ctx).Errorf("get client error: %v", err)
			return nil
		}
		if err = cli.SendReadCmd(uint16(in.Coa), uint(in.Ioa)); err != nil {
			return err
		}
	case ieccaller.IecCaller_SendTestCmd_FullMethodName:
		in := &ieccaller.SendTestCmdReq{}
		err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
		if err != nil {
			return err
		}
		cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
		if err != nil {
			logx.WithContext(ctx).Errorf("get client error: %v", err)
			return nil
		}
		if err = cli.SendTestCmd(uint16(in.Coa)); err != nil {
			return err
		}
	case ieccaller.IecCaller_SendCommand_FullMethodName:
		in := &ieccaller.SendCommandReq{}
		err = jsonx.Unmarshal([]byte(broadcastBody.Body), in)
		if err != nil {
			return err
		}
		cli, err := l.svcCtx.ClientManager.GetClient(in.Host, int(in.Port))
		if err != nil {
			logx.WithContext(ctx).Errorf("get client error: %v", err)
			return nil
		}
		if err = cli.SendCmd(uint16(in.Coa), asdu.TypeID(in.TypeId), asdu.InfoObjAddr(in.Ioa), in.Value); err != nil {
			return err
		}
	default:
		logx.WithContext(ctx).Errorf("unknown method:%s", broadcastBody.Method)
	}
	return nil
}
