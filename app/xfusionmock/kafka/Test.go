package kafka

import (
	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/net/context"
	"zero-service/app/xfusionmock/internal/svc"
)

type Test struct {
	svcCtx *svc.ServiceContext
}

func NewTest(svcCtx *svc.ServiceContext) *Test {
	return &Test{
		svcCtx: svcCtx,
	}
}

func (l Test) Consume(ctx context.Context, key, value string) error {
	logx.Infof("consumerOne Consumer, key: %+v msg:%+v", key, value)
	return nil
}
