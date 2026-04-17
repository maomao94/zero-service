// Package session 管理会话元数据 + 中断状态。
//
// 与 common/einox/memory (消息历史) 的分工：
//   - memory.Storage : 保存每条对话消息 (user / assistant / tool)
//   - session.Store  : 保存会话本身的元信息 (title, mode, status, interruptID...)
//
// 本包是取代旧 internal/store + internal/resume 的统一入口。
package session

import (
	"context"
	"time"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/common/einox/protocol"
)

// Session 会话元数据。
type Session struct {
	ID           string
	UserID       string
	Title        string
	Mode         aisolo.AgentMode
	Status       aisolo.SessionStatus
	InterruptID  string // Status=INTERRUPTED 时非空
	MessageCount int32
	LastMessage  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// InterruptRecord 记录一次中断细节, 便于 Resume 校验 / 审计 / 页面刷新回填。
//
// Data 保留触发中断时完整的 protocol.InterruptData (question / options / fields ...),
// 前端刷新后通过 GetInterrupt RPC 拿到, 直接重建 UI。
type InterruptRecord struct {
	InterruptID string
	SessionID   string
	UserID      string
	Kind        aisolo.InterruptKind
	ToolName    string
	Question    string
	Data        *protocol.InterruptData
	CreatedAt   time.Time
}

// Store 会话 + 中断的统一存储。实现只需要做好 Session 管理, InterruptRecord 做
// 轻量记录; 中断的 checkpoint 本身由 common/einox/checkpoint 管理。
type Store interface {
	// Session CRUD
	CreateSession(ctx context.Context, s *Session) error
	GetSession(ctx context.Context, userID, sessionID string) (*Session, error)
	ListSessions(ctx context.Context, userID string, page, pageSize int) ([]*Session, int64, error)
	DeleteSession(ctx context.Context, userID, sessionID string) error
	UpdateSession(ctx context.Context, s *Session) error

	// Interrupt 相关
	SaveInterrupt(ctx context.Context, r *InterruptRecord) error
	GetInterrupt(ctx context.Context, interruptID string) (*InterruptRecord, error)

	// ResetRunningToIdle 把所有残留在 RUNNING 状态的会话改回 IDLE。
	// 在服务启动时调用, 防止进程异常退出后 SSE 断开、会话卡死在 RUNNING。
	ResetRunningToIdle(ctx context.Context) (int, error)

	Close() error
}

// Config 存储后端配置。当前仅实现 memory; 后续可扩 jsonl / gormx。
type Config struct {
	Type string // memory (默认)
}
