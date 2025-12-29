package socketio

import (
	"net/http"
)

func SocketioHandler(server *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		server.HttpHandler().ServeHTTP(w, r)
	}
}
