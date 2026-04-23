package svc

import (
	"zero-service/app/djigateway/internal/config"
	"zero-service/app/djigateway/internal/hooks"
	"zero-service/common/djisdk"
	"zero-service/common/mqttx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
	Config    config.Config
	DjiClient *djisdk.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	mqttCli := mqttx.MustNewClient(c.MqttConfig)
	djiCli := djisdk.NewClient(mqttCli, c.AckTimeout, c.PendingTTL)

	if err := djiCli.SubscribeAll(); err != nil {
		logx.Errorf("[dji-gateway] subscribe topics failed: %v", err)
	}

	djiCli.OnFlightTaskProgress(hooks.OnFlightTaskProgress)
	djiCli.OnFlightTaskReady(hooks.OnFlightTaskReady)
	djiCli.OnReturnHomeInfo(hooks.OnReturnHomeInfo)
	djiCli.OnCustomDataFromPsdk(hooks.OnCustomDataFromPsdk)

	return &ServiceContext{
		Config:    c,
		DjiClient: djiCli,
	}
}
