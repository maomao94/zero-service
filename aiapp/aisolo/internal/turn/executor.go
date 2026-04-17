// Package turn 聚合一次对话"轮"的执行流程。
//
// 一个 Turn 的生命周期 (Ask 场景):
//
//  1. 校验 + 加载 session (不存在报错, 状态必须为 IDLE)
//  2. 写入用户消息 + 将 session.status 置 RUNNING
//  3. 加载会话历史 -> []adk.Message
//  4. 从 pool 里拿对应 mode 的 Agent
//  5. runner.Run(ctx, history+user, WithCheckPointID=sessionID)
//  6. PipeEvents 把 AgentEvent 转成 protocol.Event 推给前端
//  7. 收到 Interrupt: 记录 InterruptID, session.status=INTERRUPTED; Emit TurnEnd
//  8. 未中断: 保存 assistant 消息, session.status=IDLE; Emit TurnEnd
//
// Resume 场景类似, 区别在于第 5 步调 runner.ResumeWithParams, 并且 Targets
// 里带上用户给出的答复。
package turn

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/config"
	"zero-service/aiapp/aisolo/internal/modes"
	"zero-service/aiapp/aisolo/internal/session"
	"zero-service/aiapp/aisolo/internal/sessionworkdir"
	"zero-service/aiapp/aisolo/internal/uilang"
	"zero-service/common/einox/fsrestrict"
	"zero-service/common/einox/memory"
	"zero-service/common/einox/metrics"
	mw "zero-service/common/einox/middleware"
	"zero-service/common/einox/protocol"
)

// Executor 一次 turn 的执行器。每个 RPC 请求可以 new 一个, 或在 svcCtx 里复用,
// 它本身是无状态的, 依赖都是指针。
type Executor struct {
	pool     *modes.Pool
	reg      *modes.Registry
	messages memory.Storage
	sessions session.Store
	metrics  *metrics.Metrics
	appCfg   *config.Config
}

// Config Executor 依赖。
type Config struct {
	Pool     *modes.Pool
	Registry *modes.Registry
	Messages memory.Storage
	Sessions session.Store
	Metrics  *metrics.Metrics
	// App 可选；用于会话工作区目录创建等。
	App *config.Config
}

// New 构造 Executor。
func New(cfg Config) *Executor {
	m := cfg.Metrics
	if m == nil {
		m = metrics.Global()
	}
	return &Executor{
		pool:     cfg.Pool,
		reg:      cfg.Registry,
		messages: cfg.Messages,
		sessions: cfg.Sessions,
		metrics:  m,
		appCfg:   cfg.App,
	}
}

// AskInput Ask 请求输入。
type AskInput struct {
	SessionID string
	UserID    string
	Message   string
	Mode      aisolo.AgentMode  // 可选, 留空沿用 session.Mode
	Meta      map[string]string // 透传 gRPC AskReq.meta; 识别 ui_lang / uiLang 更新会话默认 UI 语言
}

// ResumeInput Resume 请求输入。
type ResumeInput struct {
	SessionID   string
	UserID      string
	InterruptID string
	Action      aisolo.ResumeAction

	Reason      string
	SelectedIDs []string
	Text        string
	FormValues  map[string]string
}

// =============================================================================
// Ask
// =============================================================================

