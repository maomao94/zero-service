package mqtt

import (
	"context"

	"zero-service/app/ieccaller/internal/svc"
	"zero-service/common/iec104/types"

	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logx"
)

type BroadcastAck struct {
	svcCtx *svc.ServiceContext
}

func NewBroadcastAck(svcCtx *svc.ServiceContext) *BroadcastAck {
	return &BroadcastAck{
		svcCtx: svcCtx,
	}
}

func (l *BroadcastAck) Consume(ctx context.Context, payload []byte, topic string, topicTemplate string) error {
	ackBody := &types.BroadcastAckBody{}
	err := jsonx.Unmarshal(payload, ackBody)
	if err != nil {
		logx.WithContext(ctx).Errorf("unmarshal broadcast ack error: %v", err)
		return nil
	}

	if l.svcCtx.BroadcastReplyPool == nil {
		return nil
	}

	if ackBody.Tid == "" {
		logx.WithContext(ctx).Errorf("broadcast ack tId is empty")
		return nil
	}

	l.svcCtx.BroadcastReplyPool.Resolve(ackBody.Tid, ackBody)
	return nil
}
