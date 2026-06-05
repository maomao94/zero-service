package kafka

import (
	"zero-service/app/ieccaller/internal/svc"
	"zero-service/common/iec104/types"

	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/net/context"
)

type BroadcastAck struct {
	svcCtx *svc.ServiceContext
}

func NewBroadcastAck(svcCtx *svc.ServiceContext) *BroadcastAck {
	return &BroadcastAck{
		svcCtx: svcCtx,
	}
}

func (l *BroadcastAck) Consume(ctx context.Context, key, value string) error {
	ackBody := &types.BroadcastAckBody{}
	err := jsonx.Unmarshal([]byte(value), ackBody)
	if err != nil {
		logx.WithContext(ctx).Errorf("unmarshal broadcast ack error: %v", err)
		return nil
	}

	if l.svcCtx.BroadcastReplyPool == nil {
		return nil
	}

	// 只有匹配本实例 BroadcastGroupId 的 ACK reply 才需要 resolve
	if ackBody.BroadcastGroupId != l.svcCtx.BroadcastInstanceId() {
		return nil
	}

	l.svcCtx.BroadcastReplyPool.Resolve(key, ackBody)
	return nil
}
