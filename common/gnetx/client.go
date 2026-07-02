package gnetx

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/panjf2000/gnet/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/timex"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Client 是 gnetx 的单连接 TCP 客户端。构造即拨号到指定远端，断线默认按固定间隔自动重连。
// 对标 mqttx/modbusx 的 MustNewClient 模型：无 Start/Stop，只有构造 + Close。
//
// 一个 Client 对应一个远端连接。需要连多个远端时，构造多个 Client。
//
// 用法：
//
//	cli := gnetx.MustNewClient("tcp", "127.0.0.1:9000",
//	    gnetx.WithClientCodec(myCodec),
//	    gnetx.WithClientHandler(myHandler),
//	    gnetx.WithClientMaxFrameLength(1<<20),
//	)
//	defer cli.Close()
//	_ = cli.Notify(ctx, msg)                  // fire-and-forget 发送
//	resp, _ := cli.Request(ctx, req, 5*time.Second) // 响应式：发请求等回包（tid 关联）
//
// 响应式编程：msg 实现 Correlatable（提供 TID）时，Request 通过 antsx.ReplyPool 按 tid
// 关联回包，回包实现 Response（提供 ResponseTID）即自动匹配返回，无需手动管理在途请求。
//
// 生命周期：MustNewClient/NewClient（拨号）→ 收发 →（断线自动重连）→ Close。
type Client struct {
	gnet.BuiltinEventEngine

	opts ClientOptions
	gcli *gnet.Client
	pool *workerPool

	network string
	address string

	sess         atomic.Pointer[Session] // 当前活跃连接，断开期间为 nil
	ready        atomic.Bool             // 首次拨号成功后置 true（OnReady 只触发一次）
	closed       atomic.Bool             // Close 后置 true，阻止后续重连
	reconnecting atomic.Bool             // 重连 goroutine 是否在跑
	reconnectCh  chan struct{}           // Close 时关闭，通知重连 goroutine 退出

	tracer oteltrace.Tracer
}

// MustNewClient 创建 Client 并立即拨号到 address，失败 panic。
// 默认启用固定间隔自动重连，并注册 proc 关闭监听（进程退出时 Close）。
func MustNewClient(network, address string, opts ...ClientOption) *Client {
	cli, err := NewClient(network, address, opts...)
	if err != nil {
		panic("gnetx: MustNewClient " + network + "/" + address + ": " + err.Error())
	}
	proc.AddShutdownListener(func() {
		cli.Close()
	})
	return cli
}

// NewClient 创建 Client 并立即拨号到 address，返回首次拨号错误。
// 拨号成功后建立连接；后续断线由内部固定间隔自动重连（默认 3s）。
func NewClient(network, address string, opts ...ClientOption) (*Client, error) {
	o := &ClientOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
	if err := o.validate(); err != nil {
		return nil, err
	}
	o.applyDefaults()

	// 把强制的 MaxFrameLength 注入内置 codec（若其未显式设 WithMaxFrameSize）。
	applyFrameLimit(o.Codec, o.MaxFrameLength)

	c := &Client{
		opts:        *o,
		pool:        defaultWorkerPool(),
		network:     network,
		address:     address,
		reconnectCh: make(chan struct{}),
		tracer:      gnetxTracer(),
	}

	gcli, err := gnet.NewClient(c, c.buildGnetOptions()...)
	if err != nil {
		return nil, err
	}
	if err := gcli.Start(); err != nil {
		return nil, err
	}
	c.gcli = gcli

	if err := c.dial(); err != nil {
		_ = gcli.Stop()
		return nil, err
	}
	return c, nil
}

// Session 返回当前活跃连接的 Session；从未连上或正在重连时返回 nil。
// 重连成功后此接口自动返回新的 Session。
func (c *Client) Session() *Session {
	return c.sess.Load()
}

// Send 通过当前连接编码并发送消息（fire-and-forget，off-loop 安全）。
// 未连接或重连中返回 ErrSessionClosed。
func (c *Client) Send(msg any) error {
	sess := c.sess.Load()
	if sess == nil {
		return ErrSessionClosed
	}
	return sess.Send(msg)
}

