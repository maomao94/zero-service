package gtwx

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest"
)

// CorsOption returns a rest.RunOption with standard CORS configuration.
// Usage: rest.MustNewServer(c.RestConf, gtwx.CorsOption())
func CorsOption() rest.RunOption {
	return rest.WithCustomCors(func(header http.Header) {
		origin := header.Get("Origin")
		if origin != "" {
			header.Set("Access-Control-Allow-Origin", origin)
		}
		header.Set("Vary", "Origin")

		header.Set("Access-Control-Allow-Credentials", "true")
		header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		header.Set("Access-Control-Allow-Headers", "Content-Type, AccessToken, X-CSRF-Token, Authorization, Token, X-Token, X-User-Id")
		header.Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")
	}, nil, "*")
}
