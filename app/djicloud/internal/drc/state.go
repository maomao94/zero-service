package drc

import (
	"sync"
	"time"
)

// State 单个设备的 DRC 运行时状态。
//
// 并发安全：所有可变字段（Enabled、StartedAt、LastDeviceHeartbeat、SessionID、MaxDeadline、seq）
// 的读写必须在持有 mu 的情况下进行。GatewaySn 在创建后不可变，无需加锁。
type State struct {
	// GatewaySn 设备序列号，创建后不可变。
	GatewaySn string
	// Enabled DRC 模式是否已启用。读写需持有 mu。
	Enabled bool
	// StartedAt DRC 模式启用时间。读写需持有 mu。
	StartedAt time.Time
	// LastDeviceHeartbeat 最近一次收到设备心跳上行的时间（由 OnDeviceHeartbeat 刷新）。读写需持有 mu。
	LastDeviceHeartbeat time.Time
	// SessionID 单次 DRC 会话标识，用于避免旧 goroutine 清理新会话。读写需持有 mu。
	SessionID string
	// MaxDeadline DRC 会话最大截止时间。超过此时间后强制清除缓存并停止心跳，
	// 零值表示无上限。该时间基于 Enable 时传入的 WithMaxTimeout 计算，
	// 不受设备心跳续期影响。读写需持有 mu。
	MaxDeadline time.Time
	// seq 内部序号，用于 DRC 杆量和心跳（递增）。读写需持有 mu。
	seq int
	// mu 保护 State 所有可变字段的并发读写。
	// 调用方在访问或修改上述任何字段前必须先 Lock，操作完成后 Unlock。
	mu sync.Mutex
}

// IsExpired 判断 DRC 会话是否已过期。
// 调用方必须已持有 s.mu。
//
// 两层过期判定：
//  1. MaxDeadline（Enable 时通过 WithMaxTimeout 设置的绝对截止时间，不可续期）
//  2. 设备心跳超时（LastDeviceHeartbeat + heartbeatTimeout），作为 cache TTL 的精确补充
//
// cache TTL 由 TimingWheel 驱动（秒级精度），此方法提供亚秒级的准确判断。
func (s *State) IsExpired(now time.Time, heartbeatTimeout time.Duration) bool {
	if !s.MaxDeadline.IsZero() && !now.Before(s.MaxDeadline) {
		return true
	}
	if s.LastDeviceHeartbeat.IsZero() || heartbeatTimeout <= 0 {
		return false
	}
	return !now.Before(s.LastDeviceHeartbeat.Add(heartbeatTimeout))
}
