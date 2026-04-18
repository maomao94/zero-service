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
	UILang       string // 会话默认 UI 语言 (zh/en), Ask ui_lang 或 CreateSession 写入; 中断未带 ui_lang 时补齐
	CreatedAt    time.Time
	UpdatedAt    time.Time
	// RunOwner / RunLeaseUntil：分布式下标识「哪一实例持有 RUNNING」及租约到期时间；
	// 进程内 memory 后端可不填，启动时 RecoverRunningSessions 仍全量清 RUNNING。
	RunOwner      string
	RunLeaseUntil time.Time // 零值表示未设置租约（仅 gormx 持久化应在一轮 Ask 开始时写入）
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

	// RecoverRunningSessions 启动时恢复卡住的 RUNNING 会话。
	// memory：全部 RUNNING → IDLE（单实例开发）。
	// gormx/jsonl：仅当租约过期或「无租约且过久未更新」时置 IDLE，避免误清健康实例上的长连接。
	RecoverRunningSessions(ctx context.Context) (int, error)

	Close() error
}

// Config 存储后端配置：memory | jsonl | gormx（gormx 需调用方注入 *gormx.DB）。
type Config struct {
	Type    string
	BaseDir string // jsonl 根目录
	// NullLeaseRecoverGrace：RUNNING 且无租约时，若 UpdatedAt 早于此间隔则启动恢复可清为 IDLE（默认 2m）。
	NullLeaseRecoverGrace time.Duration
}

// LeaseStaleForRecover 判断 RUNNING 会话是否可按「过期租约」策略安全恢复为 IDLE。
func LeaseStaleForRecover(s *Session, now time.Time, nullLeaseGrace time.Duration) bool {
	if s == nil || s.Status != aisolo.SessionStatus_SESSION_STATUS_RUNNING {
		return false
	}
	if !s.RunLeaseUntil.IsZero() {
		return now.After(s.RunLeaseUntil)
	}
	if nullLeaseGrace <= 0 {
		nullLeaseGrace = 2 * time.Minute
	}
	return s.UpdatedAt.Before(now.Add(-nullLeaseGrace))
}
