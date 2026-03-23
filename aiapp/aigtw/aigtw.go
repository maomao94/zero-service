// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"

	"zero-service/aiapp/aigtw/internal/config"
	"zero-service/aiapp/aigtw/internal/handler"
	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"

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
			httpStatus, errType, errCode := grpcStatusToHTTP(st.Code())
			return httpStatus, &types.OpenAIError{
				HTTPStatus: httpStatus,
				ErrorMsg: types.ErrorDetail{
					Type:    errType,
					Message: st.Message(),
					Code:    errCode,
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

	server := rest.MustNewServer(c.RestConf, rest.WithCustomCors(func(header http.Header) {
		origin := header.Get("Origin")
		if origin != "" {
			header.Set("Access-Control-Allow-Origin", origin)
		}
		header.Set("Vary", "Origin")

		header.Set("Access-Control-Allow-Credentials", "true")
		header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		header.Set("Access-Control-Allow-Headers", "Content-Type, AccessToken, X-CSRF-Token, Authorization, Token, X-Token, X-User-Id")
		header.Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")
	}, nil, "*"))

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	serviceGroup.Add(server)

	logx.AddGlobalFields(logx.Field("app", c.Name))

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	serviceGroup.Start()
}

// grpcStatusToHTTP 将 gRPC status code 映射为 HTTP status + OpenAI error type/code
func grpcStatusToHTTP(code codes.Code) (httpStatus int, errType string, errCode string) {
	switch code {
	case codes.NotFound:
		return http.StatusNotFound, "invalid_request_error", "model_not_found"
	case codes.InvalidArgument:
		return http.StatusBadRequest, "invalid_request_error", "invalid_request"
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests, "rate_limit_error", "rate_limit_exceeded"
	case codes.PermissionDenied, codes.Unauthenticated:
		return http.StatusForbidden, "permission_error", "permission_denied"
	case codes.DeadlineExceeded:
		return http.StatusGatewayTimeout, "timeout_error", "timeout"
	case codes.Unavailable:
		return http.StatusBadGateway, "upstream_error", "upstream_unavailable"
	default:
		return http.StatusInternalServerError, "internal_error", ""
	}
}
