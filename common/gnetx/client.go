package gnetx

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/gnet/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/proc"
	"github.com/zeromicro/go-zero/core/timex"
	oteltrace "go.opentelemetry.io/otel/trace"

	"zero-service/common/antsx"
)

// ClientConn extends Conn with the Request method.
type ClientConn interface {
	Conn
	Request(ctx context.Context, msg Correlatable, ttl time.Duration) (any, error)
}

// Client 是 gnetx 的单连接 TCP 客户端，构造即拨号，断线自动重连。
type Client struct {
	gnet.BuiltinEventEngine

	opts ClientOptions
	gcli *gnet.Client
	pool *workerPool

	address string

	replyPool *antsx.ReplyPool[any]

	sess         atomic.Pointer[session]
	closed       atomic.Bool
	reconnecting atomic.Bool
	reconnectCh  chan struct{}

	tracer  oteltrace.Tracer
	asyncWG sync.WaitGroup
}

func MustNewClient(address string, opts ...ClientOption) *Client {
	cli, err := NewClient(address, opts...)
	if err != nil {
		panic("gnetx: MustNewClient " + address + ": " + err.Error())
	}
	proc.AddWrapUpListener(func() { cli.Close() })
	return cli
}

func NewClient(address string, opts ...ClientOption) (*Client, error) {
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

	applyFrameLimit(o.Codec, o.MaxFrameLength)

	replyPool := antsx.NewReplyPool[any](
		antsx.WithName("gnetx-client-"+address),
		antsx.WithDefaultTTL(30*time.Second),
	)
	c := &Client{
		opts:        *o,
		pool:        defaultWorkerPool(),
		address:     address,
		replyPool:   replyPool,
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
		logx.Errorf("[gnetx] 首次拨号 %s 失败: %v，启动后台重连", address, err)
		c.startReconnect()
		return c, nil
	}
	return c, nil
}

func (c *Client) Session() ClientConn {
	cn := c.sess.Load()
	if cn == nil {
		return nil
	}
	return cn
}

func (c *Client) WriteAsync(ctx context.Context, msg any) error {
	cn := c.sess.Load()
	if cn == nil {
		return ErrSessionClosed
	}
	return cn.WriteAsync(ctx, msg)
}

func (c *Client) Request(ctx context.Context, msg Correlatable, ttl time.Duration) (any, error) {
	cn := c.sess.Load()
	if cn == nil {
		return nil, ErrSessionClosed
	}
	return cn.Request(ctx, msg, ttl)
}

func (c *Client) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), c.opts.ShutdownTimeout)
	defer cancel()
	c.Shutdown(ctx)
}

// Shutdown 优雅关闭客户端：先关闭当前连接，等待异步 handler 完成，再停止 gnet 引擎。
func (c *Client) Shutdown(ctx context.Context) {
	if !c.closed.CompareAndSwap(false, true) {
		return
	}
	close(c.reconnectCh)
	if cn := c.sess.Swap(nil); cn != nil {
		_ = cn.Close()
	}
	done := make(chan struct{})
	go func() {
		c.asyncWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		logx.Infof("[gnetx] client close timeout, async handlers still running")
	}
	c.replyPool.Close()
	if c.gcli != nil {
		_ = c.gcli.Stop()
	}
	logx.Infof("[gnetx] client closed %s/%s", "tcp", c.address)
}

func (c *Client) dial() error {
	_, err := c.gcli.DialContext("tcp", c.address, nil)
	return err
}

func (c *Client) OnTick() (delay time.Duration, action gnet.Action) {
	if c.opts.HeartbeatInterval <= 0 || c.opts.HeartbeatMessage == nil {
		return
	}
	cn := c.sess.Load()
	if cn == nil || cn.isClosed() {
		return c.opts.HeartbeatInterval, gnet.None
	}
	msg := c.opts.HeartbeatMessage()
	payload, err := c.opts.Codec.Encode(context.Background(), msg, cn)
	if err != nil {
		logx.Errorf("[gnetx] client heartbeat encode error: %v", err)
		return c.opts.HeartbeatInterval, gnet.None
	}
	if err := cn.gc.AsyncWrite(payload, nil); err != nil {
		logx.Errorf("[gnetx] client heartbeat send error: %v", err)
	}
	return c.opts.HeartbeatInterval, gnet.None
}

func (c *Client) OnOpen(gc gnet.Conn) ([]byte, gnet.Action) {
	cn := newSession(newSessionID(), gc, c.opts.Codec, nil, c.replyPool, c.opts.SequenceStart)
	gc.SetContext(cn)
	c.sess.Store(cn)
	if c.opts.OnConnect != nil {
		ctx := context.Background()
		if c.opts.ConnectTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, c.opts.ConnectTimeout)
			go func() {
				defer cancel()
				c.opts.OnConnect(ctx, c)
			}()
		} else {
			go c.opts.OnConnect(ctx, c)
		}
	}
	return nil, gnet.None
}

