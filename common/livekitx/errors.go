package livekitx

import (
	"errors"
	"fmt"
)

// ErrNilRoom 表示实时连接句柄或其底层 Room 为空。
var ErrNilRoom = errors.New("livekitx: nil realtime room")

// HookPanicError 表示用户 Hook panic；panic 不会穿透 SDK 读循环或 HTTP 边界。
type HookPanicError struct {
	Event string
	Value any
}

// Error 返回包含事件类型和 panic 值的错误描述。
func (e *HookPanicError) Error() string {
	return fmt.Sprintf("livekitx: %s hook panic: %v", e.Event, e.Value)
}