// Notify 是 Send 的语义别名（带 ctx，对齐 Session.Notify）。
func (c *Client) Notify(ctx context.Context, msg any) error {
	sess := c.sess.Load()
	if sess == nil {
		return ErrSessionClosed
	}
	return sess.Notify(ctx, msg)
}

// Request 响应式请求：通过当前连接发送 msg 并等待匹配 tid 的回包（opt-in 请求-响应）。
// msg 需实现 Correlatable，回包需实现 Response 且 ResponseTID 与 msg.TID 一致。
// 未连接或重连中返回 ErrSessionClosed；超时/取消由 ctx 与 ttl 控制。
// 只能在业务 goroutine 调用（会阻塞等待回包），严禁在 handler 的 on-loop 同步路径里调用。
func (c *Client) Request(ctx context.Context, msg Correlatable, ttl time.Duration) (any, error) {
	sess := c.sess.Load()
	if sess == nil {
		return nil, ErrSessionClosed
	}
	return sess.Request(ctx, msg, ttl)
}

// Close 关闭连接并停止自动重连。幂等。
func (c *Client) Close() {
	if !c.closed.CompareAndSwap(false, true) {
		return
	}
	close(c.reconnectCh)
	if sess := c.sess.Swap(nil); sess != nil {
		_ = sess.Close()
	}
	if c.gcli != nil {
		_ = c.gcli.Stop()
	}
	logx.Infof("[gnetx] client closed %s/%s", c.network, c.address)
}

// dial 拨号建立连接。用 DialContext 让 Session 在 OnOpen（event-loop 线程）里创建并绑定，
// 避免 SetContext 与 event-loop 的 OnClose/OnTraffic 竞争。DialContext 返回时 OnOpen 已完成，
// c.sess 已就绪。首次成功后在 off-loop 触发 OnReady（避免阻塞 event-loop）。
func (c *Client) dial() error {
	if _, err := c.gcli.DialContext(c.network, c.address, nil); err != nil {
		return err
	}
	if c.ready.CompareAndSwap(false, true) && c.opts.OnReady != nil {
		c.opts.OnReady(c)
	}
	return nil
}

// OnOpen 实现 gnet.EventHandler。在 event-loop 线程创建 Session 并绑定 conn.Context，
// 存入 c.sess。DialContext 在注册前已设好 gc.ctx，故此处 SetContext 与后续事件无竞争。
func (c *Client) OnOpen(conn gnet.Conn) ([]byte, gnet.Action) {
	sess := newSession(sessionIDForConn(conn), conn, c.opts.Codec, nil, true)
	conn.SetContext(sess)
	c.sess.Store(sess)
	if c.opts.SessionListener != nil {
		c.opts.SessionListener.OnCreated(sess)
	}
	return nil, gnet.None
}

// OnTraffic 实现 gnet.EventHandler。解码 → Response 自动路由 → handler 分发。
func (c *Client) OnTraffic(conn gnet.Conn) gnet.Action {
	sess, _ := conn.Context().(*Session)
	if sess == nil {
		return gnet.Close
	}
	sess.touch()

	batchLimit := c.opts.BatchReadLimit
	consumed := 0
	for i := 0; i < batchLimit; i++ {
		msg, err := c.opts.Codec.Decode(conn, sess)
		if err != nil {
			if errors.Is(err, ErrIncompletePacket) {
				break
			}
			return c.handleDecodeError(sess, err)
		}
		consumed++
		if resp, ok := msg.(Response); ok {
			if sess.resolveResponse(resp.ResponseTID(), msg) {
				continue
			}
		}
		c.dispatch(sess, msg)
	}
	if consumed > 0 && conn.InboundBuffered() > 0 {
		_ = conn.Wake(nil)
	}
	return gnet.None
}

// OnClose 实现 gnet.EventHandler。清理 Session，未主动关闭时触发自动重连。
func (c *Client) OnClose(conn gnet.Conn, err error) gnet.Action {
	sess, _ := conn.Context().(*Session)
	if sess == nil {
		return gnet.None
	}
	cause := "closed"
	if err != nil {
		cause = err.Error()
	}
	logSessionClosed(sess, cause)
	c.sess.CompareAndSwap(sess, nil)
	_ = sess.Close()
	if c.opts.SessionListener != nil {
		c.opts.SessionListener.OnDestroyed(sess)
	}
	if !c.closed.Load() {
		c.startReconnect()
	}
	return gnet.None
}

