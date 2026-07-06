package gnetx

import (
	"errors"
	"time"

	"github.com/panjf2000/gnet/v2"
)

// ServerOptions 是构造 Server 的配置。通过 NewServer + With* 选项设置。
// 遵循项目 Client Option 构造配置边界规范：option 写入 Options 结构体，不直接改运行态。
type ServerOptions struct {
	// Addr 监听地址，如 "tcp://:9000" 或 ":9000"（无 scheme 时默认 tcp）。必填。
	Addr string

	// Codec 编解码器，承载分帧与序列化。必填。
	// 可用 NewLengthPrefixCodec/NewDelimiterCodec/NewFixedLengthCodec 快速构造，或自定义实现。
	Codec Codec

	// Handler 消息处理入口。必填。可为 HandlerFunc、Router 或自定义 Handler。
	// sync handler 在 event-loop 同步执行（必须快），async handler offload 到 gnet worker pool。
	Handler Handler

	// MaxFrameLength 单帧最大字节数。必填且必须 > 0，防御恶意/异常大帧。
	// NewServer 会把此值注入内置 Codec（LengthPrefixCodec/DelimiterCodec，仅当其未通过
	// WithMaxFrameSize 显式设置时），使上限真正生效。自定义 Codec 需自行在 Decode 里限制帧长。
	MaxFrameLength int

	// IdleTimeout 读空闲超时，超过则关闭连接。0 表示不检测。
	// 独立扫描 goroutine 周期检查 lastActiveAt（非 gnet OnTick，规避多核 N× 问题）。
	IdleTimeout time.Duration

	// SlowHandlerThreshold on-loop 同步 handler 慢处理告警阈值，超过打 logx 日志。
	// 0 用默认 50ms。async handler 不计入（已 offload）。
	SlowHandlerThreshold time.Duration

	// SessionListener 会话生命周期监听（OnCreated/OnRegistered/OnDestroyed）。nil 用 noop。
	SessionListener SessionListener

	// OnDecodeError 不可恢复解码错误的处理策略。0 用默认 DecodeErrorClose。
	OnDecodeError DecodeErrorAction

	// BatchReadLimit 单次 OnTraffic 最多解码的帧数。0 用默认 64。
	// 防止单连接一次可读事件占满 event-loop。到达上限且仍有剩余字节会 Wake 重触发。
	BatchReadLimit int

	// SequenceStart 连接级发送序号起始值。每个新 Session 的首次 NextSendSeq 返回此值。
	SequenceStart uint64

	// gnet 原生选项透传（详见 gnet.Options）
	Multicore       bool               // 是否多核（多 event-loop），默认单 loop
	NumEventLoop    int                // event-loop 数量，>0 时覆盖 Multicore
	LoadBalancing   gnet.LoadBalancing // 连接分配负载均衡策略（server-only）
	TCPKeepAlive    time.Duration      // TCP_KEEPIDLE
	TCPKeepInterval time.Duration      // TCP_KEEPINTVL
	TCPKeepCount    int                // TCP_KEEPCNT
	ReadBufferCap   int                // 每连接读缓冲（字节），默认 64KB，向上取整到 2^n
	WriteBufferCap  int                // 每连接写缓冲（字节），默认 64KB，向上取整到 2^n
}

// ServerOption 配置 ServerOptions 的函数式选项。
type ServerOption func(*ServerOptions)

// WithAddr 设置监听地址，如 "tcp://:9000" 或 ":9000"。
func WithAddr(addr string) ServerOption {
	return func(o *ServerOptions) { o.Addr = addr }
}

// WithCodec 设置编解码器。
func WithCodec(c Codec) ServerOption {
	return func(o *ServerOptions) { o.Codec = c }
}

// WithServerHandler 设置消息处理入口。
func WithServerHandler(h Handler) ServerOption {
	return func(o *ServerOptions) { o.Handler = h }
}

// WithMaxFrameLength 设置单帧最大字节数。
func WithMaxFrameLength(max int) ServerOption {
	return func(o *ServerOptions) { o.MaxFrameLength = max }
}

// WithIdleTimeout 设置读空闲超时，0 表示不检测。
func WithIdleTimeout(d time.Duration) ServerOption {
	return func(o *ServerOptions) { o.IdleTimeout = d }
}

// WithSlowHandlerThreshold 设置慢处理告警阈值。
func WithSlowHandlerThreshold(d time.Duration) ServerOption {
	return func(o *ServerOptions) { o.SlowHandlerThreshold = d }
}

// WithSessionListener 设置会话监听。
func WithSessionListener(l SessionListener) ServerOption {
	return func(o *ServerOptions) { o.SessionListener = l }
}

// WithBatchReadLimit 设置单次 OnTraffic 最多解码帧数。
func WithBatchReadLimit(n int) ServerOption {
	return func(o *ServerOptions) { o.BatchReadLimit = n }
}

// WithSequenceStart 设置连接级发送序号起始值。
func WithSequenceStart(n uint64) ServerOption {
	return func(o *ServerOptions) { o.SequenceStart = n }
}

