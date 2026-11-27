package svc

import (
	"zero-service/app/bridgekafka/internal/config"
	"zero-service/common/grpcx"
	"zero-service/common/kqx"
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
		// 按 key 哈希分区：带 key 时同 key 同分区、组内有序；不带 key 时
		// go-queue 会注入纳秒时间戳作为 key，哈希后均匀分散到各分区。
		pushers[topic] = kqx.NewPusher(c.KafkaPushConfig.Brokers, topic,
			kqx.WithOrdered(),
			kqx.WithSyncPush(),
		)
	}

	var streamEventCli streamevent.StreamEventClient
	if len(c.StreamEventConf.Endpoints) > 0 || len(c.StreamEventConf.Target) > 0 {
		streamEventCli = streamevent.NewStreamEventClient(zrpc.MustNewClient(c.StreamEventConf,
			zrpc.WithUnaryClientInterceptor(grpcx.UnaryMetadataInterceptor),
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
