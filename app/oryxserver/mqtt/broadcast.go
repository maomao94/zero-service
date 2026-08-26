package mqtt

import (
	"context"

	"zero-service/app/oryxserver/internal/relay"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/mqttx/broadcast"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/encoding/protojson"
)

// Broadcast 中继拉流集群广播执行器注册（对齐 ieccaller 模式）：
// payload 为 gRPC 请求的 protojson（StopRelayPullReq），executor 本地执行业务动作，
// 返回值为 gRPC 响应的 protojson（StopRelayPullRes）。
// 消费分发骨架（反序列化、防回环、ack 回发、errorKind 归一）由 common/mqttx/broadcast 提供。
// 注意：executor 内不调用带「未命中再广播」分支的 Logic，避免集群消息风暴；
// 仅注入最小依赖（RelayRegistry），不注入 ServiceContext。
type Broadcast struct {
	registry *relay.RelayRegistry
}

func NewBroadcast(registry *relay.RelayRegistry) *Broadcast {
	return &Broadcast{
		registry: registry,
	}
}

// RegisterExecutors 注册集群广播执行器到 Broadcaster（仅 cluster 模式由 NewServiceContext 调用）。
func (b *Broadcast) RegisterExecutors(bc broadcast.Broadcaster) {
	if bc == nil {
		return
	}
	bc.AddExecutor(oryxserver.OryxServer_StopRelayPull_FullMethodName, b.stopRelayPull)
}

// stopRelayPull 停止中继拉流任务执行器。
// 语义对齐 ieccaller：仅持有任务的节点回 ack（成功/失败都回），非本节点任务
// 返回 ErrSkipAck 不回 ack，避免虚假成功。
func (b *Broadcast) stopRelayPull(ctx context.Context, method string, payload []byte) ([]byte, error) {
	in := &oryxserver.StopRelayPullReq{}
	if err := protojson.Unmarshal(payload, in); err != nil {
		logx.WithContext(ctx).Errorw("relay mqtt broadcast stop decode failed",
			logx.Field("method", method),
			logx.Field("error", err),
		)
		return nil, err
	}
	found := b.registry.StopRelayByAppStream(in.App, in.Stream)
	if !found {
		logx.WithContext(ctx).Debugw("relay mqtt broadcast task not found on this node",
			logx.Field("app", in.App),
			logx.Field("stream", in.Stream),
		)
		return nil, broadcast.ErrSkipAck
	}
	logx.WithContext(ctx).Infof("relay mqtt broadcast stop success: app=%s stream=%s", in.App, in.Stream)
	resJson, err := protojson.Marshal(&oryxserver.StopRelayPullRes{})
	if err != nil {
		return nil, err
	}
	return resJson, nil
}
