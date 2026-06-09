package svc

import (
	"zero-service/app/bridgekafka/internal/config"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/facade/streamevent/streamevent"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type ServiceContext struct {
	Config         config.Config
	Pushers        map[string]*kq.Pusher
	StreamEventCli streamevent.StreamEventClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))

	pushers := make(map[string]*kq.Pusher)
	for _, topic := range c.KafkaPushConfig.Topics {
		pushers[topic] = kq.NewPusher(c.KafkaPushConfig.Brokers, topic, kq.WithSyncPush())
	}

	var streamEventCli streamevent.StreamEventClient
	if len(c.StreamEventConf.Endpoints) > 0 || len(c.StreamEventConf.Target) > 0 {
		streamEventCli = streamevent.NewStreamEventClient(zrpc.MustNewClient(c.StreamEventConf,
			zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
			zrpc.WithDialOption(grpc.WithDefaultCallOptions(
				grpc.MaxCallSendMsgSize(50*1024*1024),
			)),
		).Conn())
	}

	return &ServiceContext{
		Config:         c,
		Pushers:        pushers,
		StreamEventCli: streamEventCli,
	}
}
