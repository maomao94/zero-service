package gnetx

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/panjf2000/gnet/v2"
)

// LengthPrefixCodec 是基于长度前缀的编解码器（对标 Netty LengthFieldBasedFrameDecoder +
// LengthFieldPrepender）。帧布局：
//
//	[0..stripBytes][serializer 产出的字节（含可选帧头字段）][trailingBytes]
//
// 其中长度字段位于 [lengthOffset, lengthOffset+lengthBytes)，由 codec 负责读写；
// Serializer 产出的字节从 stripBytes 开始，至帧尾（不含 trailingBytes）结束。
//
// 覆盖常见 TCP 二进制协议：
//
//	简单长度前缀: [2B len][payload]
//	  NewLengthPrefixCodec(2, binary.BigEndian, ser)
//
//	前缀+长度+包体: EB(2) + len(2) + 包体
//	  NewLengthPrefixCodec(2, endian, ser, WithLeadingBytes([]byte{0xEB, 0xEB}))
//
//	前缀+长度+包体+尾缀: EB(2) + len(2) + 包体 + EB(2)
//	  NewLengthPrefixCodec(2, endian, ser,
//	    WithLeadingBytes([]byte{0xEB, 0xEB}),
//	    WithTrailingBytes([]byte{0xEB, 0xEB}),
//	  )
//
//	帧头含业务字段: [msgID 2B][len 2B][payload]
//	  NewLengthPrefixCodec(2, endian, ser,
//	    WithLengthOffset(2),   // 长度字段在 byte 2
//	    WithStripBytes(0),     // 不剥离帧头，Serializer 拿到完整帧（含 msgID）
//	  )
//
// 参数语义：
//   - lengthBytes：长度字段占几字节，支持 1/2/4/8。
//   - endianness：长度字段的字节序（binary.BigEndian / binary.LittleEndian）。
//   - lengthOffset：长度字段距帧首的偏移字节数。若设置了 leadingBytes 且未显式调用本方法，
//     自动取 len(leadingBytes)。
//   - lengthAdjust：长度字段值与 framePayloadLen 的差值。
//     Decode: framePayloadLen = 字段值 + lengthAdjust（framePayloadLen = 长度字段后的字节数）
//     Encode: 字段值 = framePayloadLen - lengthAdjust
//     举例：
//   - 字段值 = postLengthBytes 字节数    → lengthAdjust = 0（最常见）
//   - 字段值 = 整帧字节数（含头）         → lengthAdjust = -(lengthOffset + lengthBytes)
//   - 字段值不含尾缀                      → lengthAdjust = len(trailingBytes)
//   - stripBytes：解码后从帧首剥离的字节数（对标 Netty initialBytesToStrip）。
//     不调用时默认剥离整个帧头（lengthOffset + lengthBytes）。
//     Serializer 仅接收帧中 [stripBytes, frameLen-len(trailingBytes)] 区间的字节。
//     设为 0 时 Serializer 拿到完整帧（含长度字段及之前的所有字节）。
//   - maxFrameSize：单帧最大字节数（含帧头）。0 表示不限。
//   - leadingBytes：帧起始固定字节（如 magic 0xEB90）。Decode 校验，Encode 写入。
//   - trailingBytes：帧末尾固定字节（如结束标志）。Decode 校验后剥离，Encode 追加。
//     长度字段默认统计 trailingBytes。
//
// 构造用 NewLengthPrefixCodec，不要直接实例化（构造器会校验参数）。
type LengthPrefixCodec struct {
	Serializer
	lengthBytes  int              // 长度字段字节数，仅允许 1/2/4/8。
	bo           binary.ByteOrder // 长度字段字节序，构造时必填。
	lengthOffset int              // 长度字段距帧首的偏移，必须 >= 0；未显式设置且有 leadingBytes 时默认 len(leadingBytes)。
	lengthAdjust int              // Decode: bodyLen = lengthField + lengthAdjust；Encode: lengthField = bodyLen - lengthAdjust。
	maxFrameSize int              // 单帧最大字节数（含帧头），0 表示不限；必须 >= 0。

	stripBytes    int    // Decode 后从帧首剥离的字节数，默认 headerLen；必须满足 0 <= stripBytes <= headerLen。
	leadingBytes  []byte // 帧首固定字节；必须完全位于 lengthOffset 前，避免覆盖长度字段。
	trailingBytes []byte // 帧尾固定字节；Decode 校验后剥离，Encode 自动追加。

	offsetExplicit bool // WithLengthOffset 是否被显式调用；用于决定 leadingBytes 是否自动设置 lengthOffset。
	stripExplicit  bool // WithStripBytes 是否被显式调用；未设置时 stripBytes 默认取 headerLen。
}

