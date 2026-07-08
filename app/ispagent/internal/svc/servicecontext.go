package svc

import (
	"zero-service/app/ispagent/internal/config"
	"zero-service/app/ispagent/internal/ispclient"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
)

// ServiceContext 为 ispagent 的依赖注入容器，持有配置和 ISP TCP 客户端管理器。
type ServiceContext struct {
	Config    config.Config
	IspClient *ispclient.Manager
}

// NewServiceContext 创建 ServiceContext 并注册 ISP 客户端关闭回调。
func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))
	m := ispclient.NewManager(c.IspSetting)
	proc.AddShutdownListener(func() { m.Close() })
	return &ServiceContext{
		Config:    c,
		IspClient: m,
	}
}
