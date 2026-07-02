package gnetx

import (
	"encoding/binary"
	"fmt"

	"github.com/panjf2000/gnet/v2"
)

// LengthPrefixCodec 是基于长度前缀的编解码器，帧格式：
//
//	[lengthOffset 字节可选前缀][lengthBytes 字节长度字段][payload]
//
// 这是 TCP 自定义二进制协议最常用的分帧方式，能正确处理半包（一帧分多次到达）
// 与粘包（多帧在一次可读事件到达）。序列化由 Serializer 承载，本 codec 只负责分帧。
//
// 参数语义：
//   - lengthBytes：长度字段占几字节，支持 1/2/4/8（uint8/16/32/64）。
//   - bo：长度字段的字节序（binary.BigEndian / binary.LittleEndian）。
//   - lengthOffset：长度字段之前要跳过的字节数。用于协议在长度字段前还有 magic/version
//     等固定头的场景。注意：内置 Encode 会把这段偏移填 0；若前缀有实际内容，请自定义 Codec。
//   - lengthAdjust：长度字段的“值”与 payload 实际字节数的差值，用于兼容不同协议对“长度”
//     的定义。Decode 时 payloadLen = 字段值 + lengthAdjust；Encode 时 字段值 = len(payload) - lengthAdjust。
//     举例：
//   - 字段值 = payload 字节数         → lengthAdjust = 0（最常见）
//   - 字段值 = 整帧字节数（含头）      → lengthAdjust = -(lengthOffset + lengthBytes)
//   - 字段值 = payload + 2 字节 CRC   → lengthAdjust = 0 且把 CRC 视为 payload 的一部分，或用 -2
//   - maxFrameSize：单帧最大字节数（含头）。0 表示不限（不推荐）。NewServer/NewClient 会在
//     未显式设置时用 MaxFrameLength 注入，防止损坏/错序流报出超大长度把连接读挂。
//
// 构造用 NewLengthPrefixCodec，不要直接实例化（构造器会校验 lengthBytes 合法性）。
type LengthPrefixCodec struct {
	Serializer
	lengthBytes  int
	bo           binary.ByteOrder // 字节序
	lengthOffset int              // 长度字段前需跳过的字节数（如 magic 头）
	lengthAdjust int              // 长度字段值与 payload 实际长度的差值调整
	maxFrameSize int              // 单帧最大字节数（含帧头），0 表示不限
}

// LengthPrefixOption 配置 LengthPrefixCodec。
type LengthPrefixOption func(*LengthPrefixCodec)

// WithLengthOffset 设置长度字段前的跳过字节数（例如协议有 magic 头时）。必须 >= 0。
func WithLengthOffset(offset int) LengthPrefixOption {
	return func(c *LengthPrefixCodec) { c.lengthOffset = offset }
}

// WithLengthAdjust 设置长度字段值与 payload 实际长度的差值（见 LengthPrefixCodec 文档举例）。
func WithLengthAdjust(adjust int) LengthPrefixOption {
	return func(c *LengthPrefixCodec) { c.lengthAdjust = adjust }
}

// WithMaxFrameSize 设置单帧最大字节数（含帧头），超过返回 ErrFrameTooLarge。
// 强烈建议设置以防御恶意/异常大帧。0 表示不限制（不推荐）。
func WithMaxFrameSize(max int) LengthPrefixOption {
	return func(c *LengthPrefixCodec) { c.maxFrameSize = max }
}

