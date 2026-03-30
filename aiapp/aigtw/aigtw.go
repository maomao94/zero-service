// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"zero-service/aiapp/aigtw/internal/config"
	"zero-service/aiapp/aigtw/internal/handler"
	"zero-service/aiapp/aigtw/internal/svc"

	"zero-service/common/ctxdata"
	"zero-service/common/ctxprop"
	"zero-service/common/gtwx"
	_ "zero-service/common/nacosx"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/aigtw.yaml", "the config file")

// serveStaticFile 返回一个静态文件服务 Handler
func serveStaticFile(baseDir, filename string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filePath := filepath.Join(baseDir, filename)
		if _, err := os.Stat(filePath); err == nil {
			http.ServeFile(w, r, filePath)
			return
		}
		http.NotFound(w, r)
	}
}

func main() {
	flag.Parse()

	// 设置 OpenAI 风格错误处理器
	gtwx.SetOpenAIErrorHandler()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// Print Go version
	tool.PrintGoVersion()

	server := rest.MustNewServer(c.RestConf, gtwx.CorsOption())

	// 全局中间件：将 Authorization header 注入 context，标记 auth-type=user，
	// 确保 gRPC 拦截器可通过 ctxdata.GetAuthorization(ctx) 传递原始 token。
	server.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if auth := r.Header.Get("Authorization"); auth != "" {
				ctx = context.WithValue(ctx, ctxdata.CtxAuthTypeKey, "user")
				ctx = context.WithValue(ctx, ctxdata.CtxAuthorizationKey, auth)
			}
			next(w, r.WithContext(ctx))
		}
	})

	// 全局中间件：将外部 JWT claim key 映射为内部标准 key。
	// server.Use 中间件在 go-zero JWT 中间件之后执行（见 rest/engine.go bindRoute），
	// 此时 JWT claims 已注入 context，可安全读取外部 key 并写入内部 key。
	if len(c.JwtAuth.ClaimMapping) > 0 {
		claimMapping := c.JwtAuth.ClaimMapping
		server.Use(func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				ctx := ctxprop.ApplyClaimMappingToCtx(r.Context(), claimMapping)
				next(w, r.WithContext(ctx))
			}
		})
	}

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	// 获取当前工作目录
	wd, _ := os.Getwd()
	staticDir := filepath.Join(wd, "aiapp", "aigtw")

	// 静态文件服务 - 支持 /chat.html, /tool.html, /results.html 等
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/chat.html",
		Handler: serveStaticFile(staticDir, "chat.html"),
	})

	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/tool.html",
		Handler: serveStaticFile(staticDir, "tool.html"),
	})

	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/results.html",
		Handler: serveStaticFile(staticDir, "results.html"),
	})

	// 根路径重定向到 chat.html
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filepath.Join(staticDir, "chat.html"))
		},
	})

	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	serviceGroup.Add(server)

	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	serviceGroup.Start()
}
