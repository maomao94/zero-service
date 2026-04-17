package builtin

import (
	"zero-service/common/einox/tool"
)

// RegisterAll 把内置工具全部注册到 kit 里。
//
// 分三类注册, 便于 Policy 做 capability 级别的白名单:
//
//	compute: echo, calculator
//	io     : now, random_id
//	human  : ask_confirm, ask_single_choice, ask_multi_choice,
//	         ask_text_input, ask_form_input, ask_info_ack
func RegisterAll(kit *tool.Kit) {
	// compute
	kit.MustRegister(tool.CapCompute, NewEcho())
	kit.MustRegister(tool.CapCompute, NewCalculator())

	// io
	kit.MustRegister(tool.CapIO, NewNow())
	kit.MustRegister(tool.CapIO, NewRandomID())

	// human (6 种中断)
	kit.MustRegister(tool.CapHuman, NewAskConfirm())
	kit.MustRegister(tool.CapHuman, NewAskSingleChoice())
	kit.MustRegister(tool.CapHuman, NewAskMultiChoice())
	kit.MustRegister(tool.CapHuman, NewAskTextInput())
	kit.MustRegister(tool.CapHuman, NewAskFormInput())
	kit.MustRegister(tool.CapHuman, NewAskInfoAck())
}

// NewDefaultKit 返回一个已注册全部内置工具的 Kit 实例。
func NewDefaultKit() *tool.Kit {
	k := tool.NewKit()
	RegisterAll(k)
	return k
}
