package interceptor

import (
	"context"
	"github.com/spf13/cast"
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
		traceId := cast.ToString(traceid[0])
		logx.WithContext(ctx).Infof("x-b3-traceid-%s", traceId)
	}
	spanid := md.Get("x-b3-spanid")
	if spanid != nil {
		spanId := cast.ToString(spanid[0])
		logx.WithContext(ctx).Infof("x-b3-spanid-%s", spanId)
	}
	resp, err = handler(ctx, req)
	if err != nil {
		logx.WithContext(ctx).Errorf("【RPC-SRV-ERR】 %+v", err)
		//causeErr := errors.Cause(err)
		//if e, ok := causeErr.(*errorx.CodeError); ok {
		//	logx.WithContext(ctx).Errorf("【RPC-SRV-ERR】 %+v", err)
		//	metadata := make(map[s`tring]string)
		//	metadata["errorCode"] = mapping.Repr(e.Code)
		//	metadata["message"] = e.Message
		//	errInfo := &errdetails.ErrorInfo{
		//		Reason:   e.Message,
		//		Domain:   "http://zero",
		//		Metadata: metadata,
		//	}
		//	var details []proto.Message
		//	details = append(details, errInfo)
		//	st, _ := status.New(codes.Internal, fmt.Sprintf("%d, %s", e.Code, e.Message)).WithDetails(details...)
		//	err = st.Err()
		//} else {
		//	logx.WithContext(ctx).Errorf("【RPC-SRV-ERR】 %+v", err)
		//}
	}
	return resp, err
}
