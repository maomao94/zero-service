package handler

import (
	"net/http"
	"zero-service/common/socketio"
	"zero-service/socketapp/socketgtw/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/socket.io",
				Handler: socketio.SocketioHandler(serverCtx.SocketServer),
			},
		},
		rest.WithJwt(serverCtx.Config.JwtAuth.AccessSecret),
	)
}
