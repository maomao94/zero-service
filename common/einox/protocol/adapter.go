package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"

	"zero-service/common/einox/middleware"
)

// RunResult 一次 Run / Resume 的业务层结果汇总。
//
// 除事件已经通过 Emitter 推给前端以外, 上层 aisolo 还需要这些结构化信息来:
//   - LastContent → 写入消息历史;
//   - HasInterrupt / InterruptID → 保存到 session.InterruptID, 并落盘等待 Resume;
//   - InterruptKind → 前端 UI 判断已经由事件解决, 这里留给服务端日志/指标使用.
type RunResult struct {
	LastContent   string
	HasInterrupt  bool
	InterruptID   string
	InterruptKind InterruptKind
	// Interrupt 是中断发生时的完整 protocol.InterruptData 副本,
	// 调用方可以把它落进 session.InterruptRecord, 前端刷新后通过
	// GetInterrupt RPC 拿到它直接重建 UI。
	Interrupt *InterruptData
}

// PipeOptions 控制 PipeEvents 的少量横切行为 (如会话级 UI 语言)。
type PipeOptions struct {
	// SessionUILang 在工具未声明 ui_lang 时写入中断事件与 RunResult, 与前端 i18n 对齐。
	SessionUILang string
}

// PipeEvents 消费 adk AgentEvent 异步流, 按协议 emit 事件, 并返回 RunResult。
//
// 不负责 turn.start / turn.end —— 调用方（turn.Executor）来包裹一轮生命周期。
func PipeEvents(em *Emitter, iter *adk.AsyncIterator[*adk.AgentEvent], opt PipeOptions) (RunResult, error) {
	var res RunResult

	for {
		ev, ok := iter.Next()
		if !ok {
			return res, nil
		}

		if ev.Err != nil {
			logx.Errorf("[protocol] agent event err: %v", ev.Err)
			_ = em.EmitError("agent_error", ev.Err.Error())
			return res, ev.Err
		}

		// 中断优先级最高, 中断后这一轮就结束了。
		if ev.Action != nil && ev.Action.Interrupted != nil {
			data := extractInterrupt(ev.Action.Interrupted.InterruptContexts)
			data.AgentName = ev.AgentName
			if strings.TrimSpace(data.UILang) == "" {
				if def := strings.TrimSpace(opt.SessionUILang); def != "" {
					data.UILang = def
				}
			}
			res.HasInterrupt = true
			res.InterruptID = data.InterruptID
			res.InterruptKind = data.Kind
			dataCopy := data
			res.Interrupt = &dataCopy
			_ = em.Emit(EventInterrupt, dataCopy)
			return res, nil
		}

		hasOutput := ev.Output != nil && ev.Output.MessageOutput != nil
		exit := ev.Action != nil && ev.Action.Exit

		if !hasOutput {
			if exit {
				return res, nil
			}
			continue
		}

		mo := ev.Output.MessageOutput
		role := mo.Role
		if role == "" && mo.Message != nil {
			role = mo.Message.Role
		}

		switch role {
		case schema.Tool:
			emitToolResult(em, mo, ev.AgentName)
		default:
			text := emitAssistantMessage(em, role, mo, ev.AgentName)
			if text != "" {
				res.LastContent = text
			}
		}

		if exit {
			return res, nil
		}
	}
}

// =============================================================================
// assistant 消息
// =============================================================================

func emitAssistantMessage(em *Emitter, role schema.RoleType, mo *adk.MessageVariant, agentName string) string {
	if mo.IsStreaming && mo.MessageStream != nil {
		return emitAssistantStream(em, role, mo.MessageStream, agentName)
	}
	if mo.Message == nil {
		return ""
	}
	return emitAssistantOneShot(em, role, mo.Message, agentName)
}

