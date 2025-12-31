package svc

import (
	"math"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/common/socketiox"
	"zero-service/facade/streamevent/streamevent"
	"zero-service/socketapp/socketgtw/internal/config"
	"zero-service/socketapp/socketgtw/internal/sockethandler"

	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type ServiceContext struct {
	Config         config.Config
	SocketServer   *socketiox.Server
	StreamEventCli streamevent.StreamEventClient
}

func NewServiceContext(c config.Config) *ServiceContext {
	svcCtx := &ServiceContext{
		Config: c,
		StreamEventCli: streamevent.NewStreamEventClient(zrpc.MustNewClient(c.StreamEventConf,
			zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
			// 添加最大消息配置
			zrpc.WithDialOption(grpc.WithDefaultCallOptions(
				grpc.MaxCallSendMsgSize(math.MaxInt32), // 发送最大2GB
				//grpc.MaxCallSendMsgSize(50 * 1024 * 1024),   // 发送最大50MB
				//grpc.MaxCallRecvMsgSize(100 * 1024 * 1024),  // 接收最大100MB
			)),
		).Conn()),
	}
	svcCtx.SocketServer = socketiox.MustServer(
		socketiox.WithContextKeys(c.SocketMetaData),
		socketiox.WithHandler(socketiox.EventUp, sockethandler.NewSocketUpHandler(svcCtx.StreamEventCli)),
	)
	return svcCtx
}
