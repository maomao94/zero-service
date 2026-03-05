package interceptor

import (
	"context"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func LoggerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if v := md.Get(ctxdata.HeaderUserId); len(v) > 0 {
		ctx = context.WithValue(ctx, ctxdata.CtxUserIdKey, v[0])
	}
	if v := md.Get(ctxdata.HeaderUserName); len(v) > 0 {
		ctx = context.WithValue(ctx, ctxdata.CtxUserNameKey, v[0])
	}
	if v := md.Get(ctxdata.HeaderAuthorization); len(v) > 0 {
		ctx = context.WithValue(ctx, ctxdata.CtxAuthorizationKey, v[0])
		ctx = logx.WithFields(ctx,
			logx.Field("header-auth", true),
		)
	} else {
		ctx = logx.WithFields(ctx,
			logx.Field("header-auth", false),
		)
	}
	if v := md.Get(ctxdata.HeaderTraceId); len(v) > 0 {
		ctx = context.WithValue(ctx, ctxdata.CtxTraceIdKey, v[0])
	}
	//traceid := md.Get("x-b3-traceid")
	//if traceid != nil {
	//	traceId := convertor.ToString(traceid[0])
	//	logx.WithContext(ctx).Infof("x-b3-traceid-%s", traceId)
	//}
	//spanid := md.Get("x-b3-spanid")
	//if spanid != nil {
	//	spanId := convertor.ToString(spanid[0])
	//	logx.WithContext(ctx).Infof("x-b3-spanid-%s", spanId)
	//}
	resp, err = handler(ctx, req)
	if err != nil {
		logx.WithContext(ctx).Errorf("【RPC-SRV-ERR】 %+v", err)
	}
	return resp, err
}
