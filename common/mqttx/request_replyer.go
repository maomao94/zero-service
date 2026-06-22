package mqttx

import (
	"context"
	"time"
)

// RequestReply performs a typed request/reply call through a reply router
// registered on client via WithReplyRouter using the same topic template.
// This is the preferred public entry point for typed request/reply.
func RequestReply[T any](ctx context.Context, c Client, topicTemplate string, tid string, send func() error, ttl ...time.Duration) (T, error) {
	var zero T
	if c == nil {
		return zero, ErrNoReplyRouter
	}
	replyClient, ok := c.(replyHandlerGetter)
	if !ok {
		return zero, ErrNoReplyRouter
	}
	handler := replyClient.getReplyHandler(topicTemplate)
	if handler == nil {
		return zero, ErrNoReplyRouter
	}
	router, ok := handler.(*ReplyRouter[T])
	if !ok {
		return zero, ErrReplyType
	}
	return router.RequestReply(ctx, tid, send, ttl...)
}
