package svc

import (
	"time"

	"zero-service/app/djigateway/internal/config"
	"zero-service/app/djigateway/internal/hooks"
	"zero-service/common/djisdk"
	"zero-service/common/mqttx"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
)

const dockOnlineTTL = 60 * time.Second

// flightProgressTTL 航线进度缓存条目的 TTL；每次 flighttask_progress 上报会 Set 同 key，任务持续进行时会不断刷新，条目留在内存中。
const flightProgressTTL = 24 * time.Hour

type ServiceContext struct {
	Config              config.Config
	DjiClient           *djisdk.Client
	OnlineCache         *collection.Cache
	FlightProgressCache *collection.Cache
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))
	mqttCli := mqttx.MustNewClient(c.MqttConfig)
	djiCli := djisdk.NewClient(mqttCli,
		djisdk.WithPendingTTL(c.PendingTTL),
		djisdk.WithReplyOptions(djisdk.ReplyOptions{
			EnableEventReply:   c.UpstreamReply.EnableEventsReply,
			EnableStatusReply:  c.UpstreamReply.EnableStatusReply,
			EnableRequestReply: c.UpstreamReply.EnableRequestsReply,
		}),
	)

	onlineCache, err := collection.NewCache(dockOnlineTTL, collection.WithName("dock-online"))
	logx.Must(err)

	flightProgressCache, err := collection.NewCache(flightProgressTTL, collection.WithName("flight-task-progress"))
	logx.Must(err)

	if err := djiCli.SubscribeAll(); err != nil {
		logx.Errorf("[dji-gateway] subscribe topics failed: %v", err)
	}

	hooks.RegisterDjiClient(djiCli, hooks.RegisterDjiClientOptions{
		OnlineCache:         onlineCache,
		FlightProgressCache: flightProgressCache,
	})

	return &ServiceContext{
		Config:              c,
		DjiClient:           djiCli,
		OnlineCache:         onlineCache,
		FlightProgressCache: flightProgressCache,
	}
}