// Ask 执行一轮对话, 通过 em 推送事件, 返回是否产生了新中断。
func (e *Executor) Ask(ctx context.Context, em *protocol.Emitter, in AskInput) error {
	start := time.Now()
	sess, err := e.sessions.GetSession(ctx, in.UserID, in.SessionID)
	if err != nil {
		e.fail(em, "session_not_found", err)
		return err
	}
	if sess.Status == aisolo.SessionStatus_SESSION_STATUS_INTERRUPTED {
		err := fmt.Errorf("session is interrupted, call ResumeStream instead")
		e.fail(em, "session_interrupted", err)
		return err
	}
	if sess.Status == aisolo.SessionStatus_SESSION_STATUS_RUNNING {
		// 保护: 虽然启动时会把 RUNNING 清理成 IDLE, 但运行期仍可能遇到并发请求。
		err := fmt.Errorf("session is running, wait for previous turn to finish")
		e.fail(em, "session_running", err)
		return err
	}

	if in.Mode != aisolo.AgentMode_AGENT_MODE_UNSPECIFIED && in.Mode != sess.Mode {
		sess.Mode = in.Mode
	}
	if lang := metaUILangFromMap(in.Meta); lang != "" && lang != sess.UILang {
		sess.UILang = lang
		sess.UpdatedAt = time.Now()
		_ = e.sessions.UpdateSession(ctx, sess)
	}
	effMode := sess.Mode

	agent, err := e.pool.Get(ctx, effMode)
	if err != nil {
		e.fail(em, "build_agent", err)
		return err
	}

	userContent := augmentSkillLaunch(in.Message, in.Meta)
	if e.appCfg != nil {
		if err := sessionworkdir.EnsureSession(*e.appCfg, in.SessionID); err != nil {
			logx.Errorf("[turn] session workspace: %v", err)
		}
	}
	ctx = fsrestrict.WithSessionID(ctx, in.SessionID)

	_ = em.TurnStart(userContent)

	if err := e.saveUserMessage(ctx, sess, userContent); err != nil {
		logx.Errorf("[turn] save user msg: %v", err)
	}

	sess.Status = aisolo.SessionStatus_SESSION_STATUS_RUNNING
	sess.InterruptID = ""
	sess.UpdatedAt = time.Now()
	_ = e.sessions.UpdateSession(ctx, sess)

	history, err := e.loadHistory(ctx, sess)
	if err != nil {
		e.fail(em, "load_history", err)
		return err
	}

	iter := agent.Runner().Run(ctx, history, adk.WithCheckPointID(in.SessionID))
	res, err := protocol.PipeEvents(em, iter, protocol.PipeOptions{SessionUILang: sess.UILang})
	if err != nil {
		e.markSessionIdle(ctx, sess)
		e.metrics.RecordTurn(ctx, modeStr(effMode), "error", time.Since(start))
		_ = em.TurnEnd(false, "", "")
		return err
	}

	if res.HasInterrupt {
		e.persistInterrupt(ctx, sess, res)
		e.metrics.RecordInterrupt(ctx, string(res.InterruptKind), interruptToolMetricLabel(res), res.InterruptID)
		e.metrics.RecordTurn(ctx, modeStr(effMode), "interrupt", time.Since(start))
		_ = em.TurnEnd(true, res.InterruptID, res.LastContent)
		return nil
	}

	if res.LastContent != "" {
		if err := e.saveAssistantMessage(ctx, sess, res.LastContent); err != nil {
			logx.Errorf("[turn] save assistant msg: %v", err)
		}
	}
	e.markSessionIdle(ctx, sess)
	e.metrics.RecordTurn(ctx, modeStr(effMode), "ok", time.Since(start))
	_ = em.TurnEnd(false, "", res.LastContent)
	return nil
}

// =============================================================================
// Resume
// =============================================================================

