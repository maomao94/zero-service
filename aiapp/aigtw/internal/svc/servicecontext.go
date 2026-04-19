package svc

import (
	"strings"

	"zero-service/aiapp/aichat/aichat"
	"zero-service/aiapp/aigtw/internal/config"
	"zero-service/aiapp/aisolo/aisolo"
	interceptor "zero-service/common/Interceptor/rpcclient"
	einoxkb "zero-service/common/einox/knowledge"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

// ServiceContext aigtw HTTP 网关的服务上下文。
type ServiceContext struct {
	Config    config.Config
	AiChatCli aichat.AiChatClient
	AiSoloCli aisolo.AiSoloClient
	Knowledge *einoxkb.Service
	// KnowledgeInitErr 非空表示 Knowledge 启用但初始化失败（如连接/校验错误），供 /health 与 /solo/v1/meta 摘要展示。
	KnowledgeInitErr string
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
	if kb, err := einoxkb.NewService(c.Knowledge, ""); err != nil {
		logx.Errorf("[svc] knowledge: %v", err)
		s.KnowledgeInitErr = truncateInitErr(err.Error())
	} else {
		s.Knowledge = kb
		if kb != nil {
			logx.Infof("[svc] knowledge ready backend=%s", c.Knowledge.EffectiveBackend())
		}
	}
	return s
}

func truncateInitErr(s string) string {
	const max = 512
	if len(s) <= max {
		return s
	}
	return strings.TrimSpace(s[:max])
}
