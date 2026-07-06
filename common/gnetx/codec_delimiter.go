package gnetx

import (
	"bytes"
	"context"

	"github.com/panjf2000/gnet/v2"
)

// DelimiterCodec 是基于字节分隔符的编解码器：一帧以指定分隔符结尾。
// 适用于以固定标记结尾的协议，例如 EB90 结束标志、0x00 分隔、或换行符 \n 分隔的文本协议。
// 序列化由 Serializer 承载，本 codec 只负责按分隔符切帧。
//
// 半包处理：若当前缓冲区里还没出现完整分隔符，返回 ErrIncompletePacket，
// 不消费任何字节，等下次可读事件累积更多数据后重试（分隔符本身也可能跨多次读取到达）。
//
// 性能说明：每次 Decode 会在"当前全部已缓冲字节"里 bytes.Index 查找分隔符。
// 对典型的小帧文本协议（每帧几十~几百字节）开销可忽略；若单帧极大且分多次到达，
// 会有 O(n²) 的重复扫描——这类大二进制帧建议改用 LengthPrefixCodec。
type DelimiterCodec struct {
	Serializer
	delimiter []byte
	strip     bool // true=Decode 返回的 raw 不含分隔符（默认）；false=保留分隔符
	maxSize   int  // 单帧最大字节数，0 表示不限
}

// DelimiterOption 配置 DelimiterCodec。
type DelimiterOption func(*DelimiterCodec)

// WithDelimiterStrip 设置是否从返回的 raw 中剥离分隔符。
// 默认 true（业务拿到的不含分隔符）；若协议解析需要分隔符本身（如校验结束标志），设 false。
func WithDelimiterStrip(strip bool) DelimiterOption {
	return func(c *DelimiterCodec) { c.strip = strip }
}

// WithDelimiterMaxSize 设置单帧最大字节数，超过返回 ErrFrameTooLarge。
// 防御一直不出现分隔符导致缓冲区无限增长。
func WithDelimiterMaxSize(max int) DelimiterOption {
	return func(c *DelimiterCodec) { c.maxSize = max }
}

// NewDelimiterCodec 构造分隔符编解码器。
// delimiter 不能为空（空分隔符会导致每次都在偏移 0 命中、切出空帧并清空缓冲，构造器直接 panic）；
// ser 为 payload 序列化器，不能为 nil。
func NewDelimiterCodec(delimiter []byte, ser Serializer, opts ...DelimiterOption) *DelimiterCodec {
	if len(delimiter) == 0 {
		panic("gnetx: NewDelimiterCodec delimiter must not be empty")
	}
	if ser == nil {
		panic("gnetx: NewDelimiterCodec serializer must not be nil")
	}
	c := &DelimiterCodec{
		Serializer: ser,
		delimiter:  delimiter,
		strip:      true,
		maxSize:    0,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Decode 从连接读一帧并交给 Serializer 转成消息。
// 分隔符可能跨多次可读事件到达，未找到分隔符时按半包处理（返回 ErrIncompletePacket）。
// 只用 Peek/Discard（on-loop 安全）。
func (c *DelimiterCodec) Decode(conn gnet.Conn, _ Conn) (any, error) {
	buffered := conn.InboundBuffered()
	if buffered == 0 {
		return nil, ErrIncompletePacket
	}
	peek, err := conn.Peek(buffered)
	if err != nil {
		return nil, mapShortBuffer(err)
	}
	idx := bytes.Index(peek, c.delimiter)
	if idx < 0 {
		// 尚无完整分隔符：超过上限直接判超大帧，否则当半包等更多数据。
		if c.maxSize > 0 && buffered > c.maxSize {
			return nil, ErrFrameTooLarge
		}
		return nil, ErrIncompletePacket
	}
	frameLen := idx + len(c.delimiter)
	if c.maxSize > 0 && frameLen > c.maxSize {
		return nil, ErrFrameTooLarge
	}
	// 拷出帧数据（peek 指向 gnet 内部 buffer，Discard 后失效）。
	var raw []byte
	if c.strip {
		raw = make([]byte, idx)
		copy(raw, peek[:idx])
	} else {
		raw = make([]byte, frameLen)
		copy(raw, peek[:frameLen])
	}
	if _, err := conn.Discard(frameLen); err != nil {
		return nil, mapShortBuffer(err)
	}
	return c.Serializer.Decode(raw)
}

// applyMaxFrameSizeIfUnset 实现 frameLimiter：仅当未显式设置 maxSize 时注入框架级上限。
func (c *DelimiterCodec) applyMaxFrameSizeIfUnset(max int) {
	if c.maxSize == 0 {
		c.maxSize = max
	}
}

// Encode 把消息序列化后追加分隔符形成完整帧。
// 注意：不会校验 payload 内部是否已含分隔符——若含，对端会提前切帧。协议设计上应保证
// payload 不含分隔符（或对分隔符做转义），这是分隔符类协议的固有约束。
func (c *DelimiterCodec) Encode(_ context.Context, msg any, _ Conn) ([]byte, error) {
	payload, err := c.Serializer.Encode(msg)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(payload)+len(c.delimiter))
	out = append(out, payload...)
	out = append(out, c.delimiter...)
	return out, nil
}
