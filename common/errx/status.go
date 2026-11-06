package errx

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// grpcStatuser 是 gRPC 错误转换接口，*status.Error、gkiterrors.Error 等均实现。
type grpcStatuser interface {
	GRPCStatus() *status.Status
}

// ToGRPCStatus 是 gRPC 服务端出口的最后一道归一化。
//
// 它只兜底“没有走统一错误入口”的错误，不重写调用方显式构造的 status：
//   - context 取消/超时保留 gRPC 语义，避免被误判为 Internal；
//   - 已实现 GRPCStatus 的错误取内层 status 返回，避免 fmt.Errorf 包装后
//     grpc-go 用 err.Error() 覆盖 message 把 cause 带出去；
//   - 其余裸错误只记录日志，对外统一返回系统繁忙。
func ToGRPCStatus(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return status.FromContextError(err).Err()
	}

	if gs, ok := err.(grpcStatuser); ok && gs.GRPCStatus() != nil {
		return gs.GRPCStatus().Err()
	}

	var gs grpcStatuser
	if errors.As(err, &gs) && gs.GRPCStatus() != nil {
		return gs.GRPCStatus().Err()
	}

	logx.Errorf("未转换的 RPC 错误，已脱敏：%+v", err)
	return status.Error(codes.Internal, SystemBusyMessage)
}
