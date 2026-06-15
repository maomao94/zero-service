package mqttx

import "errors"

var (
	// ErrNilDecoder is returned when a ReplyRouter is constructed with a nil decoder.
	ErrNilDecoder = errors.New("mqttx: reply decoder cannot be nil")
	// ErrEmptyReplyTid is returned when a decoded reply message has an empty tid.
	ErrEmptyReplyTid = errors.New("mqttx: reply message tid cannot be empty")
	// ErrNoReplyRouter is returned by RequestReply when no reply router
	// is registered for the given topic template.
	ErrNoReplyRouter = errors.New("mqttx: reply router not registered")
	// ErrReplyType is returned by RequestReply when the registered reply
	// router type does not match the caller's type parameter.
	ErrReplyType = errors.New("mqttx: reply router type mismatch")
	// ErrReplyNotMatched is returned by ReplyRouter.Consume when the reply message
	// was decoded but no pending request matched the tid.
	ErrReplyNotMatched = errors.New("mqttx: reply not matched")

	// ErrEmptyReplyID is kept for compatibility. Use ErrEmptyReplyTid.
	ErrEmptyReplyID = ErrEmptyReplyTid
)
