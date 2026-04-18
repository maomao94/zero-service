package svc

import (
	"zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aigtw/internal/config"
	"zero-service/aiapp/aisolo/aisolo"
	interceptor "zero-service/common/Interceptor/rpcclient"
	einoxrag "zero-service/common/einox/rag"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

// ServiceContext aigtw HTTP 网关的服务上下文。
type ServiceContext struct {
	Config    config.Config
	AiChatCli aichat.AiChatClient
	AiSoloCli aisolo.AiSoloClient
	Rag       *einoxrag.Service
}

// NewServiceContext 构造 ServiceContext。
func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))

	s := &ServiceContext{
		Config: c,
		AiChatCli: aichat.NewAiChatClient(zrpc.MustNewClient(c.AiChatRpcConf,
			zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
			zrpc.WithStreamClientInterceptor(interceptor.StreamTracingInterceptor)).Conn()),
		AiSoloCli: aisolo.NewAiSoloClient(zrpc.MustNewClient(c.AiSoloRpcConf,
			zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
			zrpc.WithStreamClientInterceptor(interceptor.StreamTracingInterceptor)).Conn()),
	}
	if ragSvc, err := einoxrag.NewService(c.Rag, ""); err != nil {
		logx.Errorf("[svc] rag: %v", err)
	} else {
		s.Rag = ragSvc
		if ragSvc != nil {
			logx.Info("[svc] rag service ready")
		}
	}
	return s
}
