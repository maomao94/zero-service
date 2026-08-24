package mqtt

import (
	"context"
	"fmt"

	"zero-service/app/ieccaller/ieccaller"
	"zero-service/common/iec104/client"
	"zero-service/common/mqttx/broadcast"
	"zero-service/model/gormmodel"

	"github.com/duke-git/lancet/v2/convertor"
	"github.com/wendy512/go-iecp5/asdu"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/encoding/protojson"
)

// Broadcast 集群广播业务执行器注册：method → executor 逐条注册到 broadcast.Broadcaster，
// 消费分发骨架（反序列化、防回环、ack 回发、errorKind 归一）由 common/mqttx/broadcast 提供。
// 此处仅注册「业务反序列化 + IEC104 命令执行」，protojson 反序列化保留在 executor 内。
// 依赖只注入最小集（ClientManager / 点位映射存储），不注入 ServiceContext。
type Broadcast struct {
	clientMgr *client.ClientManager
	store     *gormmodel.DevicePointMappingStore
}

func NewBroadcast(clientMgr *client.ClientManager, store *gormmodel.DevicePointMappingStore) *Broadcast {
	return &Broadcast{
		clientMgr: clientMgr,
		store:     store,
	}
}

// RegisterExecutors 注册全部集群广播业务执行器（仅 cluster 模式由 NewServiceContext 调用）。
func (b *Broadcast) RegisterExecutors(bc broadcast.Broadcaster) {
	if bc == nil {
		return
	}
	b.register(bc, ieccaller.IecCaller_SendCounterInterrogationCmd_FullMethodName, b.sendCounterInterrogationCmd)
	b.register(bc, ieccaller.IecCaller_SendInterrogationCmd_FullMethodName, b.sendInterrogationCmd)
	b.register(bc, ieccaller.IecCaller_SendReadCmd_FullMethodName, b.sendReadCmd)
	b.register(bc, ieccaller.IecCaller_SendTestCmd_FullMethodName, b.sendTestCmd)
	b.register(bc, ieccaller.IecCaller_SendCommand_FullMethodName, b.sendCommand)
	b.register(bc, ieccaller.IecCaller_SendSingleCommand_FullMethodName, b.sendSingleCommand)
	b.register(bc, ieccaller.IecCaller_SendDoubleCommand_FullMethodName, b.sendDoubleCommand)
	b.register(bc, ieccaller.IecCaller_SendStepCommand_FullMethodName, b.sendStepCommand)
	b.register(bc, ieccaller.IecCaller_SendSetpointNormalized_FullMethodName, b.sendSetpointNormalized)
	b.register(bc, ieccaller.IecCaller_SendSetpointScaled_FullMethodName, b.sendSetpointScaled)
	b.register(bc, ieccaller.IecCaller_SendSetpointFloat_FullMethodName, b.sendSetpointFloat)
	b.register(bc, ieccaller.IecCaller_SendBitstringCommand_FullMethodName, b.sendBitstringCommand)
	b.register(bc, ieccaller.IecCaller_ClearPointMappingCache_FullMethodName, b.clearPointMappingCache)
}

func (b *Broadcast) register(bc broadcast.Broadcaster, method string, fn broadcast.Executor) {
	bc.AddExecutor(method, fn)
}

// clientFor 获取目标 IEC104 节点客户端。
// 客户端不存在（设备不在本节点）时返回 broadcast.ErrSkipAck：迁移前相同语义为静默跳过
// 不回 ack（仅持有该设备的节点回包，避免非 owner 节点造成虚假失败）。
func (b *Broadcast) clientFor(ctx context.Context, method string, host string, port uint32) (*client.Client, error) {
	cli, err := b.clientMgr.GetClient(host, int(port))
	if err != nil {
		logBroadcastClientError(ctx, method, host, port, err)
		return nil, broadcast.ErrSkipAck
	}
	return cli, nil
}

func (b *Broadcast) sendCounterInterrogationCmd(ctx context.Context, method string, payload []byte) ([]byte, error) {
	in := &ieccaller.SendCounterInterrogationCmdReq{}
	if err := protojson.Unmarshal(payload, in); err != nil {
		return nil, err
	}
	cli, err := b.clientFor(ctx, method, in.Host, in.Port)
	if err != nil {
		return nil, err
	}
	if err = cli.SendCounterInterrogationCmd(uint16(in.Coa)); err != nil {
		return nil, err
	}
	return []byte("{}"), nil
}

func (b *Broadcast) sendInterrogationCmd(ctx context.Context, method string, payload []byte) ([]byte, error) {
	in := &ieccaller.SendInterrogationCmdReq{}
	if err := protojson.Unmarshal(payload, in); err != nil {
		return nil, err
	}
	cli, err := b.clientFor(ctx, method, in.Host, in.Port)
	if err != nil {
		return nil, err
	}
	if err = cli.SendInterrogationCmd(uint16(in.Coa)); err != nil {
		return nil, err
	}
	return []byte("{}"), nil
}

