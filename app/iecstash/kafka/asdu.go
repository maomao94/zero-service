package kafka

import (
	"context"
	"zero-service/app/iecstash/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type Asdu struct {
	svcCtx *svc.ServiceContext
	logger logx.Logger
}

func NewAsdu(svcCtx *svc.ServiceContext) *Asdu {
	return &Asdu{
		svcCtx: svcCtx,
		logger: logx.WithContext(context.Background()).WithFields(
			logx.Field("component", "iecstash.kafka"),
		),
	}
}

func (l Asdu) Consume(ctx context.Context, key, value string) error {
	logger := l.logger.WithContext(ctx)
	logger.Debugf("consume asdu, key: %s, bytes: %d, msg: %s", key, len(value), value)
	if err := l.svcCtx.ChunkAsduPusher.Write(value); err != nil {
		logger.WithFields(logx.Field("key", key)).Errorf("failed to write asdu to batch pusher: %v", err)
	}
	return nil
}