// startReconnect 启动固定间隔重连 goroutine（幂等，最多一个在跑）。
func (c *Client) startReconnect() {
	if !c.reconnecting.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer c.reconnecting.Store(false)
		ticker := time.NewTicker(c.opts.ReconnectInterval)
		defer ticker.Stop()
		for {
			select {
			case <-c.reconnectCh:
				return
			case <-ticker.C:
			}
			if c.closed.Load() {
				return
			}
			logx.Infof("[gnetx] reconnecting %s/%s ...", c.network, c.address)
			if err := c.dial(); err != nil {
				logx.Errorf("[gnetx] reconnect %s/%s failed: %v", c.network, c.address, err)
				continue
			}
			logx.Infof("[gnetx] reconnected %s/%s", c.network, c.address)
			return
		}
	}()
}

func (c *Client) dispatch(sess *Session, msg any) {
	h := c.opts.Handler
	if isAsync(h) {
		c.dispatchAsync(sess, msg, h)
		return
	}
	c.dispatchSync(sess, msg, h)
}

func (c *Client) dispatchSync(sess *Session, msg any, h Handler) {
	startTime := timex.Now()
	ctx, span := startClientSpan(c.tracer, sess, msg)
	defer span.End()

	reply, hErr := h.Handle(ctx, sess, msg)

	duration := timex.Since(startTime)
	if duration > c.opts.SlowHandlerThreshold {
		logx.Slowf("[gnetx] client slow handler %s id=%s", duration, sess.id)
	}
	if hErr != nil {
		span.RecordError(hErr)
		logx.Errorf("[gnetx] client handler error: %v", hErr)
		return
	}
	if reply != nil {
		if err := c.writeReply(sess, reply); err != nil {
			logx.Errorf("[gnetx] client write reply error: %v", err)
		}
	}
}

func (c *Client) dispatchAsync(sess *Session, msg any, h Handler) {
	ctx, span := startClientSpan(c.tracer, sess, msg)

	err := c.pool.Submit(func() {
		defer span.End()
		reply, hErr := h.Handle(ctx, sess, msg)
		if hErr != nil {
			span.RecordError(hErr)
			logx.Errorf("[gnetx] client async handler error: %v", hErr)
			return
		}
		if reply != nil {
			if err := sess.Send(reply); err != nil {
				logx.Errorf("[gnetx] client async write reply error: %v", err)
			}
		}
	})
	if err != nil {
		span.End()
		logx.Errorf("[gnetx] client async submit error: %v", err)
	}
}

func (c *Client) writeReply(sess *Session, reply any) error {
	payload, err := c.opts.Codec.Encode(reply, sess)
	if err != nil {
		return err
	}
	_, err = sess.conn.Write(payload)
	return err
}

func (c *Client) handleDecodeError(sess *Session, err error) gnet.Action {
	logx.Errorf("[gnetx] client decode error id=%s remote=%s: %v", sess.id, sess.RemoteAddr(), err)
	if errors.Is(err, ErrFrameTooLarge) {
		return gnet.Close
	}
	if c.opts.OnDecodeError == DecodeErrorClose {
		return gnet.Close
	}
	return gnet.None
}

func (c *Client) buildGnetOptions() []gnet.Option {
	opts := []gnet.Option{gnet.WithLogger(logxAdapter)}
	if c.opts.TCPKeepAlive > 0 {
		opts = append(opts, gnet.WithTCPKeepAlive(c.opts.TCPKeepAlive))
	}
	if c.opts.TCPKeepInterval > 0 {
		opts = append(opts, gnet.WithTCPKeepInterval(c.opts.TCPKeepInterval))
	}
	if c.opts.TCPKeepCount > 0 {
		opts = append(opts, gnet.WithTCPKeepCount(c.opts.TCPKeepCount))
	}
	if c.opts.ReadBufferCap > 0 {
		opts = append(opts, gnet.WithReadBufferCap(c.opts.ReadBufferCap))
	}
	if c.opts.WriteBufferCap > 0 {
		opts = append(opts, gnet.WithWriteBufferCap(c.opts.WriteBufferCap))
	}
	return opts
}
