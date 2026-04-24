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

type ServiceContext struct {
	Config      config.Config
	DjiClient   *djisdk.Client
	OnlineCache *collection.Cache
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))
	mqttCli := mqttx.MustNewClient(c.MqttConfig)
	djiCli := djisdk.NewClient(mqttCli, c.PendingTTL)

	onlineCache, err := collection.NewCache(dockOnlineTTL, collection.WithName("dock-online"))
	logx.Must(err)

	if err := djiCli.SubscribeAll(); err != nil {
		logx.Errorf("[dji-gateway] subscribe topics failed: %v", err)
	}

	djiCli.OnFlightTaskProgress(hooks.OnFlightTaskProgress)
	djiCli.OnFlightTaskReady(hooks.OnFlightTaskReady)
	djiCli.OnReturnHomeInfo(hooks.OnReturnHomeInfo)
	djiCli.OnCustomDataFromPsdk(hooks.OnCustomDataFromPsdk)
	djiCli.OnHmsEventNotify(hooks.OnHmsEventNotify)
	djiCli.OnOsd(hooks.OnOsd(onlineCache))
	djiCli.OnState(hooks.OnState)
	djiCli.OnStatus(hooks.OnStatus(onlineCache))
	djiCli.SetOnlineChecker(func(gatewaySn string) bool {
		return hooks.IsOnline(onlineCache, gatewaySn)
	})

	return &ServiceContext{
		Config:      c,
		DjiClient:   djiCli,
		OnlineCache: onlineCache,
	}
}
