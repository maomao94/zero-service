package gnetx

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"math"

	"github.com/panjf2000/gnet/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/common/tool"
)

// Codec 是 gnetx 的编解码契约，一个接口同时承载分帧与序列化（对齐 gnet v1 ICodec 的简洁形态）。
//
// Decode 在 OnTraffic（event-loop goroutine）中调用，只能用 gnet.Conn 的 Peek/Discard/
// InboundBuffered 等 on-loop 安全方法。半包必须返回 ErrIncompletePacket（框架据此停止本轮
// 解码并等待下次可读事件）；不可恢复错误（magic 错、超长等）返回其他非 nil error。
//
// Encode 把消息编码为完整帧字节（含分帧层加的长度/分隔符），可在 on-loop（同步回包 c.Write）
// 或 off-loop（业务 goroutine 主动 conn.WriteAsync）调用，不得读 conn。
//
// 内置 LengthPrefixCodec/DelimiterCodec/FixedLengthCodec 直接实现本接口，开箱即用；
// 用户自定义协议实现本接口即可，或用 NewFuncCodec 把两个闭包拼成 Codec。
type Codec interface {
	Decode(c gnet.Conn, conn Conn) (any, error)
	Encode(ctx context.Context, msg any, conn Conn) ([]byte, error)
}

// Serializer 承载 raw 帧字节与消息结构之间的转换，与分帧层解耦。
// 内置 Codec 把分帧（Peek/Discard）与 Serializer 组合，用户只需实现 Serializer 的序列化部分。
// 用户也可完全自定义 Codec，不用 Serializer。
type Serializer interface {
	Decode(raw []byte) (any, error)
	Encode(msg any) ([]byte, error)
}

// frameLimiter 由内置的、有"帧长度上限"语义的 Codec 实现（LengthPrefix/Delimiter）。
// NewServer/NewClient 会用 ServerOptions.MaxFrameLength 调用它，把框架级的强制帧长上限
// 注入到 codec（仅当 codec 自身未通过 WithMaxFrameSize 显式设置时）。
// 这样"MaxFrameLength 必填"这个开箱即用的安全约束才真正生效，防止损坏/错序流导致连接挂死。
// 自定义 Codec 若未实现本接口，需自行在 Decode 里限制帧长。
type frameLimiter interface {
	applyMaxFrameSizeIfUnset(max int)
}

// applyFrameLimit 把 max 注入到实现了 frameLimiter 的 codec（内置 LengthPrefix/Delimiter）。
func applyFrameLimit(c Codec, max int) {
	if fl, ok := c.(frameLimiter); ok && max > 0 {
		fl.applyMaxFrameSizeIfUnset(max)
	}
}

// NewFuncCodec 用两个函数构造 Codec，适合自定义协议一把梭（分帧+序列化一体）。
func NewFuncCodec(
	decode func(c gnet.Conn, conn Conn) (any, error),
	encode func(ctx context.Context, msg any, conn Conn) ([]byte, error),
) Codec {
	return &funcCodec{decode: decode, encode: encode}
}

type funcCodec struct {
	decode func(gnet.Conn, Conn) (any, error)
	encode func(context.Context, any, Conn) ([]byte, error)
}

func (c *funcCodec) Decode(gconn gnet.Conn, conn Conn) (any, error) {
	return c.decode(gconn, conn)
}

func (c *funcCodec) Encode(ctx context.Context, msg any, conn Conn) ([]byte, error) {
	return c.encode(ctx, msg, conn)
}

// DebugHexFormat controls how DebugSerializer renders payload bytes in logs.
type DebugHexFormat int

const (
	// HexLowerCompact renders bytes like hex.EncodeToString, for example "680e".
	HexLowerCompact DebugHexFormat = iota
	// HexUpperCompact renders compact upper-case hex, for example "680E".
	HexUpperCompact
	// HexUpperSpace renders upper-case bytes separated by spaces, for example "68 0E".
	HexUpperSpace
)

// DebugSerializerOption configures DebugSerializer logging behavior.
type DebugSerializerOption func(*debugSerializerOptions)

type debugSerializerOptions struct {
	hexFormat DebugHexFormat
}

