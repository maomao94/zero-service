package gnetx

import "errors"

// 包级别哨兵错误，调用方应使用 errors.Is 判断。
var (
	// ErrIncompletePacket 表示当前接收缓冲区不足以凑成一帧，属于半包。
	// OnTraffic 收到此错误时应停止本轮解码，剩余字节留待下次可读事件。
	ErrIncompletePacket = errors.New("gnetx: incomplete packet")

	// ErrFrameTooLarge 表示帧长度超过配置的最大值，属不可恢复错误。
	ErrFrameTooLarge = errors.New("gnetx: frame too large")

	// ErrSessionClosed 表示会话已关闭，无法再发送或接收。
	ErrSessionClosed = errors.New("gnetx: session closed")

	// ErrInvalidClientID 表示客户端业务身份为空，无法绑定到会话。
	ErrInvalidClientID = errors.New("gnetx: invalid client id")

	// ErrNoHandler 表示 Router 未找到匹配 messageID 的 handler，且未配置 fallback。
	ErrNoHandler = errors.New("gnetx: no handler for message id")

	// ErrPendingNotFound 表示入站回包未能匹配到在途请求（无人等待或已被清理）。
	ErrPendingNotFound = errors.New("gnetx: pending request not found")

	// errRawSerializerType 表示 RawSerializer 收到了非 []byte 的消息类型。
	errRawSerializerType = errors.New("gnetx: RawSerializer only accepts []byte")
)
