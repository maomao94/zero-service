package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"

	mw "zero-service/common/einox/middleware"
)

// 这 6 个工具 param 类型会被 tool.StatefulInterrupt 写进 checkpoint.
// checkpoint 底层走 encoding/gob, 遇到 interface{} 字段必须要求具体类型注册过,
// 否则会 panic/报错: "gob: type not registered for interface: builtin.askXxxParam".
// 在 init 里统一注册, 保持和 middleware/types.go 一致的风格。
func init() {
	schema.Register[*askConfirmParam]()
	schema.Register[*askSingleParam]()
	schema.Register[*askMultiParam]()
	schema.Register[*askTextParam]()
	schema.Register[*askFormParam]()
	schema.Register[*askInfoParam]()
}

// =============================================================================
// 6 种人机交互工具 —— 全部基于 tool.StatefulInterrupt + GetResumeContext
//
// 交互规约:
//
//  1. 工具第一次被调用: 构造 *XxxInfo, 调 tool.StatefulInterrupt 把原始 args
//     存进去后直接 panic-interrupt. ADK 会把 Info 包进 AgentEvent.Action.Interrupted,
//     einox/protocol/adapter.go 负责转换成 protocol.EventInterrupt 推给前端。
//  2. 用户答复到达 (Resume 时外部会把 *XxxResult 塞进 Runner):
//     工具被第二次调用, 通过 tool.GetResumeContext[*XxxResult] 拿到结果,
//     组装一段 JSON 文本返回给 LLM. LLM 随后基于用户答复继续推理。
//  3. 用户没给结果或取消: 返回 "cancelled: ..." 文本, LLM 基于此文本自行处理。
// =============================================================================

// ------------------------------------------------------------------
// ask_confirm —— 二选一审批
// ------------------------------------------------------------------

type askConfirmParam struct {
	Question string `json:"question" jsonschema:"required,description=让用户确认的问题"`
	Detail   string `json:"detail,omitempty" jsonschema:"description=问题详情, 可留空"`
	UILang   string `json:"ui_lang,omitempty" jsonschema:"description=前端 UI 语言提示, 如 zh/en"`
}

// NewAskConfirm 返回一个"请用户确认/拒绝"的中断工具。
func NewAskConfirm() tool.InvokableTool {
	const name = "ask_confirm"
	t, err := utils.InferTool(name, "AskConfirm: 向用户发起一个二选一审批 (approve / deny), 由用户点击按钮决定是否继续。",
		func(ctx context.Context, in *askConfirmParam) (string, error) {
			wasInterrupted, _, stored := tool.GetInterruptState[*askConfirmParam](ctx)
			if !wasInterrupted {
				return "", tool.StatefulInterrupt(ctx, &mw.ApprovalInfo{
					ToolName: name,
					Question: in.Question,
					Detail:   in.Detail,
					Required: true,
					UILang:   in.UILang,
				}, in)
			}

			isTarget, has, data := tool.GetResumeContext[*mw.ApprovalResult](ctx)
			if isTarget && has {
				if data.Approved {
					return fmt.Sprintf("user approved: %s", stored.Question), nil
				}
				if data.DisapproveReason != nil && *data.DisapproveReason != "" {
					return fmt.Sprintf("user denied: %s (reason: %s)", stored.Question, *data.DisapproveReason), nil
				}
				return fmt.Sprintf("user denied: %s", stored.Question), nil
			}
			return "", tool.StatefulInterrupt(ctx, &mw.ApprovalInfo{
				ToolName: name,
				Question: stored.Question,
				Detail:   stored.Detail,
				Required: true,
				UILang:   stored.UILang,
			}, stored)
		})
	if err != nil {
		panic(err)
	}
	return t
}

// ------------------------------------------------------------------
// ask_single_choice —— 单选
// ------------------------------------------------------------------

type choiceOption struct {
	ID    string `json:"id" jsonschema:"description=选项 ID, 与 value 二选一"`
	Value string `json:"value,omitempty" jsonschema:"description=与 id 等价, 模型常写 value 而漏写 id"`
	Label string `json:"label" jsonschema:"required,description=选项显示文本"`
	Desc  string `json:"desc,omitempty" jsonschema:"description=选项详细描述"`
}

type askSingleParam struct {
	Question string         `json:"question" jsonschema:"required,description=给用户的问题"`
	Options  []choiceOption `json:"options" jsonschema:"required,description=候选项列表"`
	UILang   string         `json:"ui_lang,omitempty" jsonschema:"description=前端 UI 语言提示, 如 zh/en"`
}

