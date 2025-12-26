package socketio

import (
	"net/http"

	"github.com/zeromicro/go-zero/core/threading"
)

func SocketioHandler(server *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		threading.GoSafe(func() {
			server.HttpHandler().ServeHTTP(w, r)
		})
	}
}
