package mqttx

import (
	"context"
	"time"

	"zero-service/common/antsx"
)

const defaultReplyRouterTTL = 30 * time.Second

// ReplyMessage is the protocol-neutral result decoded from a reply MQTT message.
// Tid is the request/reply unique message ID, aligned with antsx.ReplyPool logs.
type ReplyMessage[T any] struct {
	Tid   string
	Value T
}

// ReplyDecoder extracts a tid and typed value from a reply MQTT message.
// Protocol packages own topic conventions, payload schema, device identifiers, and result-code handling.
type ReplyDecoder[T any] interface {
	Decode(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[T], error)
}

// ReplyDecoderFunc adapts a function to ReplyDecoder.
type ReplyDecoderFunc[T any] func(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[T], error)

func (f ReplyDecoderFunc[T]) Decode(ctx context.Context, payload []byte, topic string, topicTemplate string) (ReplyMessage[T], error) {
	return f(ctx, payload, topic, topicTemplate)
}

type replyRouterOptions struct {
	defaultTTL time.Duration
	name       string
}

// ReplyRouterOption configures a ReplyRouter.
type ReplyRouterOption func(*replyRouterOptions)

// WithReplyRouterTTL sets the default TTL for pending reply entries.
func WithReplyRouterTTL(ttl time.Duration) ReplyRouterOption {
	return func(options *replyRouterOptions) {
		if ttl > 0 {
			options.defaultTTL = ttl
		}
	}
}

// WithReplyRouterName sets the underlying reply pool name for stats logs.
func WithReplyRouterName(name string) ReplyRouterOption {
	return func(options *replyRouterOptions) {
		if name != "" {
			options.name = name
		}
	}
}

// ReplyRouter matches MQTT replies to pending requests by a protocol-provided tid.
// Implements ConsumeHandler; reply/普通 handler 的区别由注册路径区分（WithReplyRouter vs Client.AddHandler）。
type ReplyRouter[T any] struct {
	pool   *antsx.ReplyPool[T]
	decode ReplyDecoder[T]
}

// NewReplyRouter creates a protocol-neutral MQTT reply router.
func NewReplyRouter[T any](decode ReplyDecoder[T], opts ...ReplyRouterOption) *ReplyRouter[T] {
	options := replyRouterOptions{
		defaultTTL: defaultReplyRouterTTL,
		name:       "mqttx-reply-router",
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	return &ReplyRouter[T]{
		pool:   antsx.NewReplyPool[T](antsx.WithDefaultTTL(options.defaultTTL), antsx.WithName(options.name)),
		decode: decode,
	}
}

// RequestReply registers tid, runs send, then waits for a matching reply.
// send is responsible for publishing the protocol-specific MQTT request.
func (r *ReplyRouter[T]) RequestReply(ctx context.Context, tid string, send func() error, ttl ...time.Duration) (T, error) {
	return antsx.RequestReply(ctx, r.pool, tid, send, ttl...)
}

// HandleReply decodes a reply MQTT message and resolves the matching pending request.
// The returned bool reports whether a pending request was actually resolved.
func (r *ReplyRouter[T]) HandleReply(ctx context.Context, payload []byte, topic string, topicTemplate string) (bool, error) {
	if r.decode == nil {
		return false, ErrNilDecoder
	}

	msg, err := r.decode.Decode(ctx, payload, topic, topicTemplate)
	if err != nil {
		return false, err
	}
	if msg.Tid == "" {
		return false, ErrEmptyReplyTid
	}

	return r.resolve(msg.Tid, msg.Value), nil
}

// Consume implements ConsumeHandler.
// Returns nil when the reply matched a pending request, ErrReplyNotMatched when the message
// was decoded but no pending entry existed, or the decode error otherwise.
func (r *ReplyRouter[T]) Consume(ctx context.Context, payload []byte, topic string, topicTemplate string) error {
	resolved, err := r.HandleReply(ctx, payload, topic, topicTemplate)
	if err != nil {
		return err
	}
	if !resolved {
		return ErrReplyNotMatched
	}
	return nil
}

// resolve resolves a pending request by tid.
func (r *ReplyRouter[T]) resolve(tid string, value T) bool {
	return r.pool.Resolve(tid, value)
}

// reject rejects a pending request by tid.
func (r *ReplyRouter[T]) reject(tid string, err error) bool {
	return r.pool.Reject(tid, err)
}

// has reports whether tid is currently pending.
func (r *ReplyRouter[T]) has(tid string) bool {
	return r.pool.Has(tid)
}

// Close closes the router and rejects all pending requests.
func (r *ReplyRouter[T]) Close() {
	r.pool.Close()
}
