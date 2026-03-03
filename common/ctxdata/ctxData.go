package ctxdata

import (
	"context"
	"encoding/json"
	"strconv"

	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/metadata"
)

var CtxKeyUserId = "userId"
var CtxKeyUID = "uid"

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

func GetUserIdFromCtx(ctx context.Context, logError bool) string {
	var uid string
	if userId, ok := ctx.Value(CtxKeyUserId).(string); ok {
		uid = userId
	} else if userId, ok := ctx.Value(CtxKeyUserId).(json.Number); ok {
		uid = userId.String()
	} else if userId, ok := ctx.Value(CtxKeyUserId).(int64); ok {
		uid = strconv.FormatInt(userId, 10)
	} else if userId, ok := ctx.Value(CtxKeyUserId).(int); ok {
		uid = strconv.Itoa(userId)
	}
	return uid
}

func GetUserIdFromMetadata(ctx context.Context) string {
	var uid string
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}
	values := md.Get(CtxKeyUserId)
	if len(values) > 0 {
		uid = values[0]
	}
	return uid
}
