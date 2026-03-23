// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"zero-service/aiapp/aigtw/internal/config"
	"zero-service/aiapp/aigtw/internal/handler"
	"zero-service/aiapp/aigtw/internal/svc"

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

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	// Demo page 静态文件路由
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/aigtw/demo",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			candidates := []string{}
			if exe, err := os.Executable(); err == nil {
				candidates = append(candidates, filepath.Join(filepath.Dir(exe), "sse_demo.html"))
			}
			candidates = append(candidates, "sse_demo.html", "aiapp/aigtw/sse_demo.html")

			for _, p := range candidates {
				if _, err := os.Stat(p); err == nil {
					http.ServeFile(w, r, p)
					return
				}
			}
			http.NotFound(w, r)
		},
	})

	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	serviceGroup.Add(server)

	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	serviceGroup.Start()
}
