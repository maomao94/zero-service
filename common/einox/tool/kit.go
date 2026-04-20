// Package tool 提供 einox 统一的工具管理。
//
// 设计要点：
//
//  1. Kit 是一个按能力分桶的工具注册表。桶名约定三类：
//     - "compute": 纯计算, 无副作用 (echo, calculator, time, etc)
//     - "io"     : 外部副作用 (HTTP、DB 读、文件读)
//     - "human"  : 人机交互, 通过 Interrupt / Resume 让用户介入
//     - 其它业务自定义分类也 OK, 比如 "search"
//  2. Policy 描述一个 Mode 可以使用哪些 Capability / 工具, 由 mode 侧构造。
//  3. 工具实现都放在 common/einox/tool/builtin, 按 capability 分文件。
package tool

import (
	"context"
	"fmt"
	"sort"
	"sync"

	ctool "github.com/cloudwego/eino/components/tool"
)

// Capability 工具能力分类标签。
type Capability string

const (
	CapCompute Capability = "compute"
	CapIO      Capability = "io"
	CapHuman   Capability = "human"
)

// Entry 注册到 Kit 里的一个工具条目。
type Entry struct {
	Tool       ctool.BaseTool
	Capability Capability
	Name       string
	Desc       string
}

// Kit 按能力分桶的工具注册表, 并发安全。
type Kit struct {
	mu sync.RWMutex
	m  map[string]*Entry
}

// NewKit 创建空 Kit。
func NewKit() *Kit {
	return &Kit{m: make(map[string]*Entry)}
}

// Register 注册工具; 重复名字返回 error。
func (k *Kit) Register(cap Capability, t ctool.BaseTool) error {
	if t == nil {
		return fmt.Errorf("tool: nil tool")
	}
	info, err := t.Info(context.Background())
	if err != nil {
		return fmt.Errorf("tool: get info: %w", err)
	}
	if info == nil || info.Name == "" {
		return fmt.Errorf("tool: empty info")
	}

	k.mu.Lock()
	defer k.mu.Unlock()
	if _, ok := k.m[info.Name]; ok {
		return fmt.Errorf("tool: %q already registered", info.Name)
	}
	k.m[info.Name] = &Entry{
		Tool:       t,
		Capability: cap,
		Name:       info.Name,
		Desc:       info.Desc,
	}
	return nil
}

// MustRegister 注册失败 panic, 用于 init/启动期。
func (k *Kit) MustRegister(cap Capability, t ctool.BaseTool) {
	if err := k.Register(cap, t); err != nil {
		panic(err)
	}
}

// Get 按名取出 Entry。
func (k *Kit) Get(name string) (*Entry, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	e, ok := k.m[name]
	return e, ok
}

// ByCapability 返回某一 capability 下的所有工具, 按名字升序。
func (k *Kit) ByCapability(cap Capability) []ctool.BaseTool {
	k.mu.RLock()
	defer k.mu.RUnlock()

	names := make([]string, 0)
	for n, e := range k.m {
		if e.Capability == cap {
			names = append(names, n)
		}
	}
	sort.Strings(names)

	out := make([]ctool.BaseTool, 0, len(names))
	for _, n := range names {
		out = append(out, k.m[n].Tool)
	}
	return out
}

// All 返回所有工具, 按名字升序。
func (k *Kit) All() []ctool.BaseTool {
	k.mu.RLock()
	defer k.mu.RUnlock()

	names := make([]string, 0, len(k.m))
	for n := range k.m {
		names = append(names, n)
	}
	sort.Strings(names)

	out := make([]ctool.BaseTool, 0, len(names))
	for _, n := range names {
		out = append(out, k.m[n].Tool)
	}
	return out
}

// Entries 返回所有 Entry (只读副本), 按名字升序。
func (k *Kit) Entries() []Entry {
	k.mu.RLock()
	defer k.mu.RUnlock()

	names := make([]string, 0, len(k.m))
	for n := range k.m {
		names = append(names, n)
	}
	sort.Strings(names)

	out := make([]Entry, 0, len(names))
	for _, n := range names {
		e := k.m[n]
		out = append(out, *e)
	}
	return out
}

// Select 返回指定名字的工具; 未找到的名字被静默忽略, 返回已解析的工具列表。
// 结果按输入顺序。
func (k *Kit) Select(names ...string) []ctool.BaseTool {
	k.mu.RLock()
	defer k.mu.RUnlock()
	out := make([]ctool.BaseTool, 0, len(names))
	for _, n := range names {
		if e, ok := k.m[n]; ok {
			out = append(out, e.Tool)
		}
	}
	return out
}