// NewLengthPrefixCodec 构造长度前缀编解码器。
// lengthBytes 必须是 1/2/4/8，否则 panic（尽早暴露配置错误，而非在 event-loop 上运行时 panic）；
// endianness 用 binary.BigEndian 或 binary.LittleEndian；
// ser 承载 payload 的序列化（可用 RawSerializer/JSONSerializer 或自定义）。
func NewLengthPrefixCodec(lengthBytes int, endianness binary.ByteOrder, ser Serializer, opts ...LengthPrefixOption) *LengthPrefixCodec {
	if !validLengthFieldSize(lengthBytes) {
		panic("gnetx: NewLengthPrefixCodec lengthBytes must be 1/2/4/8")
	}
	if endianness == nil {
		panic("gnetx: NewLengthPrefixCodec endianness must not be nil")
	}
	if ser == nil {
		panic("gnetx: NewLengthPrefixCodec serializer must not be nil")
	}
	c := &LengthPrefixCodec{
		Serializer:   ser,
		lengthBytes:  lengthBytes,
		bo:           endianness,
		lengthOffset: 0,
		lengthAdjust: 0,
		maxFrameSize: 0,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Decode 从连接读一帧 payload 并交给 Serializer 转成消息。半包返回 ErrIncompletePacket。
//
// 流程：先 Peek 帧头得到 payload 长度，校验长度合法性与上限，再 Peek 整帧、拷出 payload、
// Discard 消费。两次 Peek 之间不 Discard，故半包时不消费任何字节，下次可读事件从头重试。
// 只用 Peek/Discard（on-loop 安全），不持有 gnet 内部 buffer 切片跨调用。
func (c *LengthPrefixCodec) Decode(conn gnet.Conn, sess *Session) (any, error) {
	headerLen := c.lengthOffset + c.lengthBytes

	hdr, err := conn.Peek(headerLen)
	if err != nil {
		return nil, mapShortBuffer(err)
	}
	lengthField := hdr[c.lengthOffset : c.lengthOffset+c.lengthBytes]
	payloadLen := int(readUintN(lengthField, c.lengthBytes, c.bo)) + c.lengthAdjust
	// 校验顺序很重要：先查 payloadLen 合法性（负数可能来自 lengthAdjust 或 8 字节长度溢出 int），
	// 再算 frameLen 并防溢出，最后比对 maxFrameSize；避免用非法长度去 Peek 把连接读挂。
	if payloadLen < 0 {
		return nil, ErrFrameTooLarge
	}
	frameLen := headerLen + payloadLen
	if frameLen < 0 { // int 溢出
		return nil, ErrFrameTooLarge
	}
	if c.maxFrameSize > 0 && frameLen > c.maxFrameSize {
		return nil, ErrFrameTooLarge
	}

	buf, err := conn.Peek(frameLen)
	if err != nil {
		return nil, mapShortBuffer(err)
	}
	// 拷出 payload：buf 指向 gnet 内部 buffer，Discard 后即失效，必须 copy 出来交给业务。
	payload := make([]byte, payloadLen)
	copy(payload, buf[headerLen:frameLen])
	if _, err := conn.Discard(frameLen); err != nil {
		return nil, mapShortBuffer(err)
	}
	return c.Serializer.Decode(payload, sess)
}

// applyMaxFrameSizeIfUnset 实现 frameLimiter：仅当未显式设置 maxFrameSize 时注入框架级上限。
func (c *LengthPrefixCodec) applyMaxFrameSizeIfUnset(max int) {
	if c.maxFrameSize == 0 {
		c.maxFrameSize = max
	}
}

// Encode 把消息序列化后加长度前缀封装为完整帧。
// 若 payload 长度超过长度字段能表示的范围（如 2 字节字段 payload>65535），返回错误而非
// 静默截断——截断会导致对端错帧、数据损坏。
func (c *LengthPrefixCodec) Encode(msg any, sess *Session) ([]byte, error) {
	payload, err := c.Serializer.Encode(msg, sess)
	if err != nil {
		return nil, err
	}
	fieldVal := len(payload) - c.lengthAdjust
	if fieldVal < 0 {
		return nil, fmt.Errorf("gnetx: negative length field value %d (payload=%d, adjust=%d)",
			fieldVal, len(payload), c.lengthAdjust)
	}
	if uint64(fieldVal) > maxLenForBytes(c.lengthBytes) {
		return nil, fmt.Errorf("gnetx: length field value %d exceeds %d-byte capacity",
			fieldVal, c.lengthBytes)
	}
	headerLen := c.lengthOffset + c.lengthBytes
	out := make([]byte, headerLen+len(payload))
	// 偏移区域留零（用户若有 magic 头需求可自定义 Codec）。
	putUintN(out[c.lengthOffset:], c.lengthBytes, uint64(fieldVal), c.bo)
	copy(out[headerLen:], payload)
	return out, nil
}
