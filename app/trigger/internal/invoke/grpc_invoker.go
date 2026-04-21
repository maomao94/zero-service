package invoke

import (
	"context"
	"fmt"
	"net/http"
	"time"
	"zero-service/app/trigger/internal/svc"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/common/gtwx"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

type GRPCInvoker struct{}

func (g *GRPCInvoker) Execute(ctx context.Context, svcCtx *svc.ServiceContext, task *Task) *Result {
	start := time.Now()
	result := &Result{ID: task.ID}

	grpcServer := tool.MayReplaceLocalhost(task.GrpcServer)
	clientConf := zrpc.RpcClientConf{}
	conf.FillDefault(&clientConf)
	clientConf.Target = grpcServer
	clientConf.NonBlock = true
	clientConf.Timeout = 60000

	v, ok := svcCtx.ConnMap.Get(grpcServer)
	if !ok {
		conn, err := zrpc.NewClient(clientConf,
			zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor),
			zrpc.WithDialOption(grpc.WithDefaultCallOptions(grpc.ForceCodec(invokeRawCodec{}))))
		if err == nil {
			svcCtx.ConnMap.Set(grpcServer, conn)
			v = conn
		} else {
			logx.WithContext(ctx).Errorf("invoke grpc client init fail for %s: %v", grpcServer, err)
			result.Error = err.Error()
			result.StatusCode = http.StatusBadGateway
			result.CostMs = time.Since(start).Milliseconds()
			return result
		}
	}

	if v == nil {
		result.Error = "grpc connection is nil"
		result.StatusCode = http.StatusBadGateway
		result.CostMs = time.Since(start).Milliseconds()
		return result
	}

	cli := v.(*zrpc.RpcClient)
	var respBytes []byte
	zrpc.DontLogClientContentForMethod(task.Method)
	err := cli.Conn().Invoke(ctx, task.Method, task.Payload, &respBytes)

	result.CostMs = time.Since(start).Milliseconds()
	if err != nil {
		logx.WithContext(ctx).Errorf("invoke grpc failed: id=%s target=%s method=%s err=%v", task.ID, cli.Conn().Target(), task.Method, err)
		result.Error = err.Error()
		if st, ok := status.FromError(err); ok {
			result.StatusCode = int32(gtwx.GrpcCodeToHTTPStatus(st.Code()))
		}
		return result
	}

	result.Success = true
	result.StatusCode = int32(http.StatusOK)
	result.Data = respBytes
	return result
}

type invokeRawCodec struct{}

func (c invokeRawCodec) Marshal(v any) ([]byte, error) {
	return tool.ToProtoBytes(v)
}

func (c invokeRawCodec) Unmarshal(data []byte, v any) error {
	ba, ok := v.(*[]byte)
	if !ok {
		return fmt.Errorf("please pass in *[]byte")
	}
	*ba = append(*ba, data...)
	return nil
}

func (c invokeRawCodec) Name() string { return "invoke_raw" }