// NewAskSingleChoice 单选类中断工具。
func NewAskSingleChoice() tool.InvokableTool {
	const name = "ask_single_choice"
	t, err := utils.InferTool(name, "AskSingleChoice: 向用户提出单选题, 用户从候选项里选一个。",
		func(ctx context.Context, in *askSingleParam) (string, error) {
			wasInterrupted, _, stored := tool.GetInterruptState[*askSingleParam](ctx)
			if !wasInterrupted {
				return "", tool.StatefulInterrupt(ctx, &mw.SelectInfo{
					ToolName: name,
					Question: in.Question,
					Options:  toMwOptions(in.Options),
					Multi:    false,
					Required: true,
					UILang:   in.UILang,
				}, in)
			}

			isTarget, has, data := tool.GetResumeContext[*mw.SelectResult](ctx)
			if isTarget && has {
				if data.Cancelled {
					return fmt.Sprintf("user cancelled: %s", reasonOr(data.Reason, "no reason")), nil
				}
				if len(data.SelectedIDs) == 0 {
					return "user did not select any option", nil
				}
				return fmt.Sprintf("user selected: %s", data.SelectedIDs[0]), nil
			}
			return "", tool.StatefulInterrupt(ctx, &mw.SelectInfo{
				ToolName: name,
				Question: stored.Question,
				Options:  toMwOptions(stored.Options),
				Multi:    false,
				Required: true,
				UILang:   stored.UILang,
			}, stored)
		})
	if err != nil {
		panic(err)
	}
	return t
}

// ------------------------------------------------------------------
// ask_multi_choice —— 多选
// ------------------------------------------------------------------

type askMultiParam struct {
	Question  string         `json:"question" jsonschema:"required"`
	Options   []choiceOption `json:"options" jsonschema:"required"`
	MinSelect int            `json:"min_select,omitempty" jsonschema:"description=最少选择数 (默认 0)"`
	MaxSelect int            `json:"max_select,omitempty" jsonschema:"description=最多选择数 (0 表示不限)"`
	UILang    string         `json:"ui_lang,omitempty" jsonschema:"description=前端 UI 语言提示, 如 zh/en"`
}

// NewAskMultiChoice 多选类中断工具。
func NewAskMultiChoice() tool.InvokableTool {
	const name = "ask_multi_choice"
	t, err := utils.InferTool(name, "AskMultiChoice: 向用户提出多选题, 用户从候选项里选任意个。",
		func(ctx context.Context, in *askMultiParam) (string, error) {
			wasInterrupted, _, stored := tool.GetInterruptState[*askMultiParam](ctx)
			if !wasInterrupted {
				return "", tool.StatefulInterrupt(ctx, &mw.SelectInfo{
					ToolName:  name,
					Question:  in.Question,
					Options:   toMwOptions(in.Options),
					Multi:     true,
					MinSelect: in.MinSelect,
					MaxSelect: in.MaxSelect,
					Required:  in.MinSelect > 0,
					UILang:    in.UILang,
				}, in)
			}

			isTarget, has, data := tool.GetResumeContext[*mw.SelectResult](ctx)
			if isTarget && has {
				if data.Cancelled {
					return fmt.Sprintf("user cancelled: %s", reasonOr(data.Reason, "no reason")), nil
				}
				b, _ := json.Marshal(data.SelectedIDs)
				return fmt.Sprintf("user selected: %s", string(b)), nil
			}
			return "", tool.StatefulInterrupt(ctx, &mw.SelectInfo{
				ToolName:  name,
				Question:  stored.Question,
				Options:   toMwOptions(stored.Options),
				Multi:     true,
				MinSelect: stored.MinSelect,
				MaxSelect: stored.MaxSelect,
				Required:  stored.MinSelect > 0,
				UILang:    stored.UILang,
			}, stored)
		})
	if err != nil {
		panic(err)
	}
	return t
}

// ------------------------------------------------------------------
// ask_text_input —— 自由文本
// ------------------------------------------------------------------

type askTextParam struct {
	Question    string      `json:"question" jsonschema:"required,description=给用户的问题"`
	Placeholder string      `json:"placeholder,omitempty"`
	Multiline   bool        `json:"multiline,omitempty"`
	Required    bool        `json:"required,omitempty"`
	UILang      string      `json:"ui_lang,omitempty" jsonschema:"description=前端 UI 语言提示, 如 zh/en"`
	Fields      []formField `json:"fields,omitempty" jsonschema:"description=问卷字段。非空时按结构化表单交互, 可表达下拉/单选/多选等"`
}

