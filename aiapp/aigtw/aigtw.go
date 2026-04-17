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
	"strings"

	"zero-service/aiapp/aigtw/internal/config"
	"zero-service/aiapp/aigtw/internal/handler"
	"zero-service/aiapp/aigtw/internal/svc"

	"zero-service/common/ctxdata"
	"zero-service/common/ctxprop"
	"zero-service/common/gtwx"
	_ "zero-service/common/nacosx"
	"zero-service/common/tool"

	"github.com/google/uuid"
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
	if secret := os.Getenv("AIGTW_JWT_ACCESS_SECRET"); secret != "" {
		c.JwtAuth.AccessSecret = secret
	}
	if c.JwtAuth.AccessSecret == "" {
		fmt.Println("jwt access secret is empty, set JwtAuth.AccessSecret or AIGTW_JWT_ACCESS_SECRET")
		return
	}

	// Print Go version
	tool.PrintGoVersion()

	// 静态资源目录
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	staticRoot := filepath.Join(exeDir, "static")
	// fallback: 相对于当前工作目录
	if _, err := os.Stat(staticRoot); os.IsNotExist(err) {
		cwd, _ := os.Getwd()
		// 如果当前目录已经是 aigtw 目录，直接用 cwd/static
		if strings.HasSuffix(cwd, "aigtw") {
			staticRoot = filepath.Join(cwd, "static")
		} else {
			staticRoot = filepath.Join(cwd, "aiapp/aigtw/static")
		}
	}

	server := rest.MustNewServer(c.RestConf,
		gtwx.CorsOption(),
		rest.WithFileServer("/static", http.Dir(staticRoot)),
	)

	// 全局中间件
	server.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = uuid.NewString()
			}
			w.Header().Set("X-Request-Id", requestID)

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

	logx.Infof("static root: %s", staticRoot)

	server.AddRoutes([]rest.Route{
		{
			Method: http.MethodGet,
			Path:   "/",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/static/solo/index.html", http.StatusFound)
			},
		},
	})

	// 注册 API 路由
	handler.RegisterHandlers(server, ctx)

	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	serviceGroup.Add(server)

	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	serviceGroup.Start()
}