// Resume 继续一个被中断的 turn。
func (e *Executor) Resume(ctx context.Context, em *protocol.Emitter, in ResumeInput) error {
	start := time.Now()
	sess, err := e.sessions.GetSession(ctx, in.UserID, in.SessionID)
	if err != nil {
		e.fail(em, "session_not_found", err)
		return err
	}
	if sess.Status != aisolo.SessionStatus_SESSION_STATUS_INTERRUPTED {
		err := fmt.Errorf("session is not interrupted (status=%v)", sess.Status)
		e.fail(em, "not_interrupted", err)
		return err
	}
	if sess.InterruptID != in.InterruptID {
		err := fmt.Errorf("interrupt_id mismatch: got %q, expect %q", in.InterruptID, sess.InterruptID)
		e.fail(em, "interrupt_mismatch", err)
		return err
	}

	rec, err := e.sessions.GetInterrupt(ctx, in.InterruptID)
	if err != nil {
		e.fail(em, "no_interrupt_record", err)
		return err
	}

	payload, err := buildResumePayload(rec.Kind, in)
	if err != nil {
		e.fail(em, "invalid_resume_payload", err)
		return err
	}

	agent, err := e.pool.Get(ctx, sess.Mode)
	if err != nil {
		e.fail(em, "build_agent", err)
		return err
	}

	if e.appCfg != nil {
		if err := sessionworkdir.EnsureSession(*e.appCfg, in.SessionID); err != nil {
			logx.Errorf("[turn] session workspace: %v", err)
		}
	}
	ctx = fsrestrict.WithSessionID(ctx, in.SessionID)

	sess.Status = aisolo.SessionStatus_SESSION_STATUS_RUNNING
	sess.UpdatedAt = time.Now()
	_ = e.sessions.UpdateSession(ctx, sess)

	_ = em.TurnStart("")

	iter, err := agent.Runner().ResumeWithParams(ctx, in.SessionID, &adk.ResumeParams{
		Targets: map[string]any{in.InterruptID: payload},
	})
	if err != nil {
		e.fail(em, "resume_start", err)
		e.metrics.RecordResume(ctx, aisoloInterruptKindLabel(rec.Kind), "error", modeStr(sess.Mode), in.InterruptID, resumeActionStr(in.Action), time.Since(start))
		_ = em.TurnEnd(false, "", "")
		return err
	}

	res, err := protocol.PipeEvents(em, iter, protocol.PipeOptions{SessionUILang: sess.UILang})
	if err != nil {
		e.markSessionIdle(ctx, sess)
		e.metrics.RecordTurn(ctx, modeStr(sess.Mode), "error", time.Since(start))
		e.metrics.RecordResume(ctx, aisoloInterruptKindLabel(rec.Kind), "error", modeStr(sess.Mode), in.InterruptID, resumeActionStr(in.Action), time.Since(start))
		_ = em.TurnEnd(false, "", "")
		return err
	}

	if res.HasInterrupt {
		e.persistInterrupt(ctx, sess, res)
		e.metrics.RecordInterrupt(ctx, string(res.InterruptKind), interruptToolMetricLabel(res), res.InterruptID)
		e.metrics.RecordTurn(ctx, modeStr(sess.Mode), "interrupt", time.Since(start))
		e.metrics.RecordResume(ctx, aisoloInterruptKindLabel(rec.Kind), "interrupted_again", modeStr(sess.Mode), in.InterruptID, resumeActionStr(in.Action), time.Since(start))
		_ = em.TurnEnd(true, res.InterruptID, res.LastContent)
		return nil
	}

	if res.LastContent != "" {
		if err := e.saveAssistantMessage(ctx, sess, res.LastContent); err != nil {
			logx.Errorf("[turn] save assistant msg: %v", err)
		}
	}
	e.markSessionIdle(ctx, sess)
	e.metrics.RecordTurn(ctx, modeStr(sess.Mode), "ok", time.Since(start))
	e.metrics.RecordResume(ctx, aisoloInterruptKindLabel(rec.Kind), "ok", modeStr(sess.Mode), in.InterruptID, resumeActionStr(in.Action), time.Since(start))
	_ = em.TurnEnd(false, "", res.LastContent)
	return nil
}

// =============================================================================
// helpers
// =============================================================================

func (e *Executor) fail(em *protocol.Emitter, code string, err error) {
	logx.Errorf("[turn] %s: %v", code, err)
	_ = em.EmitError(code, err.Error())
}

func (e *Executor) markSessionIdle(ctx context.Context, sess *session.Session) {
	sess.Status = aisolo.SessionStatus_SESSION_STATUS_IDLE
	sess.InterruptID = ""
	sess.UpdatedAt = time.Now()
	_ = e.sessions.UpdateSession(ctx, sess)
}

// persistInterrupt 更新 session 为 INTERRUPTED + 落盘完整 InterruptData。
// 前端刷新后通过 GetInterrupt 拿 Data 回填 UI。
func (e *Executor) persistInterrupt(ctx context.Context, sess *session.Session, res protocol.RunResult) {
	sess.Status = aisolo.SessionStatus_SESSION_STATUS_INTERRUPTED
	sess.InterruptID = res.InterruptID
	sess.UpdatedAt = time.Now()
	_ = e.sessions.UpdateSession(ctx, sess)

	rec := &session.InterruptRecord{
		InterruptID: res.InterruptID,
		SessionID:   sess.ID,
		UserID:      sess.UserID,
		Kind:        toProtoKind(res.InterruptKind),
		Data:        res.Interrupt,
		CreatedAt:   time.Now(),
	}
	if res.Interrupt != nil {
		rec.ToolName = res.Interrupt.ToolName
		rec.Question = res.Interrupt.Question
	}
	_ = e.sessions.SaveInterrupt(ctx, rec)
}

