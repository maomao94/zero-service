package svc

import (
	"zero-service/aiapp/aichat/internal/config"
	"zero-service/aiapp/aichat/internal/provider"
	"zero-service/common/mcpx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ServiceContext struct {
	Config           config.Config
	Registry         *provider.Registry
	McpClient        *mcpx.Client
	AsyncResultStore mcpx.AsyncResultStore // 异步结果存储，二开可注入 Redis/MySQL 等实现
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))
	registry, err := provider.NewRegistry(c.Providers, c.Models)
	if err != nil {
		panic(err)
	}

	var mc *mcpx.Client
	mcpCfg := c.Mcpx
	if len(mcpCfg.Servers) > 0 {
		mc = mcpx.NewClient(mcpCfg)
	}

	return &ServiceContext{
		Config:           c,
		Registry:         registry,
		McpClient:        mc,
		AsyncResultStore: mcpx.NewMemoryAsyncResultStore(),
	}
}
