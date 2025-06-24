package ctxdata

import (
	"context"
	"encoding/json"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/metadata"
	"strconv"
)

var CtxKeyUserId = "userId"

type MsgBody struct {
	MsgId   string                     `json:"msgId,omitempty"`
	Carrier *propagation.HeaderCarrier `json:"carrier"`
	Msg     string                     `json:"msg,omitempty"`
	Url     string                     `json:"url" validate:"required"`
}

type ProtoMsgBody struct {
	MsgId          string                     `json:"msgId,omitempty"`
	Carrier        *propagation.HeaderCarrier `json:"carrier"`
	GrpcServer     string                     `json:"grpcServer" validate:"required"`
	Method         string                     `json:"method" validate:"required"`
	Payload        string                     `json:"payload" validate:"required"`
	RequestTimeout int64                      `json:"requestTimeout"`
}

func GetUserIdFromCtx(ctx context.Context, bool bool) int64 {
	var uid int64
	if userId, ok := ctx.Value(CtxKeyUserId).(json.Number); ok {
		if int64UserId, err := userId.Int64(); err == nil {
			uid = int64UserId
		} else {
			if bool {
				logx.WithContext(ctx).Errorf("GetUserIdFromCtx err : %+v", err)
			}
		}
	}
	return uid
}

func GetUserIdFromMetadata(ctx context.Context) int64 {
	var uid int64
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}
	values := md.Get(CtxKeyUserId)
	if values != nil {
		uid, _ = strconv.ParseInt(values[0], 10, 64)
	}
	return uid
}