func (e *Executor) loadHistory(ctx context.Context, sess *session.Session) ([]adk.Message, error) {
	msgs, err := e.messages.GetMessages(ctx, sess.UserID, sess.ID, 0)
	if err != nil {
		return nil, err
	}
	out := make([]adk.Message, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, m.ToSchemaMessage())
	}
	return out, nil
}

func (e *Executor) saveUserMessage(ctx context.Context, sess *session.Session, text string) error {
	msg := &memory.ConversationMessage{
		ID:        uuid.NewString(),
		SessionID: sess.ID,
		UserID:    sess.UserID,
		Role:      string(schema.User),
		Content:   text,
		CreatedAt: time.Now(),
	}
	if err := e.messages.SaveMessage(ctx, msg); err != nil {
		return err
	}
	sess.MessageCount++
	sess.LastMessage = text
	return nil
}

func (e *Executor) saveAssistantMessage(ctx context.Context, sess *session.Session, text string) error {
	msg := &memory.ConversationMessage{
		ID:        uuid.NewString(),
		SessionID: sess.ID,
		UserID:    sess.UserID,
		Role:      string(schema.Assistant),
		Content:   text,
		CreatedAt: time.Now(),
	}
	if err := e.messages.SaveMessage(ctx, msg); err != nil {
		return err
	}
	sess.MessageCount++
	sess.LastMessage = text
	return nil
}

// metaUILangFromMap 从 Ask 透传的 meta 解析 UI 语言键, 值经 uilang.Normalize.
func metaUILangFromMap(meta map[string]string) string {
	if meta == nil {
		return ""
	}
	for _, k := range []string{"ui_lang", "uiLang", "UILang"} {
		if v := uilang.Normalize(meta[k]); v != "" {
			return v
		}
	}
	return ""
}

// augmentSkillLaunch 若 meta 含 skill_launch，则将其置于用户消息前以触发对应技能上下文。
func augmentSkillLaunch(msg string, meta map[string]string) string {
	if meta == nil {
		return msg
	}
	lp := strings.TrimSpace(meta["skill_launch"])
	if lp == "" {
		return msg
	}
	if strings.TrimSpace(msg) == "" {
		return lp
	}
	return lp + "\n\n" + msg
}

// resumeActionStr 将 Resume 的 YES/NO 打成低基数指标标签。
func resumeActionStr(a aisolo.ResumeAction) string {
	switch a {
	case aisolo.ResumeAction_RESUME_ACTION_YES:
		return "yes"
	case aisolo.ResumeAction_RESUME_ACTION_NO:
		return "no"
	default:
		return "unspecified"
	}
}

func modeStr(m aisolo.AgentMode) string {
	switch m {
	case aisolo.AgentMode_AGENT_MODE_AGENT:
		return "agent"
	case aisolo.AgentMode_AGENT_MODE_WORKFLOW:
		return "workflow"
	case aisolo.AgentMode_AGENT_MODE_SUPERVISOR:
		return "supervisor"
	case aisolo.AgentMode_AGENT_MODE_PLAN:
		return "plan"
	case aisolo.AgentMode_AGENT_MODE_DEEP:
		return "deep"
	default:
		return "unknown"
	}
}

// interruptToolMetricLabel 填 interrupt_total 的 tool 标签：人机中断在 protocol 里带 ToolName；
// 仅 Agent、无工具名时用 agent:<name>；否则 unknown（避免无意义空串）。
func interruptToolMetricLabel(res protocol.RunResult) string {
	if res.Interrupt != nil {
		if t := strings.TrimSpace(res.Interrupt.ToolName); t != "" {
			return t
		}
		if a := strings.TrimSpace(res.Interrupt.AgentName); a != "" {
			return "agent:" + a
		}
	}
	return "unknown"
}

// aisoloInterruptKindLabel 把 aisolo.InterruptKind 转成与 protocol.InterruptKind 一致的短字符串，供指标与日志。
// 禁止对枚举底层 int32 直接 string()：会得到 Unicode 码点（如 4 -> "\\x04"），日志里像空串。
func aisoloInterruptKindLabel(k aisolo.InterruptKind) string {
	switch k {
	case aisolo.InterruptKind_INTERRUPT_KIND_APPROVAL:
		return "approval"
	case aisolo.InterruptKind_INTERRUPT_KIND_SINGLE_SELECT:
		return "single_select"
	case aisolo.InterruptKind_INTERRUPT_KIND_MULTI_SELECT:
		return "multi_select"
	case aisolo.InterruptKind_INTERRUPT_KIND_FREE_TEXT:
		return "free_text"
	case aisolo.InterruptKind_INTERRUPT_KIND_FORM_INPUT:
		return "form_input"
	case aisolo.InterruptKind_INTERRUPT_KIND_INFO_ACK:
		return "info_ack"
	default:
		return "unspecified"
	}
}

