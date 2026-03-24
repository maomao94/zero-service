package svc

import (
	"zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aigtw/internal/config"
	interceptor "zero-service/common/Interceptor/rpcclient"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config    config.Config
	AiChatCli aichat.AiChatClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))
	return &ServiceContext{
		Config: c,
		AiChatCli: aichat.NewAiChatClient(zrpc.MustNewClient(c.AiChatRpcConf,
			zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
			zrpc.WithStreamClientInterceptor(interceptor.StreamTracingInterceptor)).Conn()),
	}
}
