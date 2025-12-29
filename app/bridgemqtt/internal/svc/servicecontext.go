package svc

import (
	"zero-service/app/bridgemqtt/internal/config"
	"zero-service/app/bridgemqtt/internal/handler"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/common/mqttx"
	"zero-service/common/socketio"
	"zero-service/facade/streamevent/streamevent"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type ServiceContext struct {
	Config          config.Config
	MqttClient      *mqttx.Client
	StreamEventCli  streamevent.StreamEventClient
	SocketContainer *socketio.SocketContainer
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))
	streamEventCli := streamevent.NewStreamEventClient(zrpc.MustNewClient(c.StreamEventConf,
		zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
		// 添加最大消息配置
		zrpc.WithDialOption(grpc.WithDefaultCallOptions(
			//grpc.MaxCallSendMsgSize(math.MaxInt32), // 发送最大2GB
			grpc.MaxCallSendMsgSize(50*1024*1024), // 发送最大50MB
			//grpc.MaxCallRecvMsgSize(100 * 1024 * 1024),  // 接收最大100MB
		)),
	).Conn())
	socketContainer := socketio.NewPubContainer(c.SocketMsgGtwConf)
	mqttCLi := mqttx.MustNewClient(c.MqttConfig,
		mqttx.WithOnReady(func(cli *mqttx.Client) {
			logx.Infof("[mqtt] OnReady, client=%s", cli.GetClientID())
			// 注册转发 handler
			for _, topic := range c.MqttConfig.SubscribeTopics {
				cli.AddHandler(topic, handler.NewMqttStreamHandler(cli.GetClientID(), streamEventCli, socketContainer))
			}
		}),
	)
	return &ServiceContext{
		Config:          c,
		MqttClient:      mqttCLi,
		StreamEventCli:  streamEventCli,
		SocketContainer: socketContainer,
	}
}
