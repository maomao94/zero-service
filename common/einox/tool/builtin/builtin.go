package builtin

import (
	"fmt"

	ctool "github.com/cloudwego/eino/components/tool"

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
func RegisterAll(kit *tool.Kit) error {
	// compute
	if err := register(kit, tool.CapCompute, NewEchoTool); err != nil {
		return err
	}
	if err := register(kit, tool.CapCompute, NewCalculatorTool); err != nil {
		return err
	}

	// io
	if err := register(kit, tool.CapIO, NewNowTool); err != nil {
		return err
	}
	if err := register(kit, tool.CapIO, NewRandomIDTool); err != nil {
		return err
	}

	// human (6 种中断)
	if err := register(kit, tool.CapHuman, NewAskConfirmTool); err != nil {
		return err
	}
	if err := register(kit, tool.CapHuman, NewAskSingleChoiceTool); err != nil {
		return err
	}
	if err := register(kit, tool.CapHuman, NewAskMultiChoiceTool); err != nil {
		return err
	}
	if err := register(kit, tool.CapHuman, NewAskTextInputTool); err != nil {
		return err
	}
	if err := register(kit, tool.CapHuman, NewAskFormInputTool); err != nil {
		return err
	}
	if err := register(kit, tool.CapHuman, NewAskInfoAckTool); err != nil {
		return err
	}
	return nil
}

// NewDefaultKit 返回一个已注册全部内置工具的 Kit 实例。
func NewDefaultKit() (*tool.Kit, error) {
	k := tool.NewKit()
	if err := RegisterAll(k); err != nil {
		return nil, err
	}
	return k, nil
}

func MustNewDefaultKit() *tool.Kit {
	k, err := NewDefaultKit()
	if err != nil {
		panic(err)
	}
	return k
}

func register(kit *tool.Kit, cap tool.Capability, newTool func() (ctool.InvokableTool, error)) error {
	t, err := newTool()
	if err != nil {
		return fmt.Errorf("builtin: construct %s tool: %w", cap, err)
	}
	if err := kit.Register(cap, t); err != nil {
		return fmt.Errorf("builtin: register %s tool: %w", cap, err)
	}
	return nil
}
