package msgbody

import "go.opentelemetry.io/otel/propagation"

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
