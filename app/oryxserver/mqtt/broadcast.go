package mqtt

import (
	"context"

	"zero-service/app/oryxserver/internal/relay"
	"zero-service/common/mqttx/broadcast"

	"github.com/zeromicro/go-zero/core/logx"
)

// Broadcast 转推集群广播执行器注册：消费分发骨架（反序列化、防回环、ack 回发、errorKind 归一）
// 由 common/mqttx/broadcast 提供，此处仅注册 relay 业务执行器。
// 依赖只注入最小集（RelayManager），不注入 ServiceContext。
type Broadcast struct {
	relayMgr *relay.Manager
}

func NewBroadcast(relayMgr *relay.Manager) *Broadcast {
	return &Broadcast{
		relayMgr: relayMgr,
	}
}

// RegisterExecutors 注册集群广播执行器到 Broadcaster（仅 cluster 模式由 NewServiceContext 调用）。
func (b *Broadcast) RegisterExecutors(bc broadcast.Broadcaster) {
	if bc == nil {
		return
	}
	bc.AddExecutor(relay.MethodStreamRelayStop, b.stopRelay)
}

// stopRelay 停止转推任务执行器。
// payload 为 taskId 原文（与发送端 StreamRelayStopLogic 保持一致，发送 []byte(taskId)）。
// 语义对齐迁移前：仅持有任务的节点回 ack；非本节点任务（found==false）
// 返回 ErrSkipAck（不回 ack，避免虚假成功）。
func (b *Broadcast) stopRelay(ctx context.Context, method string, payload []byte) ([]byte, error) {
	taskID := string(payload)
	found, err := b.relayMgr.Stop(taskID)
	if err != nil {
		logx.WithContext(ctx).Errorw("relay mqtt broadcast stop failed",
			logx.Field("task_id", taskID),
			logx.Field("error", err),
		)
		return nil, err
	}
	if !found {
		logx.WithContext(ctx).Debugw("relay mqtt broadcast task not found on this node",
			logx.Field("task_id", taskID),
		)
		return nil, broadcast.ErrSkipAck
	}
	logx.WithContext(ctx).Infof("relay mqtt broadcast stop success: task_id=%s", taskID)
	return nil, nil
}
