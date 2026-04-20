package tool

import (
	ctool "github.com/cloudwego/eino/components/tool"
)

// Policy 描述一个 Mode 可以使用的工具白名单 / 能力白名单。
//
// 使用方式：
//
//	p := NewPolicy().
//	    AllowCapabilities(CapCompute, CapHuman).
//	    AllowNames("fetch_url")          // 额外放行若干 io 类工具
//	tools := p.Apply(kit)                 // 返回过滤后的工具列表
//
// 如果 AllowCapabilities 和 AllowNames 都为空, 则默认放行 Kit 里全部工具。
type Policy struct {
	caps  map[Capability]struct{}
	names map[string]struct{}
}

// NewPolicy 创建空策略 (允许所有)。
func NewPolicy() *Policy {
	return &Policy{
		caps:  make(map[Capability]struct{}),
		names: make(map[string]struct{}),
	}
}

// AllowCapabilities 放行某些能力下的全部工具。
func (p *Policy) AllowCapabilities(caps ...Capability) *Policy {
	for _, c := range caps {
		p.caps[c] = struct{}{}
	}
	return p
}

// AllowNames 按名字放行具体工具, 不受 capability 限制。
func (p *Policy) AllowNames(names ...string) *Policy {
	for _, n := range names {
		p.names[n] = struct{}{}
	}
	return p
}

// isOpen 表示策略未做任何限制。
func (p *Policy) isOpen() bool {
	return len(p.caps) == 0 && len(p.names) == 0
}

// Apply 根据策略过滤 Kit, 返回最终可用工具列表。
func (p *Policy) Apply(k *Kit) []ctool.BaseTool {
	if k == nil {
		return nil
	}
	if p == nil || p.isOpen() {
		return k.All()
	}

	entries := k.Entries()
	out := make([]ctool.BaseTool, 0, len(entries))
	for i := range entries {
		e := entries[i]
		if _, ok := p.caps[e.Capability]; ok {
			out = append(out, e.Tool)
			continue
		}
		if _, ok := p.names[e.Name]; ok {
			out = append(out, e.Tool)
		}
	}
	return out
}
