// Package kqx 在 go-queue 的 Kafka 生产端之上补充按 key 哈希分区的选项。
package kqx

import (
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-queue/kq"
)

type options struct {
	pusherOpts []kq.PushOption
	ordered    bool
}

// Option 自定义生产端行为。
type Option func(*options)

// NewPusher 创建 Kafka 生产端。
//
// 分区策略保持 go-queue 的默认行为（LeastBytes），除非显式传入 WithOrdered。
// 是否保序由调用方决定，本函数不替业务做选择。
//
// 注意：go-queue 默认的 LeastBytes 不读取消息 key，同一 key 的消息会被
// 分散到多个分区，跨分区无法保证顺序。
func NewPusher(brokers []string, topic string, opts ...Option) *kq.Pusher {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	kqOpts := make([]kq.PushOption, 0, len(o.pusherOpts)+1)
	kqOpts = append(kqOpts, o.pusherOpts...)
	if o.ordered {
		kqOpts = append(kqOpts, kq.WithBalancer(&kafka.Hash{}))
	}
	return kq.NewPusher(brokers, topic, kqOpts...)
}

// WithOrdered 按完整消息 key 哈希分区，保证同一 key 的消息投递有序。
// key 为 nil 时 kafka.Hash 会退化为轮询；PushWithKey 传入空字符串得到的是
// 非 nil 的 []byte("")，仍会按哈希稳定落分区。
func WithOrdered() Option {
	return func(o *options) {
		o.ordered = true
	}
}

// WithSyncPush 同步推送，写入失败可直接感知。
// 默认异步批量推送，异步不影响同一分区内的顺序。
//
// 注意：kq 直接构造 kafka.Writer，RequiredAcks 保持零值（RequireNone），
// 同步推送只能感知传输层错误，拿不到 broker 的写入确认；逐条推送场景慎用。
func WithSyncPush() Option {
	return func(o *options) {
		o.pusherOpts = append(o.pusherOpts, kq.WithSyncPush())
	}
}

// WithAllowAutoTopicCreation 允许 topic 不存在时自动创建。
func WithAllowAutoTopicCreation() Option {
	return func(o *options) {
		o.pusherOpts = append(o.pusherOpts, kq.WithAllowAutoTopicCreation())
	}
}

// WithChunkSize 设置异步批量推送的单批字节数。
func WithChunkSize(size int) Option {
	return func(o *options) {
		o.pusherOpts = append(o.pusherOpts, kq.WithChunkSize(size))
	}
}

// WithFlushInterval 设置异步批量推送的最大攒批时长。
func WithFlushInterval(interval time.Duration) Option {
	return func(o *options) {
		o.pusherOpts = append(o.pusherOpts, kq.WithFlushInterval(interval))
	}
}
