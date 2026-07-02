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
)

// workerPool 复用 gnet 自带的 goroutine.DefaultWorkerPool（基于 ants），
// 供 AsyncHandler offload 使用。不自己创建 ants 池，避免重复初始化。
type workerPool = goroutine.Pool

// defaultWorkerPool 返回 gnet 全局 worker 池。
func defaultWorkerPool() *workerPool {
	return goroutine.DefaultWorkerPool
}

// Server 是 gnetx 的 TCP 服务端。内部实现 gnet.EventHandler，适配上层 Handler。
// 同时实现 go-zero service.Service 接口（Start/Stop），可加入 service.NewServiceGroup()
// 统一管理生命周期。
//
// 生命周期：
//   - OnBoot: 存 Engine，启动空闲扫描（若配置）
//   - OnOpen: 建 Session，SetContext，加入 SessionManager，listener.OnCreated
//   - OnTraffic: 解码 → Response 自动路由 → handler（sync on-loop / async offload）→ 回包
//   - OnClose: listener.OnDestroyed + SessionManager 移除 + Session.Close
//   - OnShutdown: 停扫描
//
// 用法 A（直接 Run，阻塞）：
//
//	srv, _ := gnetx.NewServer(...)
//	srv.Run()
//
// 用法 B（接入 go-zero service.Group）：
//
//	sg := service.NewServiceGroup()
//	sg.Add(srv)
//	sg.Start()  // 阻塞，proc 信号触发 Stop
type Server struct {
	gnet.BuiltinEventEngine

	opts    ServerOptions
	mgr     *SessionManager
	eng     gnet.Engine
	booted  atomic.Bool    // OnBoot 是否已执行（eng 是否可用）
	asyncWG sync.WaitGroup // 在途 async handler

	sweeper *idleSweeper
	pool    *workerPool // gnet 自带 ants worker pool，async handler offload 用

	metrics *stat.Metrics    // QPS/延迟统计（每分钟输出一次），构造时自动创建
	tracer  oteltrace.Tracer // OTel tracer，构造时缓存一次，避免每报文全局查找
}

// NewServer 用选项构造 Server。校验必填项并填充默认值。
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

	// 把强制的 MaxFrameLength 注入内置 codec（若其未显式设 WithMaxFrameSize），
	// 使该安全上限真正生效，防止损坏/错序流导致连接挂死。
	applyFrameLimit(o.Codec, o.MaxFrameLength)

	mgr := NewSessionManager(o.SessionListener)
	s := &Server{
		opts:    *o,
		mgr:     mgr,
		pool:    defaultWorkerPool(),
		metrics: stat.NewMetrics("gnetx-server-" + normalizeAddrForMetrics(o.Addr)),
		tracer:  gnetxTracer(),
	}
	return s, nil
}

// Manager 返回会话管理器，用于查找/广播/主动推送。
func (s *Server) Manager() *SessionManager {
	return s.mgr
}

// Start 启动并阻塞运行服务端，直到 Stop 或出错。
// 实现 go-zero service.Starter 接口，可加入 service.NewServiceGroup()。
func (s *Server) Start() {
	if err := s.Run(); err != nil {
		logx.Errorf("[gnetx] server run error: %v", err)
	}
}

// Run 启动并阻塞运行服务端，直到 Shutdown 或出错。返回 gnet.Run 的错误。
// 与 Start 的区别：Run 返回 error，Start 吞掉 error（适配 service.Group）。
func (s *Server) Run() error {
	addr := normalizeAddr(s.opts.Addr)
	return gnet.Run(s, addr, s.buildGnetOptions()...)
}

// Stop 停止服务端（优雅）。实现 go-zero service.Stopper 接口。
// 停止接受新连接 → 等在途 async handler 完成（3s 超时）→ 关闭所有连接。
func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		logx.Errorf("[gnetx] server shutdown error: %v", err)
	}
}

