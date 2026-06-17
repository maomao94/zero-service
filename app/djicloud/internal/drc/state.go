package drc

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// DeviceSession 单个设备的 DRC 会话，包含运行时状态 + 心跳 goroutine 生命周期。
//
// 并发安全：
//   - Enabled、StartedAt、SessionID、MaxDeadline 等状态机字段的读写必须在持有 mu 的情况下进行
//   - seq 和 lastHeartbeat 使用 atomic 独立保护，可在不持有 mu 的情况下安全读写
//   - GatewaySn 在创建后不可变，无需加锁
//   - heartbeatCancel 在 Enable 时设置，Disable/expire 时调用并置 nil
type DeviceSession struct {
	// GatewaySn 设备序列号，创建后不可变。
	GatewaySn string
	// Enabled DRC 模式是否已启用。读写需持有 mu。
	Enabled bool
	// StartedAt DRC 模式启用时间。读写需持有 mu。
	StartedAt time.Time
	// SessionID 单次 DRC 会话标识，用于避免旧 goroutine 清理新会话。读写需持有 mu。
	SessionID string
	// MaxDeadline DRC 会话最大截止时间。超过此时间后强制清除缓存并停止心跳，
	// 零值表示无上限。该时间基于 Enable 时传入的 WithMaxTimeout 计算，
	// 不受设备心跳续期影响。读写需持有 mu。
	MaxDeadline time.Time
	// seq 内部序号，用于 DRC 杆量和心跳（递增）。
	seq atomic.Int64
	// lastHeartbeat 最近一次收到设备心跳上行的 UnixMilli 时间戳。
	// 使用 atomic 读写，可在不持有 mu 的情况下安全更新和查询。
	lastHeartbeat atomic.Int64
	// heartbeatCancel 取消心跳 goroutine 的函数。
	// Enable 时设置，Disable/expire 时调用并置 nil。读写需持有 mu。
	heartbeatCancel context.CancelFunc
	// mu 保护 DeviceSession 状态机字段的并发读写。
	mu sync.Mutex
}

// UpdateHeartbeat 原子更新心跳时间戳为当前时间（毫秒级）。
// 线程安全，无需持有 mu。
func (s *DeviceSession) UpdateHeartbeat() {
	s.lastHeartbeat.Store(time.Now().UnixMilli())
}

// GetLastHeartbeat 获取最近一次心跳时间。
// 线程安全，无需持有 mu。
func (s *DeviceSession) GetLastHeartbeat() time.Time {
	ms := s.lastHeartbeat.Load()
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms)
}

// IsAlive 判断当前状态是否仍然存活。
// 线程安全，无需持有 mu。
//
// 判定条件：
//  1. Enabled 为 true
//  2. 未超过 MaxDeadline（如果设置了）
//  3. 心跳未超时：time.Since(lastHeartbeat) < heartbeatTimeout
func (s *DeviceSession) IsAlive(heartbeatTimeout time.Duration) bool {
	if !s.Enabled {
		return false
	}

	now := time.Now()

	// 检查 MaxDeadline
	deadline := s.MaxDeadline
	if !deadline.IsZero() && now.After(deadline) {
		return false
	}

	// 检查心跳超时
	ms := s.lastHeartbeat.Load()
	if ms == 0 {
		return false
	}
	lastTime := time.UnixMilli(ms)
	return now.Sub(lastTime) < heartbeatTimeout
}

// isCurrentSessionAlive 判断指定 sessionID 对应的当前会话是否仍然存活。
// 调用方必须已持有 s.mu。
func (s *DeviceSession) isCurrentSessionAlive(sessionID string, heartbeatTimeout time.Duration) bool {
	return s.SessionID == sessionID && s.IsAlive(heartbeatTimeout)
}

// String 返回 DeviceSession 的可读字符串表示，用于日志和调试。
func (s *DeviceSession) String() string {
	enabled := s.Enabled
	sessionID := s.SessionID
	startedAt := s.StartedAt
	maxDeadline := s.MaxDeadline
	hasWorker := s.heartbeatCancel != nil
	lastHb := s.GetLastHeartbeat()
	seq := s.seq.Load()
	return fmt.Sprintf("DeviceSession{GatewaySn=%s Enabled=%t SessionID=%s StartedAt=%v MaxDeadline=%v LastHeartbeat=%v Seq=%d HasWorker=%t}",
		s.GatewaySn, enabled, sessionID, startedAt, maxDeadline, lastHb, seq, hasWorker)
}