func (c *Client) OnTraffic(gc gnet.Conn) gnet.Action {
	cn, _ := gc.Context().(*session)
	if cn == nil {
		logx.Errorf("[gnetx] client OnTraffic: session context is nil, closing connection remote=%s", gc.RemoteAddr())
		return gnet.Close
	}
	cn.touch()

	batchLimit := c.opts.BatchReadLimit
	consumed := 0
	for i := 0; i < batchLimit; i++ {
		msg, err := c.opts.Codec.Decode(gc, cn)
		if err != nil {
			if errors.Is(err, ErrIncompletePacket) {
				break
			}
			return c.handleDecodeError(cn, err)
		}
		consumed++
		if resp, ok := msg.(Response); ok {
			if cn.resolveResponse(resp.ResponseTID(), msg) {
				continue
			}
			if resp.ResponseTID() != "" {
				continue // 未匹配的应答消息静默丢弃，避免僵尸应答回环
			}
		}
		c.dispatch(context.Background(), cn, msg)
	}
	if consumed > 0 && gc.InboundBuffered() > 0 {
		_ = gc.Wake(nil)
	}
	return gnet.None
}

func (c *Client) OnClose(gc gnet.Conn, err error) gnet.Action {
	cn, _ := gc.Context().(*session)
	if cn == nil {
		return gnet.None
	}
	cause := "closed"
	if err != nil {
		cause = err.Error()
	}
	logx.Errorf("[gnetx] session closed id=%s alias=%s remote=%s cause=%s",
		cn.id, cn.alias, cn.RemoteAddr(), cause)
	c.sess.CompareAndSwap(cn, nil)
	_ = cn.Close()
	if !c.closed.Load() {
		c.startReconnect()
	}
	return gnet.None
}

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
			logx.Infof("[gnetx] reconnecting %s/%s ...", "tcp", c.address)
			if err := c.dial(); err != nil {
				logx.Errorf("[gnetx] reconnect %s/%s failed: %v", "tcp", c.address, err)
				continue
			}
			// Check closed again after successful dial — Close might have been called
			// while dial was in progress. If so, close the newly created session.
			if c.closed.Load() {
				if cn := c.sess.Swap(nil); cn != nil {
					_ = cn.Close()
				}
				return
			}
			logx.Infof("[gnetx] reconnected %s/%s", "tcp", c.address)
			return
		}
	}()
}

func (c *Client) dispatch(ctx context.Context, cn *session, msg any) {
	h := c.opts.Handler
	if resolver, ok := h.(RouteResolver); ok {
		resolved, err := resolver.Resolve(msg)
		if err != nil {
			logx.Errorf("[gnetx] client route resolve error: %v", err)
			return
		}
		h = resolved
	}
	if isAsync(h) {
		c.dispatchAsync(ctx, cn, msg, h)
		return
	}
	c.dispatchSync(ctx, cn, msg, h)
}

func (c *Client) dispatchSync(parentCtx context.Context, cn *session, msg any, h Handler) {
	startTime := timex.Now()
	ctx, span := startClientSpan(c.tracer, parentCtx, cn, msg)
	defer span.End()

	if pcp, ok := msg.(PacketContextProvider); ok {
		ctx = context.WithValue(ctx, PacketContextKey, pcp.PacketContext())
	}
	ctx = injectSessionLogFields(ctx, cn)

	reply, hErr := h.Handle(ctx, cn, msg)

	duration := timex.Since(startTime)
	if duration > c.opts.SlowHandlerThreshold {
		logx.WithContext(ctx).WithDuration(duration).Slowf("[gnetx] client slow handler id=%s", cn.id)
	}
	if hErr != nil {
		span.RecordError(hErr)
		logx.WithContext(ctx).Errorf("[gnetx] client handler error: %v", hErr)
		return
	}
	if reply != nil {
		if err := c.writeReply(ctx, cn, reply); err != nil {
			logx.WithContext(ctx).Errorf("[gnetx] client write reply error: %v", err)
		}
	}
}

func (c *Client) dispatchAsync(parentCtx context.Context, cn *session, msg any, h Handler) {
	ctx, span := startClientSpan(c.tracer, parentCtx, cn, msg)

	if pcp, ok := msg.(PacketContextProvider); ok {
		ctx = context.WithValue(ctx, PacketContextKey, pcp.PacketContext())
	}
	ctx = injectSessionLogFields(ctx, cn)

	c.asyncWG.Add(1)
	err := c.pool.Submit(func() {
		defer c.asyncWG.Done()
		defer span.End()
		startTime := timex.Now()
		reply, hErr := h.Handle(ctx, cn, msg)
		duration := timex.Since(startTime)
		if duration > c.opts.SlowHandlerThreshold {
			logx.WithContext(ctx).WithDuration(duration).Slowf("[gnetx] client async slow handler id=%s", cn.id)
		}
		if hErr != nil {
			span.RecordError(hErr)
			logx.WithContext(ctx).Errorf("[gnetx] client async handler error: %v", hErr)
			return
		}
		if reply != nil {
			if err := cn.WriteAsync(ctx, reply); err != nil {
				logx.WithContext(ctx).Errorf("[gnetx] client async write reply error: %v", err)
			}
		}
	})
	if err != nil {
		c.asyncWG.Done()
		span.End()
		logx.Errorf("[gnetx] client async submit error: %v", err)
	}
}

func (c *Client) writeReply(ctx context.Context, cn *session, reply any) error {
	return cn.Write(ctx, reply)
}

func (c *Client) handleDecodeError(cn *session, err error) gnet.Action {
	logx.Errorf("[gnetx] client decode error id=%s remote=%s: %v", cn.id, cn.RemoteAddr(), err)
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
	if c.opts.HeartbeatInterval > 0 && c.opts.HeartbeatMessage != nil {
		opts = append(opts, gnet.WithTicker(true))
	}
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