// Shutdown 优雅停止：停止接受新连接 → 等在途 async handler 完成（受 ctx 约束）→ 关闭。
// 与 Stop 的区别：Shutdown 接受 context 控制超时，返回 error。
// 若 server 尚未 boot（Run 未调用或未就绪），直接返回，避免对零值 Engine 调 Stop 触发 panic。
func (s *Server) Shutdown(ctx context.Context) error {
	if !s.booted.Load() {
		return nil
	}
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
		return nil
	case <-ctx.Done():
		logx.Infof("[gnetx] shutdown timeout, async handlers still running")
		return ctx.Err()
	}
}

// buildGnetOptions 把 ServerOptions 映射为 gnet.Option。注入 logx logger。
func (s *Server) buildGnetOptions() []gnet.Option {
	opts := []gnet.Option{gnet.WithLogger(logxAdapter)} // gnet 内部日志走 logx
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

// OnBoot 实现 gnet.EventHandler。存 Engine，启动空闲扫描。
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

// OnOpen 实现 gnet.EventHandler。创建 Session 并绑定到 conn.Context。
func (s *Server) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	sess := newSession(sessionIDForConn(c), c, s.opts.Codec, s.mgr, false)
	c.SetContext(sess)
	s.mgr.add(sess)
	logx.Infof("[gnetx] connected remote=%s id=%s", c.RemoteAddr(), sess.id)
	return nil, gnet.None
}

// OnTraffic 实现 gnet.EventHandler。解码并分发。
func (s *Server) OnTraffic(c gnet.Conn) gnet.Action {
	sess, _ := c.Context().(*Session)
	if sess == nil {
		return gnet.Close
	}
	sess.touch()

	batchLimit := s.opts.BatchReadLimit
	consumed := 0
	for i := 0; i < batchLimit; i++ {
		msg, err := s.opts.Codec.Decode(c, sess)
		if err != nil {
			if errors.Is(err, ErrIncompletePacket) {
				break // 半包，等下次可读事件，不要 Wake（避免空转）
			}
			return s.handleDecodeError(sess, err)
		}
		consumed++

		// opt-in 请求-响应：入站回包自动路由到在途请求
		if resp, ok := msg.(Response); ok {
			if sess.resolveResponse(resp.ResponseTID(), msg) {
				continue // 命中在途，完成，跳过 handler
			}
		}

		s.dispatch(sess, msg)
	}

	// 仅当本轮消费了帧且仍有剩余字节时，主动重触发 OnTraffic 处理后续帧；
	// 半包（consumed==0）不 Wake，等真正的可读事件，避免 event-loop 空转。
	if consumed > 0 && c.InboundBuffered() > 0 {
		_ = c.Wake(nil)
	}
	return gnet.None
}

// OnClose 实现 gnet.EventHandler。
func (s *Server) OnClose(c gnet.Conn, err error) gnet.Action {
	sess, _ := c.Context().(*Session)
	if sess == nil {
		return gnet.None
	}
	cause := "closed"
	if err != nil {
		cause = err.Error()
	}
	logSessionClosed(sess, cause)
	_ = sess.Close()
	return gnet.None
}

// OnShutdown 实现 gnet.EventHandler。停止空闲扫描（worker pool 由 gnet 全局管理，不在此释放）。
func (s *Server) OnShutdown(gnet.Engine) {
	if s.sweeper != nil {
		s.sweeper.stopSweep()
	}
}

// dispatch 把消息分发给 handler。sync handler on-loop 执行并回包；
// async handler offload 到 gnet worker pool，回包走 AsyncWrite。
func (s *Server) dispatch(sess *Session, msg any) {
	h := s.opts.Handler
	if isAsync(h) {
		s.dispatchAsync(sess, msg, h)
		return
	}
	s.dispatchSync(sess, msg, h)
}