func (b *Broadcast) sendReadCmd(ctx context.Context, method string, payload []byte) ([]byte, error) {
	in := &ieccaller.SendReadCmdReq{}
	if err := protojson.Unmarshal(payload, in); err != nil {
		return nil, err
	}
	cli, err := b.clientFor(ctx, method, in.Host, in.Port)
	if err != nil {
		return nil, err
	}
	if err = cli.SendReadCmd(uint16(in.Coa), uint(in.Ioa)); err != nil {
		return nil, err
	}
	return []byte("{}"), nil
}

func (b *Broadcast) sendTestCmd(ctx context.Context, method string, payload []byte) ([]byte, error) {
	in := &ieccaller.SendTestCmdReq{}
	if err := protojson.Unmarshal(payload, in); err != nil {
		return nil, err
	}
	cli, err := b.clientFor(ctx, method, in.Host, in.Port)
	if err != nil {
		return nil, err
	}
	if err = cli.SendTestCmd(uint16(in.Coa)); err != nil {
		return nil, err
	}
	return []byte("{}"), nil
}

func (b *Broadcast) sendCommand(ctx context.Context, method string, payload []byte) ([]byte, error) {
	in := &ieccaller.SendCommandReq{}
	if err := protojson.Unmarshal(payload, in); err != nil {
		return nil, err
	}
	cli, err := b.clientFor(ctx, method, in.Host, in.Port)
	if err != nil {
		return nil, err
	}
	if err = cli.SendCmd(uint16(in.Coa), asdu.TypeID(in.TypeId), asdu.InfoObjAddr(in.Ioa), in.Value); err != nil {
		return nil, err
	}
	return []byte("{}"), nil
}

func (b *Broadcast) sendSingleCommand(ctx context.Context, method string, payload []byte) ([]byte, error) {
	in := &ieccaller.SendSingleCommandReq{}
	if err := protojson.Unmarshal(payload, in); err != nil {
		return nil, err
	}
	cli, err := b.clientFor(ctx, method, in.Host, in.Port)
	if err != nil {
		return nil, err
	}
	ack, err := cli.SendSingleCmd(ctx, uint16(in.Coa), asdu.InfoObjAddr(in.Ioa), in.Value, in.WithTime, client.WithAck())
	if err != nil {
		return nil, err
	}
	value, ok := ack.Value.(bool)
	if !ok {
		return nil, fmt.Errorf("unexpected ack value type")
	}
	resJson, _ := protojson.Marshal(&ieccaller.SendSingleCommandRes{Value: value})
	return resJson, nil
}

func (b *Broadcast) sendDoubleCommand(ctx context.Context, method string, payload []byte) ([]byte, error) {
	in := &ieccaller.SendDoubleCommandReq{}
	if err := protojson.Unmarshal(payload, in); err != nil {
		return nil, err
	}
	cli, err := b.clientFor(ctx, method, in.Host, in.Port)
	if err != nil {
		return nil, err
	}
	ack, err := cli.SendDoubleCmd(ctx, uint16(in.Coa), asdu.InfoObjAddr(in.Ioa), asdu.DoubleCommand(in.Value), in.WithTime, client.WithAck())
	if err != nil {
		return nil, err
	}
	value, ok := ack.Value.(asdu.DoubleCommand)
	if !ok {
		return nil, fmt.Errorf("unexpected ack value type")
	}
	resJson, _ := protojson.Marshal(&ieccaller.SendDoubleCommandRes{Value: ieccaller.DoubleCommandValue(int32(value))})
	return resJson, nil
}

func (b *Broadcast) sendStepCommand(ctx context.Context, method string, payload []byte) ([]byte, error) {
	in := &ieccaller.SendStepCommandReq{}
	if err := protojson.Unmarshal(payload, in); err != nil {
		return nil, err
	}
	cli, err := b.clientFor(ctx, method, in.Host, in.Port)
	if err != nil {
		return nil, err
	}
	ack, err := cli.SendStepCmd(ctx, uint16(in.Coa), asdu.InfoObjAddr(in.Ioa), asdu.StepCommand(in.Value), in.WithTime, client.WithAck())
	if err != nil {
		return nil, err
	}
	value, ok := ack.Value.(asdu.StepCommand)
	if !ok {
		return nil, fmt.Errorf("unexpected ack value type")
	}
	resJson, _ := protojson.Marshal(&ieccaller.SendStepCommandRes{Value: int32(value)})
	return resJson, nil
}

func (b *Broadcast) sendSetpointNormalized(ctx context.Context, method string, payload []byte) ([]byte, error) {
	in := &ieccaller.SendSetpointNormalizedReq{}
	if err := protojson.Unmarshal(payload, in); err != nil {
		return nil, err
	}
	cli, err := b.clientFor(ctx, method, in.Host, in.Port)
	if err != nil {
		return nil, err
	}
	ack, err := cli.SendSetpointNormalizedCmd(ctx, uint16(in.Coa), asdu.InfoObjAddr(in.Ioa), int16(in.Value), in.WithTime, client.WithAck())
	if err != nil {
		return nil, err
	}
	value, ok := ack.Value.(asdu.Normalize)
	if !ok {
		return nil, fmt.Errorf("unexpected ack value type")
	}
	resJson, _ := protojson.Marshal(&ieccaller.SendSetpointNormalizedRes{Value: int32(value)})
	return resJson, nil
}

