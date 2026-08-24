package broadcast

import (
	"errors"
	"sync"

	"zero-service/common/antsx"
)

// errorKind 常量：广播 ack 的错误类别（SDK 只内置传输泛化类别，业务类别由 app 注册）。
const (
	// KindTimeout 表示等待应答超时（输出侧由 antsx.ErrReplyExpired 归一；输入侧还原为 antsx.ErrReplyExpired）。
	KindTimeout = "timeout"
	// KindDuplicate 表示重复的关联 ID（输出侧由 antsx.ErrDuplicateID 归一；输入侧还原为 antsx.ErrDuplicateID）。
	KindDuplicate = "duplicate"
	// KindUnknown 表示兜底错误类别（未注册错误按此处理）。
	KindUnknown = "unknown"
)

// ErrSkipAck 是执行器哨兵错误：executor 返回它表示业务处理成功但**不发送 ack**。
// 典型场景：非本节点业务对象（如任务不在本节点、设备未接入本节点的节点），
// 只有实际持有业务对象的节点回 ack，避免非 owner 节点造成虚假成功/失败。
var ErrSkipAck = errors.New("broadcast: skip ack reply")

// errorKindEntry 记录一份错误 ↔ kind 双向映射：src 用于输出侧归一，dst 用于输入侧还原。
type errorKindEntry struct {
	src error
	dst func(msg string) error
}

// errorKindRegistry 全局错误类别注册表（业务扩展注册，如 ieccaller 的 iec_rejected）。
type errorKindRegistry struct {
	mu    sync.RWMutex
	kinds map[string]errorKindEntry
}

var kindRegistry = errorKindRegistry{kinds: make(map[string]errorKindEntry)}

// RegisterErrorKind 注册一个业务错误类别（错误 ↔ kind 双向映射）。
//   - src: 输出侧判定错误（NormalizeErrorKind 用 errors.Is(err, src) 匹配）
//   - kind: 错误类别名（在 ack.ErrorKind 中传输）
//   - dst: 输入侧还原函数（ErrorFromKind 用 ack.Error 文本构造领域错误）
//
// 内置类别（timeout/duplicate/unknown）不需要注册；业务类别由 app 声明常量并注册
// （wire 值由业务自定义，本包不出现任何业务字符串）。同一 kind 重复注册时后者覆盖前者。
func RegisterErrorKind(src error, kind string, dst func(msg string) error) {
	if src == nil || kind == "" || dst == nil {
		return
	}
	kindRegistry.mu.Lock()
	defer kindRegistry.mu.Unlock()
	kindRegistry.kinds[kind] = errorKindEntry{src: src, dst: dst}
}

// NormalizeErrorKind 将错误归一为错误类别（输出侧：executor 失败 → ack.ErrorKind）。
// 内置规则：antsx.ErrReplyExpired → timeout；antsx.ErrDuplicateID → duplicate；
// 其余先匹配业务注册类别，未命中返回 KindUnknown。
func NormalizeErrorKind(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, antsx.ErrReplyExpired) {
		return KindTimeout
	}
	if errors.Is(err, antsx.ErrDuplicateID) {
		return KindDuplicate
	}
	kindRegistry.mu.RLock()
	defer kindRegistry.mu.RUnlock()
	for kind, entry := range kindRegistry.kinds {
		if errors.Is(err, entry.src) {
			return kind
		}
	}
	return KindUnknown
}

// ErrorFromKind 按 ack.ErrorKind 还原领域错误（输入侧：ack.Success=false → 调用方错误）。
// 内置规则：timeout → antsx.ErrReplyExpired；duplicate → antsx.ErrDuplicateID；
// 业务注册类别走 dst(msg)；未识别（含 unknown）按 errors.New(msg) 兜底。
func ErrorFromKind(kind, msg string) error {
	switch kind {
	case KindTimeout:
		return antsx.ErrReplyExpired
	case KindDuplicate:
		return antsx.ErrDuplicateID
	case "":
		// 容错：ack 未携带 errorKind 时按未知错误处理
		return errors.New(msg)
	}
	kindRegistry.mu.RLock()
	entry, ok := kindRegistry.kinds[kind]
	kindRegistry.mu.RUnlock()
	if ok && entry.dst != nil {
		return entry.dst(msg)
	}
	return errors.New(msg)
}
