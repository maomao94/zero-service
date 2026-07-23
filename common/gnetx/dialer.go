package gnetx

import (
	"context"
	"errors"
	"sync"

	"github.com/panjf2000/gnet/v2"

	"zero-service/common/antsx"
)

// Dialer 是短连接 TCP 客户端。内部共享一个 gnet.Client 事件引擎，
// 每次 Dial/Request 通过 gcli.DialContext 创建新连接。
// Request 用 antsx.Promise 做单次请求-响应匹配，ctx 控制超时。
// 不持有 ReplyPool，不维护 pending 表。
//
// 用法 A — Request 一步完成：
//
//	dialer := gnetx.NewDialer(gnetx.WithClientCodec(myCodec))
//	resp, err := dialer.Request(ctx, "tcp", "127.0.0.1:8080", req)
//
// 用法 B — Dial 拿到 Conn 手动控制：
//
//	conn, err := dialer.Dial(ctx, "tcp", "127.0.0.1:8080")
//	defer conn.Close()
//	conn.WriteAsync(ctx, msg)
type Dialer struct {
	gnet.BuiltinEventEngine

	opts   ClientOptions
	mu     sync.Mutex
	closed bool

	gcli     *gnet.Client
	gcliOnce sync.Once
	gcliErr  error
}

// --- per-connection dial state (stored in session attributes) ---

type dialStateKeyType struct{}

var dialStateKey dialStateKeyType

type dialState struct {
	promise *antsx.Promise[any]
	tid     string
}

func NewDialer(opts ...ClientOption) *Dialer {
	o := &ClientOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
	o.applyDefaults()
	if o.Codec == nil {
		panic("gnetx: NewDialer: Codec is required")
	}
	applyFrameLimit(o.Codec, o.MaxFrameLength)
	return &Dialer{opts: *o}
}

func (d *Dialer) client() (*gnet.Client, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil, ErrSessionClosed
	}
	d.gcliOnce.Do(func() {
		d.gcli, d.gcliErr = gnet.NewClient(d, buildGnetClientOptions(d.opts)...)
		if d.gcliErr != nil {
			return
		}
		d.gcliErr = d.gcli.Start()
	})
	return d.gcli, d.gcliErr
}

// Dial 创建一次性 TCP 连接并返回 Conn。返回的 Conn replyPool 为 nil，仅支持 WriteAsync。
// 调用方使用完毕后应调用 conn.Close() 释放底层连接。
func (d *Dialer) Dial(ctx context.Context, network, address string) (Conn, error) {
	gcli, err := d.client()
	if err != nil {
		return nil, err
	}
	gc, err := gcli.DialContext(network, address, ctx)
	if err != nil {
		return nil, err
	}
	return gc.Context().(*session), nil
}

// Request 一步完成拨号+发送请求+等待回包+关闭连接。
// msg 需实现 Correlatable，回包需实现 Response 且 ResponseTID 与 msg.TID 一致。
func (d *Dialer) Request(ctx context.Context, network, address string, msg Correlatable) (any, error) {
	gcli, err := d.client()
	if err != nil {
		return nil, err
	}

	promise := antsx.NewPromise[any]()
	ds := &dialState{promise: promise, tid: msg.TID()}

	gc, err := gcli.DialContext(network, address, ctx)
	if err != nil {
		return nil, err
	}
	cn := gc.Context().(*session)
	cn.SetAttribute(&dialStateKey, ds)

	payload, err := d.opts.Codec.Encode(ctx, msg, cn)
	if err != nil {
		cn.Close()
		return nil, err
	}
	if err := gc.AsyncWrite(payload, nil); err != nil {
		cn.Close()
		return nil, err
	}

	result, err := promise.Await(ctx)
	cn.Close()
	return result, err
}

func (d *Dialer) Close() error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	d.closed = true
	gcli := d.gcli
	d.mu.Unlock()
	if gcli != nil {
		return gcli.Stop()
	}
	return nil
}

// --- gnet.EventHandler ---

func (d *Dialer) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	cn := newSession(newSessionID(), c, d.opts.Codec, nil, nil, d.opts.SequenceStart)
	c.SetContext(cn)
	return nil, gnet.None
}

func (d *Dialer) OnTraffic(c gnet.Conn) gnet.Action {
	cn, _ := c.Context().(*session)
	if cn == nil {
		return gnet.Close
	}
	ds, _ := cn.Attribute(&dialStateKey).(*dialState)
	if ds == nil {
		return gnet.None
	}

	for i := 0; i < 64; i++ {
		msg, err := d.opts.Codec.Decode(c, cn)
		if err != nil {
			if errors.Is(err, ErrIncompletePacket) {
				break
			}
			ds.promise.Reject(err)
			return gnet.Close
		}
		if resp, ok := msg.(Response); ok && resp.ResponseTID() == ds.tid {
			ds.promise.Resolve(resp)
			return gnet.Close
		}
	}
	return gnet.None
}

func (d *Dialer) OnClose(c gnet.Conn, err error) gnet.Action {
	cn, _ := c.Context().(*session)
	if cn != nil {
		cn.closeFromEventLoop()
		if ds, ok := cn.Attribute(&dialStateKey).(*dialState); ok {
			ds.promise.Reject(ErrSessionClosed)
		}
	}
	return gnet.None
}

func buildGnetClientOptions(o ClientOptions) []gnet.Option {
	opts := []gnet.Option{gnet.WithLogger(logxAdapter)}
	if o.TCPKeepAlive > 0 {
		opts = append(opts, gnet.WithTCPKeepAlive(o.TCPKeepAlive))
	}
	if o.TCPKeepInterval > 0 {
		opts = append(opts, gnet.WithTCPKeepInterval(o.TCPKeepInterval))
	}
	if o.TCPKeepCount > 0 {
		opts = append(opts, gnet.WithTCPKeepCount(o.TCPKeepCount))
	}
	if o.ReadBufferCap > 0 {
		opts = append(opts, gnet.WithReadBufferCap(o.ReadBufferCap))
	}
	if o.WriteBufferCap > 0 {
		opts = append(opts, gnet.WithWriteBufferCap(o.WriteBufferCap))
	}
	return opts
}
