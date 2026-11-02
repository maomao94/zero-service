package gtwx

import (
	"sync/atomic"

	"zero-service/common/errx"
	"zero-service/third_party/extproto"
)

// FoldConfig 是网关错误折叠配置。
//
// 默认零值表示不折叠：所有已登记的平台码原样透传。
// 只有 Enabled=true 且 code 命中 Codes 时才折叠为 Target。
type FoldConfig struct {
	Enabled bool  `json:",optional,default=false"`
	Target  int   `json:",optional"`
	Codes   []int `json:",optional"`
	KeepMsg bool  `json:",optional,default=false"`
}

type foldPolicy struct {
	enabled bool
	target  int
	codes   map[int]struct{}
	keepMsg bool
}

func newFoldPolicy(cfg FoldConfig) foldPolicy {
	target := cfg.Target
	if target == 0 {
		target = int(extproto.Code__1_00_UNKNOWN)
	}
	codes := make(map[int]struct{}, len(cfg.Codes))
	for _, code := range cfg.Codes {
		codes[code] = struct{}{}
	}
	return foldPolicy{
		enabled: cfg.Enabled,
		target:  target,
		codes:   codes,
		keepMsg: cfg.KeepMsg,
	}
}

func (p foldPolicy) apply(code int, msg string) (int, string) {
	if !p.enabled {
		return code, msg
	}
	if code == p.target {
		return code, msg
	}
	if _, ok := p.codes[code]; !ok {
		return code, msg
	}
	if p.keepMsg && msg != "" {
		return p.target, msg
	}
	return p.target, errx.SystemBusyMessage
}

// 网关错误折叠策略。默认不配置，即统一透传；启动时由各网关按 etc 配置调用
// SetFoldConfig 覆盖。配置只影响 HTTP 信封，不影响 gRPC 传输。
var currentFold atomic.Pointer[foldPolicy]

// SetFoldConfig 设置当前网关的折叠策略。只在启动阶段调用。
func SetFoldConfig(cfg FoldConfig) {
	policy := newFoldPolicy(cfg)
	currentFold.Store(&policy)
}

func applyFold(code int, msg string) (int, string) {
	policy := currentFold.Load()
	if policy == nil {
		return code, msg
	}
	return policy.apply(code, msg)
}