// LengthPrefixOption 配置 LengthPrefixCodec。
type LengthPrefixOption func(*LengthPrefixCodec)

// WithLengthOffset 显式设置长度字段距帧首的偏移字节数。
// 若同时设置了 leadingBytes 且未调用本方法，lengthOffset 自动取 len(leadingBytes)。
func WithLengthOffset(offset int) LengthPrefixOption {
	return func(c *LengthPrefixCodec) {
		c.lengthOffset = offset
		c.offsetExplicit = true
	}
}

// WithLengthAdjust 设置长度字段值与 framePayload 实际长度的差值（见 LengthPrefixCodec 文档举例）。
func WithLengthAdjust(adjust int) LengthPrefixOption {
	return func(c *LengthPrefixCodec) { c.lengthAdjust = adjust }
}

// WithMaxFrameSize 设置单帧最大字节数（含帧头），超过返回 ErrFrameTooLarge。
// 强烈建议设置以防御恶意/异常大帧。0 表示不限制（不推荐）。
func WithMaxFrameSize(max int) LengthPrefixOption {
	return func(c *LengthPrefixCodec) { c.maxFrameSize = max }
}

// WithLeadingBytes 设置帧起始固定字节（如 magic 0xEB90）。Decode 时校验前缀匹配，
// Encode 时写入前缀填补 offset 区域。自动将 lengthOffset 对齐为 len(b)（除非同时显式调用 WithLengthOffset）。
func WithLeadingBytes(b []byte) LengthPrefixOption {
	return func(c *LengthPrefixCodec) { c.leadingBytes = append([]byte(nil), b...) }
}

// WithTrailingBytes 设置帧末尾固定字节（如结束标志 0xEB90）。Decode 时校验后从 payload 剥离，
// Encode 时追加到 payload 末尾。长度字段默认统计 trailingBytes（若需排除，用 WithLengthAdjust）。
func WithTrailingBytes(b []byte) LengthPrefixOption {
	return func(c *LengthPrefixCodec) { c.trailingBytes = append([]byte(nil), b...) }
}

// WithStripBytes 设置解码后从帧首剥离的字节数（对标 Netty initialBytesToStrip）。
// 不调用时默认剥离整个帧头（lengthOffset + lengthBytes）。
// 设为 0 时 Serializer 拿到完整帧；设为 lengthOffset 时保留长度字段及 payload。
func WithStripBytes(n int) LengthPrefixOption {
	return func(c *LengthPrefixCodec) {
		c.stripBytes = n
		c.stripExplicit = true
	}
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
	if len(c.leadingBytes) > 0 && !c.offsetExplicit {
		c.lengthOffset = len(c.leadingBytes)
	}
	if c.lengthOffset < 0 {
		panic("gnetx: NewLengthPrefixCodec lengthOffset must be non-negative")
	}
	if c.maxFrameSize < 0 {
		panic("gnetx: NewLengthPrefixCodec maxFrameSize must be non-negative")
	}
	if len(c.leadingBytes) > c.lengthOffset {
		panic("gnetx: NewLengthPrefixCodec leadingBytes length exceeds lengthOffset, would overlap length field")
	}
	if !c.stripExplicit {
		c.stripBytes = c.lengthOffset + c.lengthBytes
	}
	if c.stripBytes < 0 {
		panic("gnetx: NewLengthPrefixCodec stripBytes must be non-negative")
	}
	if c.stripBytes > c.lengthOffset+c.lengthBytes {
		panic("gnetx: NewLengthPrefixCodec stripBytes exceeds header length")
	}
	return c
}

