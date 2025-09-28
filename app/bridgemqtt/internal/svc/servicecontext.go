package svc

import (
	"zero-service/app/bridgemqtt/internal/config"
	"zero-service/app/bridgemqtt/internal/handler"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/common/mqttx"
	"zero-service/facade/mqttstream/mqttstream"

	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type ServiceContext struct {
	Config        config.Config
	MqttClient    *mqttx.Client
	MqttStreamCli mqttstream.MqttStreamClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	mqttCLi := mqttx.MustNewClient(c.MqttConfig)
	mqttStreamCli := mqttstream.NewMqttStreamClient(zrpc.MustNewClient(c.MqttStreamConf,
		zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
		// 添加最大消息配置
		zrpc.WithDialOption(grpc.WithDefaultCallOptions(
			//grpc.MaxCallSendMsgSize(math.MaxInt32), // 发送最大2GB
			grpc.MaxCallSendMsgSize(50*1024*1024), // 发送最大50MB
			//grpc.MaxCallRecvMsgSize(100 * 1024 * 1024),  // 接收最大100MB
		)),
	).Conn())
	// 注册转发 handler
	for _, topic := range c.MqttConfig.SubscribeTopics {
		mqttCLi.AddHandler(topic, handler.NewMqttStreamHandler(mqttCLi.GetClientID(), mqttStreamCli))
	}
	return &ServiceContext{
		Config:        c,
		MqttClient:    mqttCLi,
		MqttStreamCli: mqttStreamCli,
	}
}