// WithDebugHexFormat configures how DebugSerializer renders payload bytes in logs.
// The default is HexLowerCompact, matching the previous hex.EncodeToString output.
func WithDebugHexFormat(format DebugHexFormat) DebugSerializerOption {
	return func(opts *debugSerializerOptions) {
		opts.hexFormat = format
	}
}

// DebugSerializer 包装一个 Serializer，在 Debug 日志级别下打印每帧 payload 的 hex 编码。
// 默认格式保持为 hex.EncodeToString 的小写紧凑输出。
// 用法：codec := NewLengthPrefixCodec(2, endian, DebugSerializer(mySerializer))
// 如需协议抓包风格字节输出：DebugSerializer(mySerializer, WithDebugHexFormat(HexUpperSpace))。
func DebugSerializer(inner Serializer, opts ...DebugSerializerOption) Serializer {
	options := debugSerializerOptions{hexFormat: HexLowerCompact}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	return &debugSerializer{inner: inner, hexFormat: options.hexFormat}
}

type debugSerializer struct {
	inner     Serializer
	hexFormat DebugHexFormat
}

func (s *debugSerializer) Decode(raw []byte) (any, error) {
	logx.Debugf("[gnetx] recv %d bytes hex=%s", len(raw), formatDebugHex(raw, s.hexFormat))
	return s.inner.Decode(raw)
}

func (s *debugSerializer) Encode(msg any) ([]byte, error) {
	raw, err := s.inner.Encode(msg)
	if err != nil {
		return raw, err
	}
	logx.Debugf("[gnetx] send %d bytes hex=%s", len(raw), formatDebugHex(raw, s.hexFormat))
	return raw, nil
}

func formatDebugHex(raw []byte, format DebugHexFormat) string {
	switch format {
	case HexUpperCompact:
		return tool.HexBytes(raw, tool.HexUpperCompact)
	case HexUpperSpace:
		return tool.HexBytes(raw, tool.HexUpperSpace)
	default:
		return tool.HexBytes(raw, tool.HexLowerCompact)
	}
}

// mapShortBuffer 把 gnet Peek/Discard 返回的 io.ErrShortBuffer 统一映射为 ErrIncompletePacket。
// 其他错误原样返回。内置 Codec 实现应调用此函数规范化半包错误。
func mapShortBuffer(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, io.ErrShortBuffer) {
		return ErrIncompletePacket
	}
	return err
}

// validLengthFieldSize 报告 n 是否是受支持的长度字段字节数（1/2/4/8）。
// 用于在 codec 构造时提前校验，避免 readUintN/putUintN 在 event-loop 上运行时 panic。
func validLengthFieldSize(n int) bool {
	return n == 1 || n == 2 || n == 4 || n == 8
}

// maxLenForBytes 返回 n 字节无符号整数能表示的最大值。
// n>=8 时返回 uint64 上限（int 值恒可容纳，无需再校验）。
func maxLenForBytes(n int) uint64 {
	if n >= 8 {
		return math.MaxUint64
	}
	return (uint64(1) << (8 * n)) - 1
}

// readUintN 从 b 开头读取 n 字节的无符号整数，endianness 控制字节序。
// n 仅支持 1/2/4/8；其他值 panic（构造时已用 validLengthFieldSize 拦截，不会到这里）。
func readUintN(b []byte, n int, endianness binary.ByteOrder) uint64 {
	switch n {
	case 1:
		return uint64(b[0])
	case 2:
		return uint64(endianness.Uint16(b))
	case 4:
		return uint64(endianness.Uint32(b))
	case 8:
		return endianness.Uint64(b)
	default:
		panic("gnetx: unsupported length field size, must be 1/2/4/8")
	}
}

// putUintN 把 v 按 n 字节写入 b 开头。
// n 仅支持 1/2/4/8；调用方须保证 v 不超过 n 字节容量（Encode 已用 maxLenForBytes 校验），
// 否则会发生截断（如 uint16(70000)=4464），导致帧长字段写错、对端错帧。
func putUintN(b []byte, n int, v uint64, endianness binary.ByteOrder) {
	switch n {
	case 1:
		b[0] = byte(v)
	case 2:
		endianness.PutUint16(b, uint16(v))
	case 4:
		endianness.PutUint32(b, uint32(v))
	case 8:
		endianness.PutUint64(b, v)
	default:
		panic("gnetx: unsupported length field size, must be 1/2/4/8")
	}
}