// emitAssistantStream 把一条流式 assistant 消息拆成 message.start / delta... / end
// + 若干 tool.call.start (工具调用在流末尾才有稳定的 args)。
func emitAssistantStream(em *Emitter, role schema.RoleType, stream *schema.StreamReader[adk.Message], agentName string) string {
	msgID := uuid.NewString()
	roleStr := toMessageRole(role)

	var (
		content       strings.Builder
		started       bool
		toolCalls     = map[int]*toolAcc{}
		toolCallOrder []int
	)

	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			logx.Errorf("[protocol] assistant stream recv: %v", err)
			break
		}

		for _, tc := range chunk.ToolCalls {
			idx := 0
			if tc.Index != nil {
				idx = *tc.Index
			}
			acc, ok := toolCalls[idx]
			if !ok {
				acc = &toolAcc{ID: tc.ID}
				toolCalls[idx] = acc
				toolCallOrder = append(toolCallOrder, idx)
			}
			if tc.ID != "" && acc.ID == "" {
				acc.ID = tc.ID
			}
			if tc.Function.Name != "" && acc.Name == "" {
				acc.Name = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				acc.Args.WriteString(tc.Function.Arguments)
			}
		}

		if chunk.Content != "" {
			if !started {
				_ = em.Emit(EventMessageStart, MessageStartData{
					MessageID: msgID,
					Role:      roleStr,
					AgentName: agentName,
				})
				started = true
			}
			content.WriteString(chunk.Content)
			_ = em.Emit(EventMessageDelta, MessageDeltaData{
				MessageID: msgID,
				Text:      chunk.Content,
				AgentName: agentName,
			})
		}
	}

	if started {
		_ = em.Emit(EventMessageEnd, MessageEndData{
			MessageID: msgID,
			Role:      roleStr,
			Text:      content.String(),
			AgentName: agentName,
		})
	}

	for _, idx := range toolCallOrder {
		acc := toolCalls[idx]
		if acc.Name == "" {
			continue
		}
		_ = em.Emit(EventToolCallStart, ToolCallStartData{
			CallID:    acc.ID,
			Tool:      acc.Name,
			ArgsJSON:  acc.Args.String(),
			MessageID: msgID,
			AgentName: agentName,
		})
	}

	return content.String()
}

func emitAssistantOneShot(em *Emitter, role schema.RoleType, msg *schema.Message, agentName string) string {
	msgID := uuid.NewString()
	roleStr := toMessageRole(role)

	if msg.Content != "" {
		_ = em.Emit(EventMessageStart, MessageStartData{MessageID: msgID, Role: roleStr, AgentName: agentName})
		_ = em.Emit(EventMessageEnd, MessageEndData{
			MessageID: msgID,
			Role:      roleStr,
			Text:      msg.Content,
			AgentName: agentName,
		})
	}

	for _, tc := range msg.ToolCalls {
		_ = em.Emit(EventToolCallStart, ToolCallStartData{
			CallID:    tc.ID,
			Tool:      tc.Function.Name,
			ArgsJSON:  tc.Function.Arguments,
			MessageID: msgID,
			AgentName: agentName,
		})
	}
	return msg.Content
}

// =============================================================================
// tool 结果
// =============================================================================

// toolPayloadForEmit 把工具返回体映射到协议里的 result / error。
//
// 约定：若 JSON 对象里存在非空字符串字段 error，视为「软失败」（与 calculator 等工具一致），
// 填入 ToolCallEndData.Error，便于前端用告警样式展示且与 ADK 致命错误区分。
// 若对象仅有 result 字符串字段，则只下发该字符串，避免 UI 重复一层 JSON。
func toolPayloadForEmit(content string) (result, errMsg string) {
	s := strings.TrimSpace(content)
	if s == "" {
		return "", ""
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return s, ""
	}
	if raw, ok := m["error"]; ok {
		var es string
		if json.Unmarshal(raw, &es) == nil && strings.TrimSpace(es) != "" {
			return "", es
		}
	}
	if raw, ok := m["result"]; ok && len(m) == 1 {
		var rs string
		if json.Unmarshal(raw, &rs) == nil {
			return rs, ""
		}
	}
	return s, ""
}

func emitToolResult(em *Emitter, mo *adk.MessageVariant, agentName string) {
	var (
		content strings.Builder
		callID  string
		name    string
	)

	if mo.IsStreaming && mo.MessageStream != nil {
		for {
			chunk, err := mo.MessageStream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				logx.Errorf("[protocol] tool stream recv: %v", err)
				break
			}
			content.WriteString(chunk.Content)
			if callID == "" {
				callID = chunk.ToolCallID
			}
			if name == "" {
				name = chunk.ToolName
			}
		}
	} else if mo.Message != nil {
		content.WriteString(mo.Message.Content)
		callID = mo.Message.ToolCallID
		name = mo.Message.ToolName
	}

	body := content.String()
	res, errStr := toolPayloadForEmit(body)
	_ = em.Emit(EventToolCallEnd, ToolCallEndData{
		CallID:    callID,
		Tool:      name,
		Result:    res,
		Error:     errStr,
		AgentName: agentName,
	})
}

