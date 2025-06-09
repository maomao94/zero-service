package kafka

import (
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/net/context"
	"zero-service/app/iecstash/internal/svc"
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
	logx.Infof("asdu, key: %+v msg:%+v", key, value)
	l.svcCtx.AsduPusher.Write(value)
	return nil
}
