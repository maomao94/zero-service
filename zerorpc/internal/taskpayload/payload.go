package taskpayload

import "go.opentelemetry.io/otel/propagation"

type HttpPayload struct {
	MsgId   string                     `json:"msgId,omitempty"`
	Carrier *propagation.HeaderCarrier `json:"carrier"`
	Msg     string                     `json:"msg,omitempty"`
	Url     string                     `json:"url" validate:"required"`
}