// WithMulticore 是否多核（多 event-loop）。
func WithMulticore(b bool) ServerOption {
	return func(o *ServerOptions) { o.Multicore = b }
}

// WithNumEventLoop 设置 event-loop 数量（覆盖 Multicore）。
func WithNumEventLoop(n int) ServerOption {
	return func(o *ServerOptions) { o.NumEventLoop = n }
}

// WithLoadBalancing 设置连接负载均衡策略。
func WithLoadBalancing(lb gnet.LoadBalancing) ServerOption {
	return func(o *ServerOptions) { o.LoadBalancing = lb }
}

// WithTCPKeepAlive 设置 TCP keepalive（TCP_KEEPIDLE）。
func WithTCPKeepAlive(d time.Duration) ServerOption {
	return func(o *ServerOptions) { o.TCPKeepAlive = d }
}

// WithTCPKeepInterval 设置 TCP keepalive 探测间隔（TCP_KEEPINTVL）。
func WithTCPKeepInterval(d time.Duration) ServerOption {
	return func(o *ServerOptions) { o.TCPKeepInterval = d }
}

// WithTCPKeepCount 设置 TCP keepalive 探测次数（TCP_KEEPCNT）。
func WithTCPKeepCount(n int) ServerOption {
	return func(o *ServerOptions) { o.TCPKeepCount = n }
}

// WithReadBufferCap 设置每连接读缓冲容量。
func WithReadBufferCap(n int) ServerOption {
	return func(o *ServerOptions) { o.ReadBufferCap = n }
}

// WithWriteBufferCap 设置每连接写缓冲容量。
func WithWriteBufferCap(n int) ServerOption {
	return func(o *ServerOptions) { o.WriteBufferCap = n }
}

// WithOnDecodeError 设置不可恢复解码错误处理策略。
func WithOnDecodeError(a DecodeErrorAction) ServerOption {
	return func(o *ServerOptions) { o.OnDecodeError = a }
}

// slowHandlerWarning 是 on-loop 同步 handler 的慢处理告警阈值。
// 超过此阈值的 handler 会打 logx 慢处理日志。可通过 options 调整。
const defaultSlowHandlerThreshold = 50 * time.Millisecond

// defaultReconnectInterval 是 Client 断线后的默认固定重连间隔。
const defaultReconnectInterval = 3 * time.Second

// DecodeErrorAction 描述解码遇到不可恢复错误时的动作。
type DecodeErrorAction int

const (
	// DecodeErrorClose 关闭连接（默认）。
	DecodeErrorClose DecodeErrorAction = iota
	// DecodeErrorLogOnly 仅记日志，不关闭连接。
	DecodeErrorLogOnly
)

// validate 校验 ServerOptions 必填项。
func (o *ServerOptions) validate() error {
	if o.Addr == "" {
		return errors.New("gnetx: Addr is required")
	}
	if o.Codec == nil {
		return errors.New("gnetx: Codec is required")
	}
	if o.Handler == nil {
		return errors.New("gnetx: Handler is required")
	}
	if o.MaxFrameLength <= 0 {
		return errors.New("gnetx: MaxFrameLength must be positive")
	}
	return nil
}

// applyDefaults 填充零值字段为合理默认。
func (o *ServerOptions) applyDefaults() {
	if o.SlowHandlerThreshold <= 0 {
		o.SlowHandlerThreshold = defaultSlowHandlerThreshold
	}
	if o.BatchReadLimit <= 0 {
		o.BatchReadLimit = 64
	}
	if o.OnDecodeError == 0 {
		o.OnDecodeError = DecodeErrorClose
	}
}

// ClientOptions 是构造单连接 Client 的配置。
type ClientOptions struct {
	// Codec 编解码器。必填。与 server 端协议对应。
	Codec Codec

	// Handler 入站消息处理入口。必填。client 侧通常处理 server 主动推送或回包路由后的意外报文。
	Handler Handler

	// MaxFrameLength 单帧最大字节数。必填且 > 0。
	MaxFrameLength int

	// SlowHandlerThreshold on-loop 同步 handler 慢处理告警阈值。0 用默认 50ms。
	SlowHandlerThreshold time.Duration

	// OnDecodeError 不可恢复解码错误处理策略。0 用默认 DecodeErrorClose。
	OnDecodeError DecodeErrorAction

	// BatchReadLimit 单次 OnTraffic 最多解码帧数。0 用默认 64。
	BatchReadLimit int

	// SequenceStart 连接级发送序号起始值。每个新 Session 的首次 NextSendSeq 返回此值。
	SequenceStart uint64

	// ReconnectInterval 连接断开后的固定重连间隔。0 用默认 3s。
	ReconnectInterval time.Duration

	// OnReady 首次拨号成功回调（仅触发一次）。可为 nil。
	OnReady func(*Client)

	// HeartbeatInterval 应用层心跳发送间隔。0 表示不启用心跳。
	HeartbeatInterval time.Duration

	// HeartbeatMessage 心跳报文工厂，返回待编码的消息体。仅当 HeartbeatInterval > 0 且非 nil 时生效。
	HeartbeatMessage func() any

	// gnet 原生选项（单连接，无 Multicore/NumEventLoop）
	TCPKeepAlive    time.Duration // TCP_KEEPIDLE
	TCPKeepInterval time.Duration // TCP_KEEPINTVL
	TCPKeepCount    int           // TCP_KEEPCNT
	ReadBufferCap   int           // 读缓冲容量
	WriteBufferCap  int           // 写缓冲容量
}

