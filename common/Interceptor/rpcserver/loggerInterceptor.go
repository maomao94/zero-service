package interceptor

import (
	"context"
	"github.com/duke-git/lancet/v2/convertor"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func LoggerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}
	traceid := md.Get("x-b3-traceid")
	if traceid != nil {
		traceId := convertor.ToString(traceid[0])
		logx.WithContext(ctx).Infof("x-b3-traceid-%s", traceId)
	}
	spanid := md.Get("x-b3-spanid")
	if spanid != nil {
		spanId := convertor.ToString(spanid[0])
		logx.WithContext(ctx).Infof("x-b3-spanid-%s", spanId)
	}
	resp, err = handler(ctx, req)
	if err != nil {
		logx.WithContext(ctx).Errorf("【RPC-SRV-ERR】 %+v", err)
	}
	return resp, err
}
