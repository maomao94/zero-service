// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package main

import (
	"flag"
	"fmt"
	"net/http"

	"zero-service/app/oryxgtw/internal/config"
	"zero-service/app/oryxgtw/internal/handler"
	"zero-service/app/oryxgtw/internal/svc"
	_ "zero-service/common/carbonx"
	"zero-service/common/nacosx"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/oryxgtw.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf, rest.WithCustomCors(func(header http.Header) {
		origin := header.Get("Origin") // 动态获取请求域名
		if origin != "" {
			header.Set("Access-Control-Allow-Origin", origin) // 指定允许的域
		}
		header.Set("Vary", "Origin") // 避免缓存污染

		header.Set("Access-Control-Allow-Credentials", "true")                                                                          // 允许携带 Cookie/Token
		header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")                                            // 支持的请求方法
		header.Set("Access-Control-Allow-Headers", "Content-Type, AccessToken, X-CSRF-Token, Authorization, Token, X-Token, X-User-Id") // 支持的请求头
		header.Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")                                                     // 前端可以读取的响应头

	}, nil, "*"))
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	// register service to nacos
	if c.NacosConfig.IsRegister {
		sc := []constant.ServerConfig{
			*constant.NewServerConfig(c.NacosConfig.Host, c.NacosConfig.Port),
		}
		cc := &constant.ClientConfig{
			NamespaceId:         c.NacosConfig.NamespaceId,
			Username:            c.NacosConfig.Username,
			Password:            c.NacosConfig.PassWord,
			TimeoutMs:           5000,
			NotLoadCacheAtStart: true,
		}
		m := map[string]string{
			"preserved.register.source": "go-zero",
		}
		opts := nacosx.NewNacosConfig(c.NacosConfig.ServiceName, fmt.Sprintf("%s:%d", c.Host, c.Port), sc, cc, nacosx.WithMetadata(m))
		_ = nacosx.RegisterService(opts)
	}

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
