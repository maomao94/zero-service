package gnetx

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/gnet/v2"
	"github.com/panjf2000/gnet/v2/pkg/pool/goroutine"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/core/timex"
	oteltrace "go.opentelemetry.io/otel/trace"

	"zero-service/common/antsx"
)

// ServerConn extends Conn with server-specific methods.
type ServerConn interface {
	Conn
	Alias() string
	Register(alias string)
	Request(ctx context.Context, msg Correlatable, ttl time.Duration) (any, error)
}

// SessionListener 监听会话生命周期事件。
type SessionListener interface {
	OnCreated(s ServerConn)
	OnRegistered(s ServerConn)
	OnDestroyed(s ServerConn)
}

type noopSessionListener struct{}

func (noopSessionListener) OnCreated(ServerConn)    {}
func (noopSessionListener) OnRegistered(ServerConn) {}
func (noopSessionListener) OnDestroyed(ServerConn)  {}

type workerPool = goroutine.Pool

func defaultWorkerPool() *workerPool { return goroutine.DefaultWorkerPool }

// Server 是 gnetx 的 TCP 服务端，实现 gnet.EventHandler 和 go-zero service.Service。
type Server struct {
	gnet.BuiltinEventEngine

	opts    ServerOptions
	mgr     *SessionManager
	eng     gnet.Engine
	booted  atomic.Bool
	asyncWG sync.WaitGroup

	sweeper   *idleSweeper
	pool      *workerPool
	replyPool *antsx.ReplyPool[any]

	metrics *stat.Metrics
	tracer  oteltrace.Tracer
}

func NewServer(opts ...ServerOption) (*Server, error) {
	o := &ServerOptions{}
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

	mgr := NewSessionManager(o.SessionListener)
	replyPool := antsx.NewReplyPool[any](
		antsx.WithName("gnetx-server-"+normalizeAddrForMetrics(o.Addr)),
		antsx.WithDefaultTTL(30*time.Second),
	)
	return &Server{
		opts:      *o,
		mgr:       mgr,
		pool:      defaultWorkerPool(),
		replyPool: replyPool,
		metrics:   stat.NewMetrics("gnetx-server-" + normalizeAddrForMetrics(o.Addr)),
		tracer:    gnetxTracer(),
	}, nil
}

func (s *Server) Manager() *SessionManager { return s.mgr }

func (s *Server) Start() {
	if err := s.Run(); err != nil {
		logx.Errorf("[gnetx] server run error: %v", err)
	}
}

func (s *Server) Run() error {
	return gnet.Run(s, normalizeAddr(s.opts.Addr), s.buildGnetOptions()...)
}

func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		logx.Errorf("[gnetx] server shutdown error: %v", err)
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	if !s.booted.Load() {
		return nil
	}
	defer s.replyPool.Close()
	if err := s.eng.Stop(ctx); err != nil {
		return err
	}
	done := make(chan struct{})
	go func() {
		s.asyncWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		logx.Infof("[gnetx] shutdown timeout, async handlers still running")
	}
	return nil
}

func (s *Server) buildGnetOptions() []gnet.Option {
	opts := []gnet.Option{gnet.WithLogger(logxAdapter)}
	if s.opts.Multicore {
		opts = append(opts, gnet.WithMulticore(true))
	}
	if s.opts.NumEventLoop > 0 {
		opts = append(opts, gnet.WithNumEventLoop(s.opts.NumEventLoop))
	}
	if s.opts.LoadBalancing != 0 {
		opts = append(opts, gnet.WithLoadBalancing(s.opts.LoadBalancing))
	}
	if s.opts.TCPKeepAlive > 0 {
		opts = append(opts, gnet.WithTCPKeepAlive(s.opts.TCPKeepAlive))
	}
	if s.opts.TCPKeepInterval > 0 {
		opts = append(opts, gnet.WithTCPKeepInterval(s.opts.TCPKeepInterval))
	}
	if s.opts.TCPKeepCount > 0 {
		opts = append(opts, gnet.WithTCPKeepCount(s.opts.TCPKeepCount))
	}
	if s.opts.ReadBufferCap > 0 {
		opts = append(opts, gnet.WithReadBufferCap(s.opts.ReadBufferCap))
	}
	if s.opts.WriteBufferCap > 0 {
		opts = append(opts, gnet.WithWriteBufferCap(s.opts.WriteBufferCap))
	}
	return opts
}

func (s *Server) OnBoot(eng gnet.Engine) gnet.Action {
	s.eng = eng
	s.booted.Store(true)
	if s.opts.IdleTimeout > 0 {
		s.sweeper = newIdleSweeper(s.mgr, s.opts.IdleTimeout)
		s.sweeper.start(s.opts.IdleTimeout)
	}
	logx.Infof("[gnetx] server booting on %s", s.opts.Addr)
	return gnet.None
}

func (s *Server) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	cn := newSession(newSessionID(), c, s.opts.Codec, s.mgr, s.replyPool, s.opts.SequenceStart)
	c.SetContext(cn)
	s.mgr.add(cn)
	logx.Infof("[gnetx] connected remote=%s id=%s", c.RemoteAddr(), cn.id)
	return nil, gnet.None
}

