package svc

import (
	"context"

	"zero-service/aiapp/aichat/internal/config"
	"zero-service/aiapp/aichat/internal/mcpclient"
	"zero-service/aiapp/aichat/internal/provider"

	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
	Config    config.Config
	Registry  *provider.Registry
	McpClient *mcpclient.McpClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))
	registry, err := provider.NewRegistry(c.Providers, c.Models)
	if err != nil {
		panic(err)
	}

	var mc *mcpclient.McpClient
	if len(c.McpServers) > 0 {
		mc, err = mcpclient.NewMcpClient(context.Background(), c.McpServers[0].Endpoint)
		if err != nil {
			logx.Errorf("connect mcp server failed: %v, tools disabled", err)
		}
	}

	return &ServiceContext{
		Config:    c,
		Registry:  registry,
		McpClient: mc,
	}
}
