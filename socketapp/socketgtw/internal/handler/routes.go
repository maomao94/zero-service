package handler

import (
	"net/http"
	"zero-service/common/socketiox"
	"zero-service/socketapp/socketgtw/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	routes := []rest.Route{
		{
			Method:  http.MethodGet,
			Path:    "/socket.io",
			Handler: socketiox.SocketioHandler(serverCtx.SocketServer),
		},
	}
	var opts []rest.RouteOption
	//if len(serverCtx.Config.JwtAuth.AccessSecret) != 0 {
	//	opts = append(opts, rest.WithJwt(serverCtx.Config.JwtAuth.AccessSecret))
	//}
	server.AddRoutes(routes, opts...)
}