// NewAskTextInput 自由文本输入类中断工具。
func NewAskTextInput() tool.InvokableTool {
	const name = "ask_text_input"
	t, err := utils.InferTool(name, "AskTextInput: 向用户征集一段自由文本 (单行或多行)。",
		func(ctx context.Context, in *askTextParam) (string, error) {
			wasInterrupted, _, stored := tool.GetInterruptState[*askTextParam](ctx)
			if !wasInterrupted {
				if len(in.Fields) > 0 {
					return "", tool.StatefulInterrupt(ctx, &mw.FormInputInfo{
						ToolName: name,
						Question: in.Question,
						Fields:   toMwFields(in.Fields),
						Required: in.Required,
						UILang:   in.UILang,
					}, in)
				}
				return "", tool.StatefulInterrupt(ctx, &mw.TextInputInfo{
					ToolName:    name,
					Question:    in.Question,
					Placeholder: in.Placeholder,
					Multiline:   in.Multiline,
					Required:    in.Required,
					UILang:      in.UILang,
				}, in)
			}

			if len(stored.Fields) > 0 {
				isTarget, has, data := tool.GetResumeContext[*mw.FormInputResult](ctx)
				if isTarget && has {
					if data.Cancelled {
						return fmt.Sprintf("user cancelled: %s", reasonOr(data.Reason, "no reason")), nil
					}
					b, _ := json.Marshal(data.Values)
					return fmt.Sprintf("user submitted: %s", string(b)), nil
				}
				return "", tool.StatefulInterrupt(ctx, &mw.FormInputInfo{
					ToolName: name,
					Question: stored.Question,
					Fields:   toMwFields(stored.Fields),
					Required: stored.Required,
					UILang:   stored.UILang,
				}, stored)
			}

			isTarget, has, data := tool.GetResumeContext[*mw.TextInputResult](ctx)
			if isTarget && has {
				if data.Cancelled {
					return fmt.Sprintf("user cancelled: %s", reasonOr(data.Reason, "no reason")), nil
				}
				return fmt.Sprintf("user text: %s", data.Text), nil
			}
			return "", tool.StatefulInterrupt(ctx, &mw.TextInputInfo{
				ToolName:    name,
				Question:    stored.Question,
				Placeholder: stored.Placeholder,
				Multiline:   stored.Multiline,
				Required:    stored.Required,
				UILang:      stored.UILang,
			}, stored)
		})
	if err != nil {
		panic(err)
	}
	return t
}

// ------------------------------------------------------------------
// ask_form_input —— 结构化表单
// ------------------------------------------------------------------

type formField struct {
	Name        string         `json:"name" jsonschema:"required,description=字段名, 作为提交时的 key"`
	Label       string         `json:"label" jsonschema:"required,description=字段显示文本"`
	Type        string         `json:"type" jsonschema:"required,description=字段类型, string|number|boolean"`
	Required    bool           `json:"required,omitempty"`
	Placeholder string         `json:"placeholder,omitempty"`
	Default     string         `json:"default,omitempty"`
	Widget      string         `json:"widget,omitempty" jsonschema:"description=控件类型, text|textarea|select|radio|multi_select(多选下拉, 勿写 multiselect)|checkbox|switch|number"`
	Options     []choiceOption `json:"options,omitempty" jsonschema:"description=候选项, 下拉/单选/多选类控件使用"`
	MinSelect   int            `json:"min_select,omitempty" jsonschema:"description=多选控件最小选择数"`
	MaxSelect   int            `json:"max_select,omitempty" jsonschema:"description=多选控件最大选择数, 0 表示不限"`
	AllowCustom bool           `json:"allow_custom,omitempty" jsonschema:"description=为 true 时在单选/下拉/多选下增加「其他」自定义输入, 固定选项场景可省略"`
}

type askFormParam struct {
	Question string      `json:"question" jsonschema:"required"`
	Fields   []formField `json:"fields" jsonschema:"required"`
	Required bool        `json:"required,omitempty" jsonschema:"description=是否必须提交表单"`
	UILang   string      `json:"ui_lang,omitempty" jsonschema:"description=前端 UI 语言提示, 如 zh/en"`
}

