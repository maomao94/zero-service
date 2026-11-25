package svc

import (
	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/gis/gis"
	"zero-service/app/openapi/internal/config"
	"zero-service/app/openapi/internal/dispatch"
	"zero-service/app/podengine/podengine"
	"zero-service/common/grpcx"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config   config.Config
	Registry *dispatch.Registry
}

func NewServiceContext(c config.Config) *ServiceContext {
	client := podengine.NewPodEngineClient(zrpc.MustNewClient(c.PodEngineConf,
		zrpc.WithUnaryClientInterceptor(grpcx.UnaryMetadataInterceptor)).Conn())
	modbusClient := bridgemodbus.NewBridgeModbusClient(zrpc.MustNewClient(c.BridgeModbusConf,
		zrpc.WithUnaryClientInterceptor(grpcx.UnaryMetadataInterceptor)).Conn())
	var gisClient gis.GisClient
	if c.GisConf != nil {
		gisClient = gis.NewGisClient(zrpc.MustNewClient(*c.GisConf,
			zrpc.WithUnaryClientInterceptor(grpcx.UnaryMetadataInterceptor)).Conn())
	}
	return &ServiceContext{
		Config:   c,
		Registry: dispatch.NewRegistry(client, modbusClient, gisClient),
	}
}
