package mqttx

import "errors"

var (
	// ErrNilDecoder is returned when a ReplyRouter is constructed with a nil decoder.
	ErrNilDecoder = errors.New("mqttx: reply decoder cannot be nil")
	// ErrEmptyReplyTid is returned when a decoded reply message has an empty tid.
	ErrEmptyReplyTid = errors.New("mqttx: reply message tid cannot be empty")
	// ErrNoReplyRouter is returned by ReplyRouter.RequestReply when the caller
	// holds no reply router (nil router or router not provided by the SDK).
	ErrNoReplyRouter = errors.New("mqttx: reply router not registered")
	// ErrReplyNotMatched is returned by ReplyRouter.Consume when the reply message
	// was decoded but no pending request matched the tid.
	ErrReplyNotMatched = errors.New("mqttx: reply not matched")

	// ErrEmptyReplyID is kept for compatibility. Use ErrEmptyReplyTid.
	ErrEmptyReplyID = ErrEmptyReplyTid
)
