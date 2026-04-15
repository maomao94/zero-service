// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
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

func main() {
	flag.Parse()

	// 设置 OpenAI 风格错误处理器
	gtwx.SetOpenAIErrorHandler()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// Print Go version
	tool.PrintGoVersion()

	server := rest.MustNewServer(c.RestConf, gtwx.CorsOption())

	// 全局中间件
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

	// 静态文件目录 - 使用相对路径（程序运行目录为aigtw根目录）
	staticDir := "."

	// 根路径和静态文件路由（需要在 RegisterHandlers 之前添加）
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filepath.Join(staticDir, "chat.html"))
		},
	})

	for _, name := range []string{"chat.html", "tool.html", "results.html", "solo.html"} {
		fname := name
		server.AddRoute(rest.Route{
			Method: http.MethodGet,
			Path:   "/" + fname,
			Handler: func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, filepath.Join(staticDir, fname))
			},
		})
	}

	// 注册 API 路由
	handler.RegisterHandlers(server, ctx)

	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	serviceGroup.Add(server)

	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	serviceGroup.Start()
}
