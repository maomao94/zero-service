package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"zero-service/app/livegtw/internal/config"
	"zero-service/app/livegtw/internal/handler"
	"zero-service/app/livegtw/internal/handler/testpage"
	"zero-service/app/livegtw/internal/handler/webhook"
	"zero-service/app/livegtw/internal/svc"
	_ "zero-service/common/carbonx"
	"zero-service/common/gtwx"
	_ "zero-service/common/nacosx"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/livegtw.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	tool.PrintGoVersion()

	if secret := os.Getenv("LIVEGTW_JWT_ACCESS_SECRET"); secret != "" {
		c.JwtAuth.AccessSecret = secret
	}

	// gRPC 错误码 → HTTP 状态码统一转换（extproto reason 保留）
	gtwx.SetGrpcErrorHandler()

	server := rest.MustNewServer(c.RestConf, gtwx.CorsOption())

	ctx := svc.NewServiceContext(c)

	// 业务 API 路由组中间件：JWT 验证后运行，设置 auth-type + claims 桥接
	meetingAuth := handler.NewMeetingAuthMiddleware(c.JwtAuth.ClaimMapping)
	handler.RegisterHandlers(server, ctx, meetingAuth)

	// LiveKit webhook 接收（验签后转发 app/live，不走业务鉴权中间件）
	server.AddRoute(rest.Route{
		Method:  http.MethodPost,
		Path:    "/webhook/livekit",
		Handler: webhook.LiveKitWebhookHandler(ctx),
	})

	// HTML 测试页（免认证，不走业务鉴权中间件）
	if c.EnableTestPage {
		server.AddRoute(rest.Route{
			Method:  http.MethodGet,
			Path:    "/test/meeting",
			Handler: testpage.MeetingTestPageHandler(),
		})
	}

	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	serviceGroup.Add(server)
	serviceGroup.Start()
}
