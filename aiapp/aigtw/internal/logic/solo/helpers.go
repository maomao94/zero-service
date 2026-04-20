// Code written by hand. DO NOT re-generate.
package solo

import (
	"strings"

	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/modeweb"
)

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
	case "yes":
		return aisolo.ResumeAction_RESUME_ACTION_YES
	case "no":
		return aisolo.ResumeAction_RESUME_ACTION_NO
	}
	return aisolo.ResumeAction_RESUME_ACTION_UNSPECIFIED
}

// sessionToType 将 gRPC Session 转为 HTTP types.SoloSessionInfo。
func sessionToType(s *aisolo.Session) *types.SoloSessionInfo {
	if s == nil {
		return nil
	}
	return &types.SoloSessionInfo{
		SessionId:         s.GetSessionId(),
		UserId:            s.GetUserId(),
		Mode:              modeweb.ToSoloString(s.GetMode()),
		Status:            sessionStatusToString(s.GetStatus()),
		InterruptId:       s.GetInterruptId(),
		Title:             s.GetTitle(),
		CreatedAt:         s.GetCreatedAt(),
		UpdatedAt:         s.GetUpdatedAt(),
		MessageCount:      int(s.GetMessageCount()),
		LastMessage:       s.GetLastMessage(),
		UiLang:            s.GetUiLang(),
		KnowledgeBaseId:   s.GetKnowledgeBaseId(),
		KnowledgeBaseName: s.GetKnowledgeBaseName(),
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
		UiLang:      info.GetUiLang(),
		AgentName:   info.GetAgentName(),
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
			Widget: f.GetWidget(), MinSelect: int(f.GetMinSelect()), MaxSelect: int(f.GetMaxSelect()),
			AllowCustom: f.GetAllowCustom(),
		})
		for _, opt := range f.GetOptions() {
			out.Fields[len(out.Fields)-1].Options = append(out.Fields[len(out.Fields)-1].Options, &types.SoloOption{
				Id: opt.GetId(), Label: opt.GetLabel(), Desc: opt.GetDesc(),
			})
		}
	}
	return out
}

// modeToType 将 gRPC ModeInfo 转为 HTTP types.SoloModeInfo。
func modeToType(m *aisolo.ModeInfo) *types.SoloModeInfo {
	if m == nil {
		return nil
	}
	return &types.SoloModeInfo{
		Mode:         modeweb.ToSoloString(m.GetMode()),
		Name:         m.GetName(),
		Description:  m.GetDescription(),
		Capabilities: m.GetCapabilities(),
		Default:      m.GetDefault(),
	}
}
