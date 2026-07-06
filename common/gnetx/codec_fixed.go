package gnetx

import (
	"context"

	"github.com/panjf2000/gnet/v2"
)

// FixedLengthCodec 是定长编解码器：每帧字节数固定不变，无长度字段、无分隔符。
// 适用于定长报文协议——例如每帧固定 16 字节的心跳、固定结构的传感器上报等。
// 序列化由 Serializer 承载，本 codec 只负责按固定长度切帧。
//
// 半包处理：缓冲区不足一帧长度时返回 ErrIncompletePacket，不消费字节，等更多数据。
type FixedLengthCodec struct {
	Serializer
	length int
}

// NewFixedLengthCodec 构造定长编解码器。length 必须 > 0，否则 panic（尽早暴露配置错误）。
// ser 为 payload 序列化器，不能为 nil。
func NewFixedLengthCodec(length int, ser Serializer) *FixedLengthCodec {
	if length <= 0 {
		panic("gnetx: NewFixedLengthCodec length must be positive")
	}
	if ser == nil {
		panic("gnetx: NewFixedLengthCodec serializer must not be nil")
	}
	return &FixedLengthCodec{Serializer: ser, length: length}
}

// Decode 从连接读一帧固定长度字节并交给 Serializer 转成消息。只用 Peek/Discard（on-loop 安全）。
func (c *FixedLengthCodec) Decode(conn gnet.Conn, _ Conn) (any, error) {
	buf, err := conn.Peek(c.length)
	if err != nil {
		return nil, mapShortBuffer(err)
	}
	// 拷出（buf 指向 gnet 内部 buffer，Discard 后失效）。
	raw := make([]byte, c.length)
	copy(raw, buf)
	if _, err := conn.Discard(c.length); err != nil {
		return nil, mapShortBuffer(err)
	}
	return c.Serializer.Decode(raw)
}

// Encode 把消息序列化为字节直接返回（定长协议无需加帧头）。
// 由用户/Serializer 保证产出长度符合固定长度约定；本 codec 不强制校验长度，
// 以便某些协议在 payload 里自带定长布局。
func (c *FixedLengthCodec) Encode(_ context.Context, msg any, _ Conn) ([]byte, error) {
	return c.Serializer.Encode(msg)
}
