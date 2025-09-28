package svc

import (
	"zero-service/app/bridgemqtt/internal/config"
	"zero-service/common/mqttx"
)

type ServiceContext struct {
	Config     config.Config
	MqttClient *mqttx.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:     c,
		MqttClient: mqttx.MustNewClient(c.MqttConfig),
	}
}
