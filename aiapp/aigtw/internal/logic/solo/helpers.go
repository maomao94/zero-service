// Code written by hand. DO NOT re-generate.
package solo

import (
	"strings"

	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/aiapp/aisolo/aisolo"
)

// parseMode 将外部的字符串 mode 映射为 gRPC 枚举。空字符串 / 未知值返回 UNSPECIFIED，
// 由下游 aisolo 侧决定是否回退到默认 mode。
func parseMode(s string) aisolo.AgentMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "default":
		return aisolo.AgentMode_AGENT_MODE_UNSPECIFIED
	case "agent":
		return aisolo.AgentMode_AGENT_MODE_AGENT
	case "workflow":
		return aisolo.AgentMode_AGENT_MODE_WORKFLOW
	case "supervisor":
		return aisolo.AgentMode_AGENT_MODE_SUPERVISOR
	case "plan", "plan-execute", "planexecute":
		return aisolo.AgentMode_AGENT_MODE_PLAN
	case "deep", "deepagent", "deep-agent":
		return aisolo.AgentMode_AGENT_MODE_DEEP
	}
	return aisolo.AgentMode_AGENT_MODE_UNSPECIFIED
}

// modeToString 将 gRPC mode 枚举转为对外的字符串。
func modeToString(m aisolo.AgentMode) string {
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
	}
	return ""
}

// sessionStatusToString 会话状态枚举 -> 字符串。
func sessionStatusToString(s aisolo.SessionStatus) string {
	switch s {
	case aisolo.SessionStatus_SESSION_STATUS_IDLE:
		return "idle"
	case aisolo.SessionStatus_SESSION_STATUS_RUNNING:
		return "running"
	case aisolo.SessionStatus_SESSION_STATUS_INTERRUPTED:
		return "interrupted"
	}
	return ""
}

// parseResumeAction 将字符串 action 映射为 gRPC 枚举。
func parseResumeAction(s string) aisolo.ResumeAction {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "approve", "approved":
		return aisolo.ResumeAction_RESUME_ACTION_APPROVE
	case "deny", "reject":
		return aisolo.ResumeAction_RESUME_ACTION_DENY
	case "select":
		return aisolo.ResumeAction_RESUME_ACTION_SELECT
	case "text":
		return aisolo.ResumeAction_RESUME_ACTION_TEXT
	case "form":
		return aisolo.ResumeAction_RESUME_ACTION_FORM
	case "ack", "acknowledge":
		return aisolo.ResumeAction_RESUME_ACTION_ACK
	case "cancel", "canceled":
		return aisolo.ResumeAction_RESUME_ACTION_CANCEL
	}
	return aisolo.ResumeAction_RESUME_ACTION_UNSPECIFIED
}

// sessionToType 将 gRPC Session 转为 HTTP types.SoloSessionInfo。
func sessionToType(s *aisolo.Session) *types.SoloSessionInfo {
	if s == nil {
		return nil
	}
	return &types.SoloSessionInfo{
		SessionId:    s.GetSessionId(),
		UserId:       s.GetUserId(),
		Mode:         modeToString(s.GetMode()),
		Status:       sessionStatusToString(s.GetStatus()),
		InterruptId:  s.GetInterruptId(),
		Title:        s.GetTitle(),
		CreatedAt:    s.GetCreatedAt(),
		UpdatedAt:    s.GetUpdatedAt(),
		MessageCount: int(s.GetMessageCount()),
		LastMessage:  s.GetLastMessage(),
	}
}

// messageToType 将 gRPC Message 转为 HTTP types.SoloMessageInfo。
func messageToType(m *aisolo.Message) *types.SoloMessageInfo {
	if m == nil {
		return nil
	}
	return &types.SoloMessageInfo{
		Id:         m.GetId(),
		SessionId:  m.GetSessionId(),
		UserId:     m.GetUserId(),
		Role:       m.GetRole(),
		Content:    m.GetContent(),
		CreatedAt:  m.GetCreatedAt(),
		ToolCallId: m.GetToolCallId(),
		ToolName:   m.GetToolName(),
	}
}

// interruptKindToString 将 gRPC InterruptKind 转为对外字符串 (与 protocol.InterruptKind 保持一致)。
func interruptKindToString(k aisolo.InterruptKind) string {
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
	}
	return ""
}

// interruptToType 将 gRPC InterruptInfo 转为 HTTP types.SoloInterruptInfo。
func interruptToType(info *aisolo.InterruptInfo) *types.SoloInterruptInfo {
	if info == nil {
		return nil
	}
	out := &types.SoloInterruptInfo{
		InterruptId: info.GetInterruptId(),
		Kind:        interruptKindToString(info.GetKind()),
		ToolName:    info.GetToolName(),
		Required:    info.GetRequired(),
		Question:    info.GetQuestion(),
		Detail:      info.GetDetail(),
		MinSelect:   int(info.GetMinSelect()),
		MaxSelect:   int(info.GetMaxSelect()),
		Placeholder: info.GetPlaceholder(),
		Multiline:   info.GetMultiline(),
		Title:       info.GetTitle(),
		Body:        info.GetBody(),
	}
	for _, o := range info.GetOptions() {
		out.Options = append(out.Options, &types.SoloOption{
			Id: o.GetId(), Label: o.GetLabel(), Desc: o.GetDesc(),
		})
	}
	for _, f := range info.GetFields() {
		out.Fields = append(out.Fields, &types.SoloField{
			Name: f.GetName(), Label: f.GetLabel(), Type: f.GetType(),
			Required: f.GetRequired(), Placeholder: f.GetPlaceholder(), Default: f.GetDefault(),
		})
	}
	return out
}

// modeToType 将 gRPC ModeInfo 转为 HTTP types.SoloModeInfo。
func modeToType(m *aisolo.ModeInfo) *types.SoloModeInfo {
	if m == nil {
		return nil
	}
	return &types.SoloModeInfo{
		Mode:         modeToString(m.GetMode()),
		Name:         m.GetName(),
		Description:  m.GetDescription(),
		Capabilities: m.GetCapabilities(),
		Default:      m.GetDefault(),
	}
}