// NewAskFormInput 表单类中断工具。
func NewAskFormInput() tool.InvokableTool {
	const name = "ask_form_input"
	t, err := utils.InferTool(name, "AskFormInput: 向用户征集一个结构化表单 (多字段, 每字段有名字/标签/类型)。",
		func(ctx context.Context, in *askFormParam) (string, error) {
			wasInterrupted, _, stored := tool.GetInterruptState[*askFormParam](ctx)
			if !wasInterrupted {
				return "", tool.StatefulInterrupt(ctx, &mw.FormInputInfo{
					ToolName: name,
					Question: in.Question,
					Fields:   toMwFields(in.Fields),
					Required: in.Required,
					UILang:   in.UILang,
				}, in)
			}

			isTarget, has, data := tool.GetResumeContext[*mw.FormInputResult](ctx)
			if isTarget && has {
				if data.Cancelled {
					return fmt.Sprintf("user cancelled: %s", reasonOr(data.Reason, "no reason")), nil
				}
				b, _ := json.Marshal(data.Values)
				return fmt.Sprintf("user submitted: %s", string(b)), nil
			}
			return "", tool.StatefulInterrupt(ctx, &mw.FormInputInfo{
				ToolName: name,
				Question: stored.Question,
				Fields:   toMwFields(stored.Fields),
				Required: stored.Required,
				UILang:   stored.UILang,
			}, stored)
		})
	if err != nil {
		panic(err)
	}
	return t
}

// ------------------------------------------------------------------
// ask_info_ack —— 展示信息 + 确认继续
// ------------------------------------------------------------------

type askInfoParam struct {
	Title  string `json:"title" jsonschema:"required,description=信息标题"`
	Body   string `json:"body" jsonschema:"required,description=信息正文, 支持 markdown"`
	UILang string `json:"ui_lang,omitempty" jsonschema:"description=前端 UI 语言提示, 如 zh/en"`
}

// NewAskInfoAck 展示-确认类中断工具。
func NewAskInfoAck() tool.InvokableTool {
	const name = "ask_info_ack"
	t, err := utils.InferTool(name, "AskInfoAck: 向用户展示一段信息, 要求用户点击 '我知道了' 才能继续。",
		func(ctx context.Context, in *askInfoParam) (string, error) {
			wasInterrupted, _, stored := tool.GetInterruptState[*askInfoParam](ctx)
			if !wasInterrupted {
				return "", tool.StatefulInterrupt(ctx, &mw.InfoAckInfo{
					ToolName: name,
					Title:    in.Title,
					Body:     in.Body,
					UILang:   in.UILang,
				}, in)
			}

			isTarget, has, data := tool.GetResumeContext[*mw.InfoAckResult](ctx)
			if isTarget && has {
				if data.Ack {
					return "user acknowledged", nil
				}
				return fmt.Sprintf("user cancelled: %s", reasonOr(data.Reason, "no reason")), nil
			}
			return "", tool.StatefulInterrupt(ctx, &mw.InfoAckInfo{
				ToolName: name,
				Title:    stored.Title,
				Body:     stored.Body,
				UILang:   stored.UILang,
			}, stored)
		})
	if err != nil {
		panic(err)
	}
	return t
}

// =============================================================================
// 辅助
// =============================================================================

func toMwOptions(src []choiceOption) []mw.InterruptOption {
	out := make([]mw.InterruptOption, 0, len(src))
	for _, o := range src {
		id := strings.TrimSpace(o.ID)
		if id == "" {
			id = strings.TrimSpace(o.Value)
		}
		out = append(out, mw.InterruptOption{ID: id, Label: o.Label, Desc: o.Desc})
	}
	return out
}

// normalizeFormWidget 把模型常见写法统一成前端识别的 widget 名。
func normalizeFormWidget(w string) string {
	s := strings.ToLower(strings.TrimSpace(w))
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	switch s {
	case "multiselect", "multi_select":
		return "multi_select"
	default:
		return strings.TrimSpace(w)
	}
}

func toMwFields(src []formField) []mw.FormField {
	out := make([]mw.FormField, 0, len(src))
	for _, f := range src {
		out = append(out, mw.FormField{
			Name:        f.Name,
			Label:       f.Label,
			Type:        f.Type,
			Required:    f.Required,
			Placeholder: f.Placeholder,
			Default:     f.Default,
			Widget:      normalizeFormWidget(f.Widget),
			Options:     toMwOptions(f.Options),
			MinSelect:   f.MinSelect,
			MaxSelect:   f.MaxSelect,
			AllowCustom: f.AllowCustom,
		})
	}
	return out
}

func reasonOr(reason, def string) string {
	if reason == "" {
		return def
	}
	return reason
}
