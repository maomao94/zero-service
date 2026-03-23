// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"zero-service/aiapp/aigtw/internal/config"
	"zero-service/aiapp/aigtw/internal/handler"
	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"

	"zero-service/common/gtwx"
	_ "zero-service/common/nacosx"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var configFile = flag.String("f", "etc/aigtw.yaml", "the config file")

func main() {
	flag.Parse()

	// 设置 OpenAI 风格错误处理器
	httpx.SetErrorHandlerCtx(func(ctx context.Context, err error) (int, any) {
		var openAIErr *types.OpenAIError
		if errors.As(err, &openAIErr) {
			return openAIErr.HTTPStatus, openAIErr
		}

		// 处理 gRPC status error
		if st, ok := status.FromError(err); ok {
			httpStatus := gtwx.GrpcCodeToHTTPStatus(st.Code())
			return httpStatus, &types.OpenAIError{
				HTTPStatus: httpStatus,
				ErrorMsg: types.ErrorDetail{
					Type:    grpcCodeToOpenAIType(st.Code()),
					Message: st.Message(),
					Code:    grpcCodeToOpenAICode(st.Code()),
				},
			}
		}

		return http.StatusInternalServerError, &types.OpenAIError{
			HTTPStatus: http.StatusInternalServerError,
			ErrorMsg: types.ErrorDetail{
				Type:    "internal_error",
				Message: err.Error(),
			},
		}
	})

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

// grpcCodeToOpenAIType 将 gRPC status code 映射为 OpenAI error type
func grpcCodeToOpenAIType(code codes.Code) string {
	switch code {
	case codes.NotFound:
		return "invalid_request_error"
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return "invalid_request_error"
	case codes.Unauthenticated:
		return "authentication_error"
	case codes.PermissionDenied:
		return "permission_error"
	case codes.ResourceExhausted:
		return "rate_limit_error"
	case codes.DeadlineExceeded, codes.Canceled:
		return "timeout_error"
	case codes.Unavailable:
		return "upstream_error"
	case codes.AlreadyExists, codes.Aborted:
		return "conflict_error"
	default:
		return "internal_error"
	}
}

// grpcCodeToOpenAICode 将 gRPC status code 映射为 OpenAI error code
func grpcCodeToOpenAICode(code codes.Code) string {
	switch code {
	case codes.NotFound:
		return "model_not_found"
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		return "invalid_request"
	case codes.Unauthenticated:
		return "invalid_api_key"
	case codes.PermissionDenied:
		return "permission_denied"
	case codes.ResourceExhausted:
		return "rate_limit_exceeded"
	case codes.DeadlineExceeded, codes.Canceled:
		return "timeout"
	case codes.Unavailable:
		return "upstream_unavailable"
	case codes.AlreadyExists, codes.Aborted:
		return "conflict"
	default:
		return ""
	}
}
