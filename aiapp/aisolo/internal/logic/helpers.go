package logic

import (
	"time"

	"github.com/google/uuid"

	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/aiapp/aisolo/internal/session"
	"zero-service/common/einox/protocol"
)

// askStreamSender 把 turn.Executor emit 出来的每一帧 JSON 转成 AskStreamResp 发下去。
type askStreamSender struct {
	stream    aisolo.AiSolo_AskStreamServer
	sessionID string
}

func (s *askStreamSender) Write(p []byte) (int, error) {
	// 协议层已经是一行完整 JSON + \n
	data := string(p)
	if err := s.stream.Send(&aisolo.AskStreamResp{
		Chunk: &aisolo.AskStreamChunk{
			SessionId: s.sessionID,
			Data:      data,
		},
	}); err != nil {
		return 0, err
	}
	return len(p), nil
}

// resumeStreamSender 对应 ResumeStreamResp 的 writer。
type resumeStreamSender struct {
	stream    aisolo.AiSolo_ResumeStreamServer
	sessionID string
}

func (s *resumeStreamSender) Write(p []byte) (int, error) {
	data := string(p)
	if err := s.stream.Send(&aisolo.ResumeStreamResp{
		Chunk: &aisolo.ResumeStreamChunk{
			SessionId: s.sessionID,
			Data:      data,
		},
	}); err != nil {
		return 0, err
	}
	return len(p), nil
}

// newSession 构造新 Session 实例 (ID 自动生成)。
func newSession(userID, title string, mode aisolo.AgentMode) *session.Session {
	now := time.Now()
	if mode == aisolo.AgentMode_AGENT_MODE_UNSPECIFIED {
		mode = aisolo.AgentMode_AGENT_MODE_AGENT
	}
	return &session.Session{
		ID:        uuid.NewString(),
		UserID:    userID,
		Title:     title,
		Mode:      mode,
		Status:    aisolo.SessionStatus_SESSION_STATUS_IDLE,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// toInterruptKind 把 protocol.InterruptKind 映射为 aisolo proto 枚举。
func toInterruptKind(k protocol.InterruptKind) aisolo.InterruptKind {
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

// interruptToProto 把 session.InterruptRecord + protocol.InterruptData 还原为 aisolo.InterruptInfo。
func interruptToProto(r *session.InterruptRecord) *aisolo.InterruptInfo {
	if r == nil {
		return nil
	}
	info := &aisolo.InterruptInfo{
		InterruptId: r.InterruptID,
		Kind:        r.Kind,
		ToolName:    r.ToolName,
		Question:    r.Question,
	}
	d := r.Data
	if d == nil {
		return info
	}
	if info.Kind == aisolo.InterruptKind_INTERRUPT_KIND_UNSPECIFIED {
		info.Kind = toInterruptKind(d.Kind)
	}
	info.ToolName = d.ToolName
	info.Required = d.Required
	info.UiLang = d.UILang
	info.AgentName = d.AgentName
	info.Question = d.Question
	info.Detail = d.Detail
	info.MinSelect = int32(d.MinSelect)
	info.MaxSelect = int32(d.MaxSelect)
	info.Placeholder = d.Placeholder
	info.Multiline = d.Multiline
	info.Title = d.Title
	info.Body = d.Body
	for _, o := range d.Options {
		info.Options = append(info.Options, &aisolo.Option{Id: o.ID, Label: o.Label, Desc: o.Desc})
	}
	for _, f := range d.Fields {
		info.Fields = append(info.Fields, &aisolo.Field{
			Name: f.Name, Label: f.Label, Type: f.Type,
			Required: f.Required, Placeholder: f.Placeholder, Default: f.Default,
			Widget: f.Widget, MinSelect: int32(f.MinSelect), MaxSelect: int32(f.MaxSelect),
			AllowCustom: f.AllowCustom,
		})
		for _, opt := range f.Options {
			info.Fields[len(info.Fields)-1].Options = append(info.Fields[len(info.Fields)-1].Options, &aisolo.Option{
				Id: opt.ID, Label: opt.Label, Desc: opt.Desc,
			})
		}
	}
	return info
}

// toProtoSession 把内部 session.Session 转换为对外 aisolo.Session。
func toProtoSession(s *session.Session) *aisolo.Session {
	if s == nil {
		return nil
	}
	return &aisolo.Session{
		SessionId:    s.ID,
		UserId:       s.UserID,
		Title:        s.Title,
		Mode:         s.Mode,
		Status:       s.Status,
		InterruptId:  s.InterruptID,
		CreatedAt:    s.CreatedAt.Unix(),
		UpdatedAt:    s.UpdatedAt.Unix(),
		MessageCount: s.MessageCount,
		LastMessage:  s.LastMessage,
		UiLang:       s.UILang,
	}
}
