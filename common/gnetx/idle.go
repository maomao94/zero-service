package gnetx

import "time"

// idleSweeper 周期扫描所有会话，关闭超过 IdleTimeout 未活跃的连接。
// 用独立 goroutine 而非 gnet OnTick，规避 OnTick 多核 N× 触发且无 per-loop 连接枚举 API 的问题。
// Close 跨 goroutine 安全（gnet.Conn.Close 安全）。
type idleSweeper struct {
	mgr    *SessionManager
	period time.Duration
	stop   chan struct{}
}

// newIdleSweeper 创建空闲扫描器。period 自动取 IdleTimeout/2（下限 1s）。
func newIdleSweeper(mgr *SessionManager, idle time.Duration) *idleSweeper {
	period := idle / 2
	if period < time.Second {
		period = time.Second
	}
	return &idleSweeper{
		mgr:    mgr,
		period: period,
		stop:   make(chan struct{}),
	}
}

// start 启动扫描 goroutine。
func (s *idleSweeper) start(idle time.Duration) {
	go s.loop(idle)
}

// loop 扫描主循环。
func (s *idleSweeper) loop(idle time.Duration) {
	ticker := time.NewTicker(s.period)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.sweep(idle)
		}
	}
}

// sweep 执行一次扫描，关闭超时会话。
func (s *idleSweeper) sweep(idle time.Duration) {
	now := time.Now()
	for _, sess := range s.mgr.All() {
		if now.Sub(sess.LastActiveAt()) > idle {
			_ = sess.Close()
		}
	}
}

// stop 停止扫描。
func (s *idleSweeper) stopSweep() {
	close(s.stop)
}
