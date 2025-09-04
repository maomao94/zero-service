package svc

import (
	"fmt"
	"net/http"
	"time"
	"zero-service/app/lalproxy/internal/config"

	"github.com/zeromicro/go-zero/rest/httpc"
)

type ServiceContext struct {
	Config     config.Config
	LalBaseUrl string        // LAL服务器基础URL
	LalClient  httpc.Service // 使用go-zero的httpc客户端
}

func NewServiceContext(c config.Config) *ServiceContext {
	timeout := time.Duration(c.LalServer.Timeout) * time.Millisecond
	// 创建带超时设置的HTTP客户端
	httpClient := &http.Client{
		Timeout: timeout,
	}
	lalClient := httpc.NewServiceWithClient(
		"httpc-lal",
		httpClient,
	)
	// 构建LAL服务器基础URL
	baseUrl := fmt.Sprintf("http://%s:%d", c.LalServer.Ip, c.LalServer.Port)
	return &ServiceContext{
		Config:     c,
		LalBaseUrl: baseUrl,
		LalClient:  lalClient,
	}
}