// dispatchSync 同步执行 handler（on-loop，必须快），回包走 c.Write。
// 创建 OTel span 并记录 stat.Metrics（QPS/延迟百分位）。
func (s *Server) dispatchSync(sess *Session, msg any, h Handler) {
	startTime := timex.Now()
	ctx, span := startServerSpan(s.tracer, sess, msg)
	defer span.End()

	reply, hErr := h.Handle(ctx, sess, msg)

	duration := timex.Since(startTime)
	if duration > s.opts.SlowHandlerThreshold {
		logx.Slowf("[gnetx] slow handler %s id=%s", duration, sess.id)
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
		if err := s.writeReply(sess, reply); err != nil {
			logx.Errorf("[gnetx] write reply error: %v", err)
		}
	}
}

// dispatchAsync 异步执行 handler（offload 到 gnet worker pool），回包走 AsyncWrite。
// span 随闭包捕获，end 在 handler 结束后触发。metrics 记录提交耗时。
func (s *Server) dispatchAsync(sess *Session, msg any, h Handler) {
	startTime := timex.Now()
	ctx, span := startServerSpan(s.tracer, sess, msg)

	s.asyncWG.Add(1)
	err := s.pool.Submit(func() {
		defer s.asyncWG.Done()
		defer span.End()
		reply, hErr := h.Handle(ctx, sess, msg)
		if hErr != nil {
			span.RecordError(hErr)
			logx.Errorf("[gnetx] async handler error: %v", hErr)
			return
		}
		if reply != nil {
			if err := sess.Send(reply); err != nil {
				logx.Errorf("[gnetx] async write reply error: %v", err)
			}
		}
	})
	if err != nil {
		s.asyncWG.Done()
		span.End()
		logx.Errorf("[gnetx] async submit error: %v", err)
	}
	s.recordMetrics(timex.Since(startTime))
}

// recordMetrics 向 stat.Metrics 记录一条 Task（含 Duration）。metrics 在构造时必建，无需 nil 检查。
func (s *Server) recordMetrics(d time.Duration) {
	s.metrics.Add(stat.Task{Duration: d})
}

// normalizeAddrForMetrics 把 server addr 缩写成 metrics 名（去掉 scheme 和端口 wildcard）。
func normalizeAddrForMetrics(addr string) string {
	addr = normalizeAddr(addr)
	// 去掉 scheme 前缀 "tcp://"
	for _, scheme := range []string{"tcp://", "tcp4://", "tcp6://"} {
		if len(addr) > len(scheme) && addr[:len(scheme)] == scheme {
			addr = addr[len(scheme):]
			break
		}
	}
	return addr
}

// writeReply 同步回包（on-loop）。编码后 c.Write。
func (s *Server) writeReply(sess *Session, reply any) error {
	payload, err := s.opts.Codec.Encode(reply, sess)
	if err != nil {
		return err
	}
	_, err = sess.conn.Write(payload)
	return err
}

// handleDecodeError 处理不可恢复解码错误，按配置策略决定是否关闭连接。
func (s *Server) handleDecodeError(sess *Session, err error) gnet.Action {
	logx.Errorf("[gnetx] decode error id=%s remote=%s: %v", sess.id, sess.RemoteAddr(), err)
	if errors.Is(err, ErrFrameTooLarge) {
		return gnet.Close
	}
	if s.opts.OnDecodeError == DecodeErrorClose {
		return gnet.Close
	}
	return gnet.None
}

// isAsync 判断 handler 是否标记为异步。
func isAsync(h Handler) bool {
	ah, ok := h.(AsyncHandler)
	return ok && ah.IsAsync()
}

// normalizeAddr 规范化地址，无 scheme 时补 tcp://。
func normalizeAddr(addr string) string {
	if addr == "" {
		return ""
	}
	if containsScheme(addr) {
		return addr
	}
	return "tcp://" + addr
}

// containsScheme 粗略判断地址是否带 scheme。
func containsScheme(addr string) bool {
	for _, scheme := range []string{"tcp://", "tcp4://", "tcp6://", "unix://", "udp://"} {
		if len(addr) >= len(scheme) && addr[:len(scheme)] == scheme {
			return true
		}
	}
	return false
}

// sessionIDForConn 用远端地址派生会话 id。
func sessionIDForConn(c gnet.Conn) string {
	return c.RemoteAddr().String()
}