func (b *Broadcast) sendSetpointScaled(ctx context.Context, method string, payload []byte) ([]byte, error) {
	in := &ieccaller.SendSetpointScaledReq{}
	if err := protojson.Unmarshal(payload, in); err != nil {
		return nil, err
	}
	cli, err := b.clientFor(ctx, method, in.Host, in.Port)
	if err != nil {
		return nil, err
	}
	ack, err := cli.SendSetpointScaledCmd(ctx, uint16(in.Coa), asdu.InfoObjAddr(in.Ioa), int16(in.Value), in.WithTime, client.WithAck())
	if err != nil {
		return nil, err
	}
	value, ok := ack.Value.(int16)
	if !ok {
		return nil, fmt.Errorf("unexpected ack value type")
	}
	resJson, _ := protojson.Marshal(&ieccaller.SendSetpointScaledRes{Value: int32(value)})
	return resJson, nil
}

func (b *Broadcast) sendSetpointFloat(ctx context.Context, method string, payload []byte) ([]byte, error) {
	in := &ieccaller.SendSetpointFloatReq{}
	if err := protojson.Unmarshal(payload, in); err != nil {
		return nil, err
	}
	cli, err := b.clientFor(ctx, method, in.Host, in.Port)
	if err != nil {
		return nil, err
	}
	fv, err := convertor.ToFloat(in.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid float value: %s", in.Value)
	}
	ack, err := cli.SendSetpointFloatCmd(ctx, uint16(in.Coa), asdu.InfoObjAddr(in.Ioa), float32(fv), in.WithTime, client.WithAck())
	if err != nil {
		return nil, err
	}
	ackValue, ok := ack.Value.(float32)
	if !ok {
		return nil, fmt.Errorf("unexpected ack value type")
	}
	resJson, _ := protojson.Marshal(&ieccaller.SendSetpointFloatRes{Value: convertor.ToString(ackValue)})
	return resJson, nil
}

func (b *Broadcast) sendBitstringCommand(ctx context.Context, method string, payload []byte) ([]byte, error) {
	in := &ieccaller.SendBitstringCommandReq{}
	if err := protojson.Unmarshal(payload, in); err != nil {
		return nil, err
	}
	cli, err := b.clientFor(ctx, method, in.Host, in.Port)
	if err != nil {
		return nil, err
	}
	ack, err := cli.SendBitstringCmd(ctx, uint16(in.Coa), asdu.InfoObjAddr(in.Ioa), uint32(in.Value), in.WithTime, client.WithAck())
	if err != nil {
		return nil, err
	}
	value, ok := ack.Value.(uint32)
	if !ok {
		return nil, fmt.Errorf("unexpected ack value type")
	}
	resJson, _ := protojson.Marshal(&ieccaller.SendBitstringCommandRes{Value: uint64(value)})
	return resJson, nil
}

// clearPointMappingCache 清除点位映射缓存（fire-and-forget：不回 ack，迁移前语义不变）。
func (b *Broadcast) clearPointMappingCache(ctx context.Context, method string, payload []byte) ([]byte, error) {
	in := &ieccaller.ClearPointMappingCacheReq{}
	if err := protojson.Unmarshal(payload, in); err != nil {
		return nil, err
	}
	clearedCount := int64(0)
	if b.store != nil {
		if len(in.Keys) > 0 {
			for _, key := range in.Keys {
				if _, exists := b.store.GetCache(ctx, key); exists {
					if err := b.store.RemoveCache(ctx, key); err != nil {
						logx.WithContext(ctx).Errorw("mqtt broadcast cache remove failed",
							logx.Field("key", key),
							logx.Field("error", err),
						)
						continue
					}
					clearedCount++
				}
			}
		}
		if len(in.KeyInfos) > 0 {
			for _, info := range in.KeyInfos {
				key := b.store.GenerateCacheKey(info.TagStation, info.Coa, info.Ioa)
				if _, exists := b.store.GetCache(ctx, key); exists {
					if err := b.store.RemoveCache(ctx, key); err != nil {
						logx.WithContext(ctx).Errorw("mqtt broadcast cache remove failed",
							logx.Field("tag_station", info.TagStation),
							logx.Field("coa", info.Coa),
							logx.Field("ioa", info.Ioa),
							logx.Field("error", err),
						)
						continue
					}
					clearedCount++
				}
			}
		}
		logx.WithContext(ctx).Infow("mqtt broadcast cache cleared", logx.Field("cleared_count", clearedCount))
	}
	return nil, broadcast.ErrSkipAck
}

func logBroadcastClientError(ctx context.Context, method string, host string, port uint32, err error) {
	logx.WithContext(ctx).Debugw(fmt.Sprintf("mqtt broadcast client skipped: method=%s target=%s:%d", method, host, port),
		logx.Field("error", err),
	)
}
