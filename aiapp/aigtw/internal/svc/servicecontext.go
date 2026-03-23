package svc

import (
	"zero-service/aiapp/aichat/aichatclient"
	"zero-service/aiapp/aigtw/internal/config"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/zerorpc/zerorpc"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config     config.Config
	ZeroRpcCli zerorpc.ZerorpcClient
	AiChatCli  aichatclient.AiChat
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		ZeroRpcCli: zerorpc.NewZerorpcClient(zrpc.MustNewClient(c.ZeroRpcConf,
			zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor)).Conn()),
		AiChatCli: aichatclient.NewAiChat(zrpc.MustNewClient(c.AiChatRpcConf,
			zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
			zrpc.WithStreamClientInterceptor(interceptor.StreamTracingInterceptor))),
	}
}
