package svc

import (
	"zero-service/app/live/live"
	"zero-service/app/livegtw/internal/config"
	"zero-service/app/livegtw/internal/middleware"
	"zero-service/common/grpcx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config
	// LiveRpcCli app/live 会议业务服务客户端（身份经 grpcx metadata 透传）
	LiveRpcCli live.LiveRpcClient
	// MeetingAuth 会议鉴权中间件
	MeetingAuth rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))
	m := middleware.NewMeetingAuthMiddleware(c.JwtAuth.ClaimMapping)
	return &ServiceContext{
		Config: c,
		LiveRpcCli: live.NewLiveRpcClient(zrpc.MustNewClient(c.LiveRpcConf,
			zrpc.WithUnaryClientInterceptor(grpcx.UnaryMetadataInterceptor)).Conn()),
		MeetingAuth: m.Handle,
	}
}
