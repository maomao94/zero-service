package broadcast

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"zero-service/common/mqttx"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/jsonx"
)

// Broadcaster 广播集群客户端：封装 fire-and-forget 与等待 ack 两种发送模式，
// 以及消费分发（防回环 + method→executor 路由 + ack 回发）。
//
// 协议隔离：调用方只暴露 (method, payload []byte)，SDK 在发送时构造完整
// BroadcastBody（Tid 自动生成、AckTopic=本实例 ack 主题、Method=参数）与
// 完整 BroadcastAckBody（Success/Error/ErrorKind/ResponseBody 由消费骨架填充），
// 调用方与主题/协议字段零接触。
type Broadcaster interface {
	// Broadcast 以 fire-and-forget 方式发布广播请求，不等待 ack。
	// SDK 内部自动生成 Tid 并填充本实例 ack 主题。
	Broadcast(ctx context.Context, method string, payload []byte) error

	// BroadcastReply 发布广播请求并等待执行节点的 ack。
	// SDK 内部自动生成 Tid（与 pending 条目一致）并填充本实例 ack 主题；
	// timeout <= 0 时使用默认 TTL（10s，可通过 WithReplyTTL 配置）。
	// 成功返回业务结果字节（ack.ResponseBody）；ack.Success==false 时按
	// ack.ErrorKind 还原为领域错误（ErrorFromKind）。
	BroadcastReply(ctx context.Context, method string, payload []byte, timeout time.Duration) ([]byte, error)

	// AddBroadcastHandler 将消费处理挂到 mqttx.Client 上（订阅 BroadcastTopicPattern(prefix)）。
	AddBroadcastHandler() error

	// AddExecutor 注册 method 业务执行器。
	AddExecutor(method string, fn Executor)

	// ConsumeBroadcast 广播消费入口（mqttx.ConsumeHandler 签名），见 dispatcher.go。
	ConsumeBroadcast(ctx context.Context, payload []byte, topic string, topicTemplate string) error
}

// BroadcasterOption 配置 NewBroadcaster。
type BroadcasterOption func(*broadcasterOptions)

type broadcasterOptions struct {
	prefix   string
	replyTTL time.Duration
}

// WithPrefix 设置广播主题前缀（如 "oryx/server"、"iec"）。
func WithPrefix(prefix string) BroadcasterOption {
	return func(opts *broadcasterOptions) {
		opts.prefix = prefix
	}
}

// WithReplyTTL 设置 BroadcastReply 的默认等待时间（timeout<=0 时生效）。
func WithReplyTTL(ttl time.Duration) BroadcasterOption {
	return func(opts *broadcasterOptions) {
		if ttl > 0 {
			opts.replyTTL = ttl
		}
	}
}

type broadcaster struct {
	client     mqttx.Client
	instanceID string
	options    broadcasterOptions

	mu        sync.RWMutex
	executors map[string]Executor
}

// NewBroadcaster 创建广播客户端。
//   - c: 已创建的 mqttx.Client（ack 应答 router 需在创建时通过
//     mqttx.WithReplyRouter(BroadcastAckTopic(prefix, instanceID), NewAckReplyRouter(...)) 注册，
//     否则 BroadcastReply 返回 mqttx.ErrNoReplyRouter）
//   - instanceID: 本实例唯一 ID（如 "oryx-relay-{uid}"），用于 ack 通道隔离与防回环
//
// 必须通过 WithPrefix 提供业务前缀（SDK 不硬编码业务前缀）。
func NewBroadcaster(c mqttx.Client, instanceID string, opts ...BroadcasterOption) Broadcaster {
	options := broadcasterOptions{replyTTL: defaultAckReplyTTL}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	return &broadcaster{
		client:     c,
		instanceID: instanceID,
		options:    options,
		executors:  make(map[string]Executor),
	}
}

// Broadcast 发布广播请求（fire-and-forget，不等待 ack）。
// SDK 内部生成 Tid、填充 AckTopic=本实例 ack 主题，调用方只见 method + payload。
func (b *broadcaster) Broadcast(ctx context.Context, method string, payload []byte) error {
	if b.client == nil {
		return errors.New("broadcast: mqtt client is nil")
	}
	body, err := b.wrapRequest(method, payload)
	if err != nil {
		return err
	}
	return b.publishBroadcast(ctx, body)
}

// BroadcastReply 发布广播请求并等待执行节点 ack。
// Tid 由 SDK 生成并同时用于 pending 注册与发布（保证 ack 关联）；返回业务结果字节，
// ack.Success==false 时按 errorKind 还原为领域错误。
func (b *broadcaster) BroadcastReply(ctx context.Context, method string, payload []byte, timeout time.Duration) ([]byte, error) {
	if b.client == nil {
		return nil, errors.New("broadcast: mqtt client is nil")
	}
	body, err := b.wrapRequest(method, payload)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = b.options.replyTTL
	}
	ack, err := mqttx.RequestReply[*BroadcastAckBody](ctx, b.client, b.ackTopic(), body.Tid, func() error {
		return b.publishBroadcast(ctx, body)
	}, timeout)
	if err != nil {
		return nil, err
	}
	return replyResult(ack)
}

// wrapRequest 构造完整 BroadcastBody（Tid 自动生成、AckTopic=本实例 ack 主题、Method=参数）。
func (b *broadcaster) wrapRequest(method string, payload []byte) (*BroadcastBody, error) {
	tid, err := tool.SimpleUUID()
	if err != nil {
		return nil, fmt.Errorf("generate broadcast tid failed: %w", err)
	}
	return &BroadcastBody{
		Tid:      tid,
		AckTopic: b.ackTopic(),
		Method:   method,
		Body:     string(payload),
	}, nil
}

// publishBroadcast 将构造好的 broadcast body 以 PublishWithTrace 发布到广播主题。
func (b *broadcaster) publishBroadcast(ctx context.Context, body *BroadcastBody) error {
	byteData, err := jsonx.Marshal(body)
	if err != nil {
		return fmt.Errorf("broadcast marshal error: %w", err)
	}
	pushCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := b.client.PublishWithTrace(pushCtx, b.broadcastTopic(), byteData); err != nil {
		return fmt.Errorf("publish broadcast to mqtt: %w", err)
	}
	return nil
}

// replyResult 将 ack 归一为调用方可见结果：成功返回业务结果字节，失败按 errorKind 还原领域错误。
func replyResult(ack *BroadcastAckBody) ([]byte, error) {
	if !ack.Success {
		return nil, ErrorFromKind(ack.ErrorKind, ack.Error)
	}
	return []byte(ack.ResponseBody), nil
}

// AddBroadcastHandler 将本实例的广播消费注册到 mqttx.Client。
func (b *broadcaster) AddBroadcastHandler() error {
	if b.client == nil {
		return errors.New("broadcast: mqtt client is nil")
	}
	return b.client.AddHandlerFunc(BroadcastTopicPattern(b.options.prefix), b.ConsumeBroadcast)
}

// AddExecutor 注册 method 业务执行器（重复注册后者覆盖前者）。
func (b *broadcaster) AddExecutor(method string, fn Executor) {
	if method == "" || fn == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.executors[method] = fn
}

// broadcastTopic 集群广播主题（{prefix}/broadcast）。
func (b *broadcaster) broadcastTopic() string {
	return BroadcastTopic(b.options.prefix)
}

// ackTopic 本实例 ack 通道主题。
func (b *broadcaster) ackTopic() string {
	return BroadcastAckTopic(b.options.prefix, b.instanceID)
}