func (s *Server) OnTraffic(c gnet.Conn) gnet.Action {
	cn, _ := c.Context().(*session)
	if cn == nil {
		logx.Errorf("[gnetx] OnTraffic: session context is nil, closing connection remote=%s", c.RemoteAddr())
		return gnet.Close
	}
	cn.touch()

	batchLimit := s.opts.BatchReadLimit
	consumed := 0
	for i := 0; i < batchLimit; i++ {
		msg, err := s.opts.Codec.Decode(c, cn)
		if err != nil {
			if errors.Is(err, ErrIncompletePacket) {
				break
			}
			return s.handleDecodeError(cn, err)
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
		s.dispatch(context.Background(), cn, msg)
	}

	if consumed > 0 && c.InboundBuffered() > 0 {
		_ = c.Wake(nil)
	}
	return gnet.None
}

func (s *Server) OnClose(c gnet.Conn, err error) gnet.Action {
	cn, _ := c.Context().(*session)
	if cn == nil {
		return gnet.None
	}
	cause := "closed"
	if err != nil {
		cause = err.Error()
	}
	logx.Errorf("[gnetx] session closed id=%s alias=%s remote=%s cause=%s",
		cn.id, cn.alias, cn.RemoteAddr(), cause)
	_ = cn.Close()
	return gnet.None
}

func (s *Server) OnShutdown(gnet.Engine) {
	if s.sweeper != nil {
		s.sweeper.stopSweep()
	}
}

func (s *Server) dispatch(ctx context.Context, cn *session, msg any) {
	h := s.opts.Handler
	if resolver, ok := h.(RouteResolver); ok {
		resolved, err := resolver.Resolve(msg)
		if err != nil {
			logx.Errorf("[gnetx] route resolve error: %v", err)
			return
		}
		h = resolved
	}
	if isAsync(h) {
		s.dispatchAsync(ctx, cn, msg, h)
		return
	}
	s.dispatchSync(ctx, cn, msg, h)
}

func (s *Server) dispatchSync(parentCtx context.Context, cn *session, msg any, h Handler) {
	startTime := timex.Now()
	ctx, span := startServerSpan(s.tracer, parentCtx, cn, msg)
	defer span.End()

	if pcp, ok := msg.(PacketContextProvider); ok {
		ctx = context.WithValue(ctx, PacketContextKey, pcp.PacketContext())
	}
	ctx = injectSessionLogFields(ctx, cn)

	reply, hErr := h.Handle(ctx, cn, msg)

	duration := timex.Since(startTime)
	if duration > s.opts.SlowHandlerThreshold {
		logx.Slowf("[gnetx] slow handler %s id=%s", duration, cn.id)
	}
	if hErr != nil {
		span.RecordError(hErr)
		logx.Errorf("[gnetx] handler error: %v", hErr)
	}
	s.recordMetrics(duration)
	if hErr != nil {
		return
	}
	if reply != nil {
		if err := s.writeReply(ctx, cn, reply); err != nil {
			logx.Errorf("[gnetx] write reply error: %v", err)
		}
	}
}

func (s *Server) dispatchAsync(parentCtx context.Context, cn *session, msg any, h Handler) {
	ctx, span := startServerSpan(s.tracer, parentCtx, cn, msg)

	if pcp, ok := msg.(PacketContextProvider); ok {
		ctx = context.WithValue(ctx, PacketContextKey, pcp.PacketContext())
	}
	ctx = injectSessionLogFields(ctx, cn)

	s.asyncWG.Add(1)
	err := s.pool.Submit(func() {
		defer s.asyncWG.Done()
		defer span.End()
		startTime := timex.Now()
		reply, hErr := h.Handle(ctx, cn, msg)
		duration := timex.Since(startTime)
		if duration > s.opts.SlowHandlerThreshold {
			logx.Slowf("[gnetx] async slow handler %s id=%s", duration, cn.id)
		}
		s.recordMetrics(duration)
		if hErr != nil {
			span.RecordError(hErr)
			logx.Errorf("[gnetx] async handler error: %v", hErr)
			return
		}
		if reply != nil {
			if err := cn.WriteAsync(ctx, reply); err != nil {
				logx.Errorf("[gnetx] async write reply error: %v", err)
			}
		}
	})
	if err != nil {
		s.asyncWG.Done()
		span.End()
		logx.Errorf("[gnetx] async submit error: %v", err)
	}
}

func (s *Server) recordMetrics(d time.Duration) { s.metrics.Add(stat.Task{Duration: d}) }

func (s *Server) writeReply(ctx context.Context, cn *session, reply any) error {
	return cn.Write(ctx, reply)
}

func (s *Server) handleDecodeError(cn *session, err error) gnet.Action {
	logx.Errorf("[gnetx] decode error id=%s remote=%s: %v", cn.id, cn.RemoteAddr(), err)
	if errors.Is(err, ErrFrameTooLarge) {
		return gnet.Close
	}
	if s.opts.OnDecodeError == DecodeErrorClose {
		return gnet.Close
	}
	return gnet.None
}

func normalizeAddrForMetrics(addr string) string {
	addr = normalizeAddr(addr)
	for _, scheme := range []string{"tcp://", "tcp4://", "tcp6://"} {
		if len(addr) > len(scheme) && addr[:len(scheme)] == scheme {
			addr = addr[len(scheme):]
			break
		}
	}
	return addr
}

func normalizeAddr(addr string) string {
	if addr == "" {
		return ""
	}
	if containsScheme(addr) {
		return addr
	}
	return "tcp://" + addr
}

func containsScheme(addr string) bool {
	for _, scheme := range []string{"tcp://", "tcp4://", "tcp6://", "unix://", "udp://"} {
		if len(addr) >= len(scheme) && addr[:len(scheme)] == scheme {
			return true
		}
	}
	return false
}
