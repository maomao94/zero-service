package socketio

import "net/http"

func SocketioHandler(server *Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		go server.HttpHandler().ServeHTTP(w, r)
	}
}
