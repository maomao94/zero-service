package antsx

import (
	"errors"
	"fmt"
	"runtime/debug"
)

// 包级别哨兵错误，用于在不同模块中统一标识特定异常场景。
// 调用方应使用 errors.Is 进行判断。
var (
	// ErrReplyExpired 表示 ReplyPool 中的条目已超时过期。
	ErrReplyExpired = errors.New("antsx: reply pool entry expired")

	// ErrDuplicateID 表示 ReplyPool 中注册了重复的关联 ID。
	ErrDuplicateID = errors.New("antsx: duplicate id")

	// ErrReplyClosed 表示 ReplyPool 已关闭，无法注册新的条目。
	ErrReplyClosed = errors.New("antsx: reply pool closed")

	// ErrChanClosed 表示向已关闭的 UnboundedChan 发送数据。
	ErrChanClosed = errors.New("antsx: send on closed channel")

	// ErrNoValue 是 StreamReaderWithConvert 中的哨兵错误，
	// convert 函数返回此错误表示跳过当前元素，继续读取下一个。
	ErrNoValue = errors.New("antsx: no value")

	// ErrRecvAfterClosed 表示在 Copy 产生的 childReader 关闭后仍尝试 Recv。
	ErrRecvAfterClosed = errors.New("antsx: recv after stream closed")

	// ErrEmptyPromises 表示 PromiseRace/PromiseAny 收到了空的 promises 切片。
	ErrEmptyPromises = errors.New("antsx: empty promises")
)

// panicErr 将 goroutine 中的 panic 信息包装为标准 error。
// 内部携带 panic 值和捕获时的调用栈，便于排查问题。
type panicErr struct {
	info  any
	stack []byte
}

// newPanicErr 创建一个包含当前调用栈的 panicErr 实例。
func newPanicErr(info any) *panicErr {
	return &panicErr{info: info, stack: debug.Stack()}
}

// Error 返回 panic 信息及完整调用栈的格式化字符串。
func (e *panicErr) Error() string {
	return fmt.Sprintf("antsx: panic: %v\n%s", e.info, e.stack)
}

// Unwrap 尝试将 panic 值解包为 error。
// 如果 panic 抛出的本身是一个 error，则返回该 error 以支持 errors.Is/As 判断。
func (e *panicErr) Unwrap() error {
	if err, ok := e.info.(error); ok {
		return err
	}
	return nil
}

// SourceEOF 表示 MergeNamedStreamReaders 中某条具名源流已到达 EOF。
// 调用方可通过 GetSourceName 提取源流名称，据此判断哪条子流已结束。
type SourceEOF struct {
	source string
}

// Error 返回带源流名称的 EOF 描述字符串。
func (e *SourceEOF) Error() string {
	return fmt.Sprintf("antsx: source %q reached EOF", e.source)
}

// GetSourceName 尝试从 err 中提取 SourceEOF 携带的源流名称。
// 返回 (name, true) 表示 err 是 SourceEOF；否则返回 ("", false)。
func GetSourceName(err error) (string, bool) {
	var se *SourceEOF
	if errors.As(err, &se) {
		return se.source, true
	}
	return "", false
}