// =============================================================================
// 中断
// =============================================================================

// extractInterrupt 从 adk InterruptContexts 里挑 root-cause, 按 Info 的具体
// 类型转换为 InterruptData。未识别的类型使用 free_text 承载错误说明（不伪装为 approval）。
func extractInterrupt(ctxs []*adk.InterruptCtx) InterruptData {
	if len(ctxs) == 0 {
		return InterruptData{
			Kind:      InterruptFreeText,
			Question:  "Missing interrupt context",
			Required:  false,
			Multiline: false,
		}
	}
	chosen := ctxs[0]
	for _, ic := range ctxs {
		if ic.IsRootCause {
			chosen = ic
			break
		}
	}

	d := InterruptData{
		InterruptID: chosen.ID,
	}

	switch info := chosen.Info.(type) {
	case *middleware.ApprovalInfo:
		d.Kind = InterruptApproval
		d.Question = info.Question
		d.Detail = info.Detail
		d.ToolName = info.ToolName
		d.Required = info.Required
		d.UILang = info.UILang

	case *middleware.SelectInfo:
		d.Kind = InterruptSingleSelect
		if info.Multi {
			d.Kind = InterruptMultiSelect
		}
		d.Question = info.Question
		d.ToolName = info.ToolName
		d.MinSelect = info.MinSelect
		d.MaxSelect = info.MaxSelect
		d.Required = info.Required
		d.UILang = info.UILang
		d.Options = toProtoOptions(info.Options)

	case *middleware.TextInputInfo:
		d.Kind = InterruptFreeText
		d.Question = info.Question
		d.Placeholder = info.Placeholder
		d.Multiline = info.Multiline
		d.ToolName = info.ToolName
		d.Required = info.Required
		d.UILang = info.UILang

	case *middleware.FormInputInfo:
		d.Kind = InterruptFormInput
		d.Question = info.Question
		d.Fields = toProtoFields(info.Fields)
		d.ToolName = info.ToolName
		d.Required = info.Required
		d.UILang = info.UILang

	case *middleware.InfoAckInfo:
		d.Kind = InterruptInfoAck
		d.Title = info.Title
		d.Body = info.Body
		d.ToolName = info.ToolName
		d.UILang = info.UILang

	default:
		d.Kind = InterruptFreeText
		d.Question = "Unsupported interrupt type"
		d.Detail = fmt.Sprintf("%v", chosen.Info)
		d.Required = false
		d.Multiline = false
	}

	return d
}

func toProtoOptions(src []middleware.InterruptOption) []Option {
	if len(src) == 0 {
		return nil
	}
	out := make([]Option, 0, len(src))
	for _, o := range src {
		out = append(out, Option{ID: o.ID, Label: o.Label, Desc: o.Desc})
	}
	return out
}

func toProtoFields(src []middleware.FormField) []Field {
	if len(src) == 0 {
		return nil
	}
	out := make([]Field, 0, len(src))
	for _, f := range src {
		out = append(out, Field{
			Name:        f.Name,
			Label:       f.Label,
			Type:        f.Type,
			Required:    f.Required,
			Placeholder: f.Placeholder,
			Default:     f.Default,
			Widget:      f.Widget,
			Options:     toProtoOptions(f.Options),
			MinSelect:   f.MinSelect,
			MaxSelect:   f.MaxSelect,
			AllowCustom: f.AllowCustom,
		})
	}
	return out
}

// =============================================================================
// 辅助
// =============================================================================

type toolAcc struct {
	ID   string
	Name string
	Args strings.Builder
}

func toMessageRole(r schema.RoleType) MessageRole {
	switch r {
	case schema.User:
		return RoleUser
	case schema.Assistant:
		return RoleAssistant
	case schema.Tool:
		return RoleTool
	case schema.System:
		return RoleSystem
	default:
		if r != "" {
			return MessageRole(r)
		}
		return RoleAssistant
	}
}
