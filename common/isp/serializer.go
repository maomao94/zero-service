package isp

import (
	"context"
	"encoding/binary"
	"fmt"
	"html"

	"zero-service/common/gnetx"

	"github.com/zeromicro/go-zero/core/logx"
)

// ISP 帧序列化常量。
// 帧结构：0xEB90(2B BE) + SendSeq(8B LE) + RecvSeq(8B LE) + SessionSource(1B)
//   - XMLLength(4B LE) + XML + 0xEB90(2B BE)
//
// SendSeq=sendSerialNo(本端自增), RecvSeq=receiveSerialNo(对端回执=上次收到的对端SendSeq)
// LengthPrefixCodec 处理头尾标志和长度字段；Serializer 只处理中间 21 字节头 + XML。
const (
	serializerHeaderLen = 21 // SendSeq(8) + RecvSeq(8) + SessionSource(1) + XMLLength(4)
	lengthOffset        = 19 // XMLLength 在完整帧中的偏移（含前导 0xEB90 的 2 字节）
	stripBytes          = 2  // 只剥前导 0xEB90，剩余 21 字节头交给 Serializer
)

// Serializer 实现 gnetx.Serializer，负责 ISP 帧 payload（21B 二进制头 + XML 体）与 Message 的互转。
type Serializer struct {
	RootName string // 默认根元素名称
}

func NewSerializer(rootName string) Serializer {
	return Serializer{RootName: NormalizeRootName(rootName)}
}

// Decode 解码 payload 为 Message。payload 布局：
//
//	[0:8]   SendSeq      小端（对端发送序号）
//	[8:16]  RecvSeq      小端（对端回执 = 上次收到的本端 SendSeq）
//	[16]    SessionSource
//	[17:21] XMLLength    小端（由 Codec 已校验并写入，此处做一致性校验）
//	[21:]   XML Body
func (s Serializer) Decode(raw []byte) (any, error) {
	if len(raw) < serializerHeaderLen {
		return nil, fmt.Errorf("isp: payload 太短: %d 字节", len(raw))
	}
	xmlLen := binary.LittleEndian.Uint32(raw[17:21])
	if int(xmlLen) != len(raw)-serializerHeaderLen {
		return nil, fmt.Errorf("isp: XML 长度不一致: 声明=%d 实际=%d", xmlLen, len(raw)-serializerHeaderLen)
	}
	msg, err := ParseXML(raw[serializerHeaderLen:])
	if err != nil {
		return nil, err
	}
	msg.SendSeq = binary.LittleEndian.Uint64(raw[0:8])
	msg.RecvSeq = binary.LittleEndian.Uint64(raw[8:16])
	msg.SessionSource = raw[16]
	if msg.RootName == "" {
		msg.RootName = s.RootName
	}
	return msg, nil
}

// Encode 将 Message 编码为 payload（21B 头 + XML）。
// XMLLength 字段由 LengthPrefixCodec 在帧级 Encode 时根据 bodyLen + trailingBytes 重新计算并覆盖，
// 因此此处写入的 XMLLength 只需与 XML 实际长度一致即可。
func (s Serializer) Encode(v any) ([]byte, error) {
	msg, ok := v.(*Message)
	if !ok {
		return nil, fmt.Errorf("isp: 不支持的消息类型 %T", v)
	}
	msg.EnsureDefaults(s.RootName)
	xmlBody, err := BuildXML(msg, s.RootName)
	if err != nil {
		return nil, err
	}
	out := make([]byte, serializerHeaderLen+len(xmlBody))
	// 以下均为小端字节序
	binary.LittleEndian.PutUint64(out[0:8], msg.SendSeq)
	binary.LittleEndian.PutUint64(out[8:16], msg.RecvSeq)
	out[16] = msg.SessionSource
	binary.LittleEndian.PutUint32(out[17:21], uint32(len(xmlBody)))
	copy(out[21:], xmlBody)
	return out, nil
}

// NewCodec 构造 ISP 协议的 gnetx Codec。
// debug=true 时：DebugSerializer(hex) + xmlLogSerializer(XML明文)。
func NewCodec(rootName string, maxFrameLength int, debug bool) gnetx.Codec {
	ser := gnetx.Serializer(NewSerializer(rootName))
	if debug {
		ser = gnetx.DebugSerializer(ser)
		ser = &xmlLogSerializer{inner: ser}
	}
	return gnetx.NewLengthPrefixCodec(4, binary.LittleEndian, ser,
		gnetx.WithLeadingBytes([]byte{0xEB, 0x90}),
		gnetx.WithTrailingBytes([]byte{0xEB, 0x90}),
		gnetx.WithLengthOffset(lengthOffset),
		gnetx.WithLengthAdjust(2),
		gnetx.WithStripBytes(stripBytes),
		gnetx.WithMaxFrameSize(maxFrameLength),
	)
}

// EncodeFrame 将消息编码为完整 ISP 帧（含头尾标志），用于测试。
func EncodeFrame(msg *Message, rootName string, maxFrameLength int) ([]byte, error) {
	return NewCodec(rootName, maxFrameLength, false).Encode(context.Background(), msg, nil)
}

// xmlLogSerializer 打印 XML 明文（hex 由 gnetx.DebugSerializer 处理）。
type xmlLogSerializer struct{ inner gnetx.Serializer }

func (s *xmlLogSerializer) Decode(raw []byte) (any, error) {
	msg, err := s.inner.Decode(raw)
	if m, ok := msg.(*Message); ok && m.RawXML != "" {
		logx.Debugf("[isp] recv %s xml:\n%s, recvSeq: %d, sendSeq: %d", m.MessageName(), html.UnescapeString(m.RawXML), m.RecvSeq, m.SendSeq)
	}
	return msg, err
}

func (s *xmlLogSerializer) Encode(v any) ([]byte, error) {
	raw, err := s.inner.Encode(v)
	if err == nil {
		if m, ok := v.(*Message); ok {
			xml, _ := BuildXML(m, m.RootName)
			if len(xml) > 0 {
				logx.Debugf("[isp] send %s xml:\n%s, sendSeq: %d, recvSeq: %d", m.MessageName(), html.UnescapeString(string(xml)), m.SendSeq, m.RecvSeq)
			}
		}
	}
	return raw, err
}