// Decode 从连接读一帧并交给 Serializer 转成消息。半包返回 ErrIncompletePacket。
//
// 流程：校验 leadingBytes → Peek 帧头得 bodyLen → 校验合法性 → Peek 整帧 →
// 校验 trailingBytes → 从 stripBytes 位置裁出数据交给 Serializer → Discard 消费。
func (c *LengthPrefixCodec) Decode(conn gnet.Conn, _ Conn) (any, error) {
	headerLen := c.lengthOffset + c.lengthBytes

	hdr, err := conn.Peek(headerLen)
	if err != nil {
		return nil, mapShortBuffer(err)
	}
	if len(c.leadingBytes) > 0 && !bytes.Equal(hdr[:len(c.leadingBytes)], c.leadingBytes) {
		return nil, fmt.Errorf("gnetx: frame prefix mismatch, expected %x got %x",
			c.leadingBytes, hdr[:len(c.leadingBytes)])
	}

	lengthField := hdr[c.lengthOffset : c.lengthOffset+c.lengthBytes]
	rawBodyLen := readUintN(lengthField, c.lengthBytes, c.bo)
	if rawBodyLen > uint64(math.MaxInt) {
		return nil, ErrFrameTooLarge
	}
	bodyLen := int(rawBodyLen)
	if c.lengthAdjust > 0 && bodyLen > math.MaxInt-c.lengthAdjust {
		return nil, ErrFrameTooLarge
	}
	bodyLen += c.lengthAdjust
	if bodyLen < 0 {
		return nil, ErrFrameTooLarge
	}
	if bodyLen > math.MaxInt-headerLen {
		return nil, ErrFrameTooLarge
	}
	frameLen := headerLen + bodyLen
	if c.maxFrameSize > 0 && frameLen > c.maxFrameSize {
		return nil, ErrFrameTooLarge
	}

	buf, err := conn.Peek(frameLen)
	if err != nil {
		return nil, mapShortBuffer(err)
	}
	if len(c.trailingBytes) > 0 {
		if bodyLen < len(c.trailingBytes) {
			return nil, fmt.Errorf("gnetx: frame payload too short for trailing bytes (%d < %d)",
				bodyLen, len(c.trailingBytes))
		}
		trailStart := frameLen - len(c.trailingBytes)
		if !bytes.Equal(buf[trailStart:frameLen], c.trailingBytes) {
			return nil, fmt.Errorf("gnetx: frame suffix mismatch, expected %x got %x",
				c.trailingBytes, buf[trailStart:frameLen])
		}
	}

	// Deliver [stripBytes, frameLen-len(trailing)) to Serializer.
	// stripBytes defaults to headerLen (strip full header); 0 means keep everything.
	serializedLen := frameLen - c.stripBytes - len(c.trailingBytes)
	if serializedLen < 0 {
		return nil, fmt.Errorf("gnetx: negative serialized payload length %d", serializedLen)
	}
	serializedPayload := make([]byte, serializedLen)
	copy(serializedPayload, buf[c.stripBytes:frameLen-len(c.trailingBytes)])

	if _, err := conn.Discard(frameLen); err != nil {
		return nil, mapShortBuffer(err)
	}
	return c.Serializer.Decode(serializedPayload)
}

// applyMaxFrameSizeIfUnset 实现 frameLimiter：仅当未显式设置 maxFrameSize 时注入框架级上限。
func (c *LengthPrefixCodec) applyMaxFrameSizeIfUnset(max int) {
	if c.maxFrameSize == 0 {
		c.maxFrameSize = max
	}
}

// Encode 把消息序列化后封装为完整帧。
//
// Serializer 产出的字节从帧的 stripBytes 位置开始放置（默认跳过整个帧头，直接写 body）。
// 长度字段由 codec 在 [lengthOffset, lengthOffset+lengthBytes) 位置写入；
// Serializer 产出中对应的字节被跳过（不会重复写入）。
// trailingBytes 追加到帧尾。
func (c *LengthPrefixCodec) Encode(_ context.Context, msg any, _ Conn) ([]byte, error) {
	payload, err := c.Serializer.Encode(msg)
	if err != nil {
		return nil, err
	}

	headerLen := c.lengthOffset + c.lengthBytes
	if c.lengthOffset < 0 || c.stripBytes < 0 || c.stripBytes > headerLen {
		return nil, fmt.Errorf("gnetx: invalid length-prefix codec configuration")
	}
	preLen := 0
	if c.lengthOffset > c.stripBytes {
		preLen = c.lengthOffset - c.stripBytes
	}
	var skipLen int
	if c.stripBytes <= c.lengthOffset {
		skipLen = c.lengthBytes
	}
	postLen := len(payload) - preLen - skipLen
	if postLen < 0 {
		return nil, fmt.Errorf("gnetx: serializer payload too short for pre+skip (%d < %d)",
			len(payload), preLen+skipLen)
	}

	bodyLen := postLen + len(c.trailingBytes)
	fieldVal := bodyLen - c.lengthAdjust
	if fieldVal < 0 {
		return nil, fmt.Errorf("gnetx: negative length field value %d (bodyLen=%d, adjust=%d)",
			fieldVal, bodyLen, c.lengthAdjust)
	}
	if uint64(fieldVal) > maxLenForBytes(c.lengthBytes) {
		return nil, fmt.Errorf("gnetx: length field value %d exceeds %d-byte capacity",
			fieldVal, c.lengthBytes)
	}

	out := make([]byte, headerLen+bodyLen)
	if len(c.leadingBytes) > 0 {
		copy(out, c.leadingBytes)
	}
	if preLen > 0 {
		copy(out[c.stripBytes:c.lengthOffset], payload[:preLen])
	}
	putUintN(out[c.lengthOffset:], c.lengthBytes, uint64(fieldVal), c.bo)
	copy(out[headerLen:], payload[preLen+skipLen:])
	if len(c.trailingBytes) > 0 {
		copy(out[headerLen+postLen:], c.trailingBytes)
	}
	return out, nil
}