// ClientOption 配置 ClientOptions 的函数式选项。
type ClientOption func(*ClientOptions)

// WithClientCodec 设置编解码器。
func WithClientCodec(c Codec) ClientOption {
	return func(o *ClientOptions) { o.Codec = c }
}

// WithClientHandler 设置消息处理入口。
func WithClientHandler(h Handler) ClientOption {
	return func(o *ClientOptions) { o.Handler = h }
}

// WithClientMaxFrameLength 设置单帧最大字节数。
func WithClientMaxFrameLength(max int) ClientOption {
	return func(o *ClientOptions) { o.MaxFrameLength = max }
}

// WithClientSlowHandlerThreshold 设置慢处理告警阈值。
func WithClientSlowHandlerThreshold(d time.Duration) ClientOption {
	return func(o *ClientOptions) { o.SlowHandlerThreshold = d }
}

// WithClientBatchReadLimit 设置单次 OnTraffic 最多解码帧数。
func WithClientBatchReadLimit(n int) ClientOption {
	return func(o *ClientOptions) { o.BatchReadLimit = n }
}

// WithClientSequenceStart 设置连接级发送序号起始值。
func WithClientSequenceStart(n uint64) ClientOption {
	return func(o *ClientOptions) { o.SequenceStart = n }
}

// WithClientReconnectInterval 设置断线后的固定重连间隔。
func WithClientReconnectInterval(d time.Duration) ClientOption {
	return func(o *ClientOptions) { o.ReconnectInterval = d }
}

// WithClientOnReady 设置首次拨号成功回调（仅触发一次）。
func WithClientOnReady(fn func(*Client)) ClientOption {
	return func(o *ClientOptions) { o.OnReady = fn }
}

// WithClientOnDecodeError 设置解码错误策略。
func WithClientOnDecodeError(a DecodeErrorAction) ClientOption {
	return func(o *ClientOptions) { o.OnDecodeError = a }
}

// WithClientTCPKeepAlive 设置 TCP keepalive。
func WithClientTCPKeepAlive(d time.Duration) ClientOption {
	return func(o *ClientOptions) { o.TCPKeepAlive = d }
}

// WithClientTCPKeepInterval 设置 TCP keepalive 探测间隔。
func WithClientTCPKeepInterval(d time.Duration) ClientOption {
	return func(o *ClientOptions) { o.TCPKeepInterval = d }
}

// WithClientTCPKeepCount 设置 TCP keepalive 探测次数。
func WithClientTCPKeepCount(n int) ClientOption {
	return func(o *ClientOptions) { o.TCPKeepCount = n }
}

// WithClientReadBufferCap 设置读缓冲容量。
func WithClientReadBufferCap(n int) ClientOption {
	return func(o *ClientOptions) { o.ReadBufferCap = n }
}

// WithClientWriteBufferCap 设置写缓冲容量。
func WithClientWriteBufferCap(n int) ClientOption {
	return func(o *ClientOptions) { o.WriteBufferCap = n }
}

// WithClientHeartbeatInterval 设置应用层心跳发送间隔。
func WithClientHeartbeatInterval(d time.Duration) ClientOption {
	return func(o *ClientOptions) { o.HeartbeatInterval = d }
}

// WithClientHeartbeatMessage 设置心跳报文工厂。
func WithClientHeartbeatMessage(fn func() any) ClientOption {
	return func(o *ClientOptions) { o.HeartbeatMessage = fn }
}

// validate 校验 ClientOptions 必填项。
func (o *ClientOptions) validate() error {
	if o.Codec == nil {
		return errors.New("gnetx: Codec is required")
	}
	if o.Handler == nil {
		return errors.New("gnetx: Handler is required")
	}
	if o.MaxFrameLength <= 0 {
		return errors.New("gnetx: MaxFrameLength must be positive")
	}
	return nil
}

// applyDefaults 填充零值字段为合理默认。
func (o *ClientOptions) applyDefaults() {
	if o.SlowHandlerThreshold <= 0 {
		o.SlowHandlerThreshold = defaultSlowHandlerThreshold
	}
	if o.BatchReadLimit <= 0 {
		o.BatchReadLimit = 64
	}
	if o.OnDecodeError == 0 {
		o.OnDecodeError = DecodeErrorClose
	}
	if o.ReconnectInterval <= 0 {
		o.ReconnectInterval = defaultReconnectInterval
	}
}
