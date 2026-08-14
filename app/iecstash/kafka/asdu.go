package kafka

import (
	"context"
	"zero-service/app/iecstash/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type Asdu struct {
	svcCtx *svc.ServiceContext
}

func NewAsdu(svcCtx *svc.ServiceContext) *Asdu {
	return &Asdu{
		svcCtx: svcCtx,
	}
}

func (l Asdu) Consume(ctx context.Context, key, value string) error {
	logx.Debugf("asdu, key: %+v, msg:%+v", key, value)
	l.svcCtx.ChunkAsduPusher.Write(value)
	return nil
}
