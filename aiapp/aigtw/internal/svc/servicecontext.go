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

func (s *ServiceContext) Dependencies() map[string]string {
	deps := map[string]string{
		"jwt":        "ok",
		"aichat_rpc": "ok",
		"aisolo_rpc": "ok",
	}
	if strings.TrimSpace(s.Config.JwtAuth.AccessSecret) == "" {
		deps["jwt"] = "missing"
	}
	if s.AiChatCli == nil {
		deps["aichat_rpc"] = "missing"
	}
	if s.AiSoloCli == nil {
		deps["aisolo_rpc"] = "missing"
	}
	if s.Config.Knowledge.Enabled {
		deps["knowledge_backend"] = s.Config.Knowledge.EffectiveBackend()
	}
	if s.Knowledge != nil {
		deps["knowledge"] = "ok"
	} else if s.Config.Knowledge.Enabled {
		deps["knowledge"] = "misconfigured"
		if s.KnowledgeInitErr != "" {
			deps["knowledge_error"] = s.KnowledgeInitErr
		}
	} else {
		deps["knowledge"] = "disabled"
	}
	return deps
}

func (s *ServiceContext) Ready() bool {
	deps := s.Dependencies()
	return deps["jwt"] == "ok" && deps["aichat_rpc"] == "ok" && deps["aisolo_rpc"] == "ok" && deps["knowledge"] != "misconfigured"
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

// Close 释放 ServiceContext 持有的资源。
func (s *ServiceContext) Close() error {
	if s.Knowledge != nil {
		_ = s.Knowledge.Close()
	}
	return nil
}

func truncateInitErr(s string) string {
	const max = 512
	if len(s) <= max {
		return s
	}
	return strings.TrimSpace(s[:max])
}