// toProtoKind 把 protocol.InterruptKind 转 aisolo.InterruptKind 枚举。
func toProtoKind(k protocol.InterruptKind) aisolo.InterruptKind {
	switch k {
	case protocol.InterruptApproval:
		return aisolo.InterruptKind_INTERRUPT_KIND_APPROVAL
	case protocol.InterruptSingleSelect:
		return aisolo.InterruptKind_INTERRUPT_KIND_SINGLE_SELECT
	case protocol.InterruptMultiSelect:
		return aisolo.InterruptKind_INTERRUPT_KIND_MULTI_SELECT
	case protocol.InterruptFreeText:
		return aisolo.InterruptKind_INTERRUPT_KIND_FREE_TEXT
	case protocol.InterruptFormInput:
		return aisolo.InterruptKind_INTERRUPT_KIND_FORM_INPUT
	case protocol.InterruptInfoAck:
		return aisolo.InterruptKind_INTERRUPT_KIND_INFO_ACK
	default:
		return aisolo.InterruptKind_INTERRUPT_KIND_UNSPECIFIED
	}
}

// buildResumePayload 把 ResumeInput 根据 kind 转成对应的 *mw.XxxResult。
// Action 仅 YES / NO：NO 表示取消或拒绝（按 kind 映射到 Cancelled 或 Approved=false）。
func buildResumePayload(kind aisolo.InterruptKind, in ResumeInput) (any, error) {
	switch in.Action {
	case aisolo.ResumeAction_RESUME_ACTION_NO:
		return cancelPayload(kind, in.Reason), nil
	case aisolo.ResumeAction_RESUME_ACTION_YES:
		// continue
	default:
		return nil, fmt.Errorf("resume action must be YES or NO, got %v", in.Action)
	}

	switch kind {
	case aisolo.InterruptKind_INTERRUPT_KIND_APPROVAL:
		return &mw.ApprovalResult{Approved: true}, nil
	case aisolo.InterruptKind_INTERRUPT_KIND_SINGLE_SELECT,
		aisolo.InterruptKind_INTERRUPT_KIND_MULTI_SELECT:
		return &mw.SelectResult{SelectedIDs: in.SelectedIDs}, nil
	case aisolo.InterruptKind_INTERRUPT_KIND_FREE_TEXT:
		return &mw.TextInputResult{Text: in.Text}, nil
	case aisolo.InterruptKind_INTERRUPT_KIND_FORM_INPUT:
		return &mw.FormInputResult{Values: in.FormValues}, nil
	case aisolo.InterruptKind_INTERRUPT_KIND_INFO_ACK:
		return &mw.InfoAckResult{Ack: true}, nil
	}
	return nil, fmt.Errorf("unsupported kind/action combo: kind=%v action=%v", kind, in.Action)
}

func cancelPayload(kind aisolo.InterruptKind, reason string) any {
	switch kind {
	case aisolo.InterruptKind_INTERRUPT_KIND_APPROVAL:
		r := reason
		return &mw.ApprovalResult{Approved: false, DisapproveReason: &r}
	case aisolo.InterruptKind_INTERRUPT_KIND_SINGLE_SELECT,
		aisolo.InterruptKind_INTERRUPT_KIND_MULTI_SELECT:
		return &mw.SelectResult{Cancelled: true, Reason: reason}
	case aisolo.InterruptKind_INTERRUPT_KIND_FREE_TEXT:
		return &mw.TextInputResult{Cancelled: true, Reason: reason}
	case aisolo.InterruptKind_INTERRUPT_KIND_FORM_INPUT:
		return &mw.FormInputResult{Cancelled: true, Reason: reason}
	case aisolo.InterruptKind_INTERRUPT_KIND_INFO_ACK:
		return &mw.InfoAckResult{Ack: false, Reason: reason}
	}
	return nil
}
