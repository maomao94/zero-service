package gnetx

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"testing"
)

// codec 测试用 mockConn + RawSerializer，直接测 Codec.Decode/Encode。

func TestLengthPrefixCodecFullFrame(t *testing.T) {
	c := NewLengthPrefixCodec(4, binary.BigEndian, RawSerializer{}, WithMaxFrameSize(1<<20))
	// 构造 [4B len=5][5B payload "hello"]
	frame := make([]byte, 9)
	binary.BigEndian.PutUint32(frame, 5)
	copy(frame[4:], []byte("hello"))
	mc := newMockConn(frame)

	msg, err := c.Decode(mc, nil)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	b, _ := msg.([]byte)
	if string(b) != "hello" {
		t.Fatalf("payload = %q, want hello", b)
	}
	if mc.buf.Len() != 0 {
		t.Fatalf("buffer not drained, left %d", mc.buf.Len())
	}
}

func TestLengthPrefixCodecPartial(t *testing.T) {
	c := NewLengthPrefixCodec(4, binary.BigEndian, RawSerializer{})
	// 不足长度字段
	mc := newMockConn([]byte{0x00, 0x00})
	_, err := c.Decode(mc, nil)
	if !errors.Is(err, ErrIncompletePacket) {
		t.Fatalf("want ErrIncompletePacket, got %v", err)
	}
	// 有长度字段但 payload 不全
	frame := make([]byte, 4)
	binary.BigEndian.PutUint32(frame, 5)
	mc2 := newMockConn(frame)
	_, err = c.Decode(mc2, nil)
	if !errors.Is(err, ErrIncompletePacket) {
		t.Fatalf("want ErrIncompletePacket for partial payload, got %v", err)
	}
}

func TestLengthPrefixCodecMultipleFrames(t *testing.T) {
	c := NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{})
	// [2B len=3][abc][2B len=2][de]
	buf := []byte{0x00, 0x03, 'a', 'b', 'c', 0x00, 0x02, 'd', 'e'}
	mc := newMockConn(buf)

	m1, err := c.Decode(mc, nil)
	if err != nil {
		t.Fatalf("frame1: %v", err)
	}
	if string(m1.([]byte)) != "abc" {
		t.Fatalf("frame1 = %q", m1)
	}
	m2, err := c.Decode(mc, nil)
	if err != nil {
		t.Fatalf("frame2: %v", err)
	}
	if string(m2.([]byte)) != "de" {
		t.Fatalf("frame2 = %q", m2)
	}
}

func TestLengthPrefixCodecTooLarge(t *testing.T) {
	c := NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{}, WithMaxFrameSize(10))
	buf := []byte{0x00, 100}
	mc := newMockConn(buf)
	_, err := c.Decode(mc, nil)
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("want ErrFrameTooLarge, got %v", err)
	}
}

func TestLengthPrefixCodecEncodeRoundtrip(t *testing.T) {
	c := NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{})
	out, err := c.Encode(context.Background(), []byte("hi"), nil)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	// [0x00 0x02][h i]
	if len(out) != 4 || out[0] != 0 || out[1] != 2 || out[2] != 'h' || out[3] != 'i' {
		t.Fatalf("encode = %v", out)
	}
	// 回环
	mc := newMockConn(out)
	msg, err := c.Decode(mc, nil)
	if err != nil || string(msg.([]byte)) != "hi" {
		t.Fatalf("roundtrip: %v %q", err, msg)
	}
}

func TestDelimiterCodec(t *testing.T) {
	delim := []byte{0xEB, 0x90}
	c := NewDelimiterCodec(delim, RawSerializer{}) // 默认 strip=true

	mc := newMockConn([]byte("hello\xEB\x90"))
	m, err := c.Decode(mc, nil)
	if err != nil || string(m.([]byte)) != "hello" {
		t.Fatalf("frame1: %v %q", err, m)
	}

	// 两帧粘包
	mc2 := newMockConn([]byte("a\xEB\x90bb\xEB\x90"))
	m1, _ := c.Decode(mc2, nil)
	m2, _ := c.Decode(mc2, nil)
	if string(m1.([]byte)) != "a" || string(m2.([]byte)) != "bb" {
		t.Fatalf("frames = %q %q", m1, m2)
	}
}

func TestDelimiterCodecNoStrip(t *testing.T) {
	delim := []byte{0xEB, 0x90}
	c := NewDelimiterCodec(delim, RawSerializer{}, WithDelimiterStrip(false))
	mc := newMockConn([]byte("hi\xEB\x90"))
	m, err := c.Decode(mc, nil)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if string(m.([]byte)) != "hi\xEB\x90" {
		t.Fatalf("payload = %q", m)
	}
}

func TestDelimiterCodecPartial(t *testing.T) {
	c := NewDelimiterCodec([]byte{0x0A}, RawSerializer{})
	mc := newMockConn([]byte("no-delimiter"))
	_, err := c.Decode(mc, nil)
	if !errors.Is(err, ErrIncompletePacket) {
		t.Fatalf("want ErrIncompletePacket, got %v", err)
	}
}

func TestDelimiterCodecEncode(t *testing.T) {
	c := NewDelimiterCodec([]byte{0x0A}, RawSerializer{})
	out, err := c.Encode(context.Background(), []byte("hi"), nil)
	if err != nil || string(out) != "hi\x0A" {
		t.Fatalf("encode = %q err %v", out, err)
	}
}

func TestFixedLengthCodec(t *testing.T) {
	c := NewFixedLengthCodec(3, RawSerializer{})
	mc := newMockConn([]byte("abcdef"))
	m1, _ := c.Decode(mc, nil)
	m2, _ := c.Decode(mc, nil)
	if string(m1.([]byte)) != "abc" || string(m2.([]byte)) != "def" {
		t.Fatalf("frames = %q %q", m1, m2)
	}
}

func TestFixedLengthCodecPartial(t *testing.T) {
	c := NewFixedLengthCodec(5, RawSerializer{})
	mc := newMockConn([]byte("ab"))
	_, err := c.Decode(mc, nil)
	if !errors.Is(err, ErrIncompletePacket) {
		t.Fatalf("want ErrIncompletePacket, got %v", err)
	}
}

// TestLengthPrefixCodecEncodeOverflow 验证 payload 超过长度字段容量时 Encode 报错而非静默截断。
func TestLengthPrefixCodecEncodeOverflow(t *testing.T) {
	c := NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{}) // 2 字节字段，max 65535
	big := make([]byte, 70000)                                      // 超过 65535
	_, err := c.Encode(context.Background(), big, nil)
	if err == nil {
		t.Fatal("expect error when payload exceeds length field capacity, got nil")
	}
}

// TestLengthPrefixCodecInvalidLengthBytes 验证非法 lengthBytes 在构造时 panic（而非运行时）。
func TestLengthPrefixCodecInvalidLengthBytes(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expect panic for invalid lengthBytes=3")
		}
	}()
	NewLengthPrefixCodec(3, binary.BigEndian, RawSerializer{})
}

// TestDelimiterCodecEmptyPanic 验证空分隔符在构造时 panic。
func TestDelimiterCodecEmptyPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expect panic for empty delimiter")
		}
	}()
	NewDelimiterCodec([]byte{}, RawSerializer{})
}

// TestLengthPrefixCodecNegativePayloadLen 验证负数 payloadLen（lengthAdjust 导致）返回 ErrFrameTooLarge。
func TestLengthPrefixCodecNegativePayloadLen(t *testing.T) {
	c := NewLengthPrefixCodec(1, binary.BigEndian, RawSerializer{}, WithLengthAdjust(-10))
	buf := []byte{0x01, 'x'}
	mc := newMockConn(buf)
	_, err := c.Decode(mc, nil)
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("want ErrFrameTooLarge for negative payloadLen, got %v", err)
	}
}

func TestLengthPrefixCodecUint64LengthOverflow(t *testing.T) {
	c := NewLengthPrefixCodec(8, binary.BigEndian, RawSerializer{},
		WithLengthAdjust(2), WithMaxFrameSize(1024))
	frame := make([]byte, 9)
	binary.BigEndian.PutUint64(frame[:8], math.MaxUint64)
	frame[8] = 'x'

	_, err := c.Decode(newMockConn(frame), nil)
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("want ErrFrameTooLarge for uint64-to-int overflow, got %v", err)
	}
}

func TestLengthPrefixCodecEncodeMaxFits(t *testing.T) {
	c := NewLengthPrefixCodec(1, binary.BigEndian, RawSerializer{}) // 1 字节字段，max 255
	payload := make([]byte, 255)
	out, err := c.Encode(context.Background(), payload, nil)
	if err != nil {
		t.Fatalf("Encode 255-byte payload with 1-byte field should succeed: %v", err)
	}
	if len(out) != 256 || out[0] != 255 {
		t.Fatalf("encode result unexpected: len=%d hdr=%d", len(out), out[0])
	}
}

// --- leading/trailing bytes tests ---

func TestLengthPrefixLeadingBytes(t *testing.T) {
	// EB(2) + length(2, BE) + "hello"
	c := NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{},
		WithLeadingBytes([]byte{0xEB, 0xEB}),
	)
	// Encode → Decode roundtrip
	out, err := c.Encode(context.Background(), []byte("hello"), nil)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	expect := []byte{0xEB, 0xEB, 0x00, 0x05, 'h', 'e', 'l', 'l', 'o'}
	if !bytesEqual(out, expect) {
		t.Fatalf("encode = %x, want %x", out, expect)
	}

	mc := newMockConn(out)
	msg, err := c.Decode(mc, nil)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if string(msg.([]byte)) != "hello" {
		t.Fatalf("payload = %q, want hello", msg)
	}
}

func TestLengthPrefixLeadingBytesMismatch(t *testing.T) {
	c := NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{},
		WithLeadingBytes([]byte{0xEB, 0xEB}),
	)
	// wrong prefix: 0xAA 0xAA
	mc := newMockConn([]byte{0xAA, 0xAA, 0x00, 0x03, 'f', 'o', 'o'})
	_, err := c.Decode(mc, nil)
	if err == nil {
		t.Fatal("expect prefix mismatch error, got nil")
	}
}

func TestLengthPrefixTrailingBytes(t *testing.T) {
	// length(2, BE) + payload + EB(2); length field counts trailing bytes
	c := NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{},
		WithTrailingBytes([]byte{0xEB, 0xEB}),
	)
	out, err := c.Encode(context.Background(), []byte("ok"), nil)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	// len("ok")+2 = 4 → [0x00, 0x04]["ok"][0xEB, 0xEB]
	expect := []byte{0x00, 0x04, 'o', 'k', 0xEB, 0xEB}
	if !bytesEqual(out, expect) {
		t.Fatalf("encode = %x, want %x", out, expect)
	}

	mc := newMockConn(out)
	msg, err := c.Decode(mc, nil)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if string(msg.([]byte)) != "ok" {
		t.Fatalf("payload = %q, want ok", msg)
	}
}

func TestLengthPrefixTrailingBytesMismatch(t *testing.T) {
	c := NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{},
		WithTrailingBytes([]byte{0xEB, 0xEB}),
	)
	// correct length but wrong trailing (0xAA instead of 0xEB)
	mc := newMockConn([]byte{0x00, 0x03, 'a', 0xAA, 0xAA})
	_, err := c.Decode(mc, nil)
	if err == nil {
		t.Fatal("expect suffix mismatch error, got nil")
	}
}

func TestLengthPrefixLeadingAndTrailing(t *testing.T) {
	// EB(2) + length(2, LE) + "data" + EB(2)
	c := NewLengthPrefixCodec(2, binary.LittleEndian, RawSerializer{},
		WithLeadingBytes([]byte{0xEB, 0xEB}),
		WithTrailingBytes([]byte{0xEB, 0xEB}),
	)
	out, err := c.Encode(context.Background(), []byte("data"), nil)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	// len("data")+2 = 6 (LE) → [0xEB, 0xEB][0x06, 0x00]["data"][0xEB, 0xEB]
	expect := []byte{0xEB, 0xEB, 0x06, 0x00, 'd', 'a', 't', 'a', 0xEB, 0xEB}
	if !bytesEqual(out, expect) {
		t.Fatalf("encode = %x, want %x", out, expect)
	}

	mc := newMockConn(out)
	msg, err := c.Decode(mc, nil)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if string(msg.([]byte)) != "data" {
		t.Fatalf("payload = %q, want data", msg)
	}
}

func TestLengthPrefixLeadingAndTrailingWithAdjust(t *testing.T) {
	c := NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{},
		WithLeadingBytes([]byte{0xEB, 0xEB}),
		WithTrailingBytes([]byte{0xEB, 0xEB}),
		WithLengthAdjust(2), // fieldVal = 6 - 2 = 4 (exclude trailing from length)
	)
	payload := []byte("data")
	out, err := c.Encode(context.Background(), payload, nil)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	// [EB,EB][0x00,0x04]["data"][EB,EB]
	expect := []byte{0xEB, 0xEB, 0x00, 0x04, 'd', 'a', 't', 'a', 0xEB, 0xEB}
	if !bytesEqual(out, expect) {
		t.Fatalf("encode = %x, want %x", out, expect)
	}
}

func TestLengthPrefixLeadingBytesAutoOffset(t *testing.T) {
	// WithLeadingBytes should auto-set lengthOffset
	c := NewLengthPrefixCodec(1, binary.BigEndian, RawSerializer{},
		WithLeadingBytes([]byte{0x01, 0x02, 0x03}),
	)
	if c.lengthOffset != 3 {
		t.Fatalf("auto offset = %d, want 3", c.lengthOffset)
	}
	// leading(3) + length(1) + payload "a"
	out, err := c.Encode(context.Background(), []byte("a"), nil)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	expect := []byte{0x01, 0x02, 0x03, 0x01, 'a'}
	if !bytesEqual(out, expect) {
		t.Fatalf("encode = %x, want %x", out, expect)
	}
}

func TestLengthPrefixLeadingBytesExplicitOffset(t *testing.T) {
	// Explicit offset should NOT be overridden by leading bytes
	c := NewLengthPrefixCodec(1, binary.BigEndian, RawSerializer{},
		WithLeadingBytes([]byte{0xEB, 0xEB}),
		WithLengthOffset(4),
	)
	if c.lengthOffset != 4 {
		t.Fatalf("explicit offset = %d, want 4", c.lengthOffset)
	}
}

func TestLengthPrefixLeadingBytesLargerThanOffset(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expect panic when leadingBytes > lengthOffset")
		}
	}()
	NewLengthPrefixCodec(1, binary.BigEndian, RawSerializer{},
		WithLeadingBytes([]byte{1, 2, 3, 4}),
		WithLengthOffset(2),
	)
}

func TestLengthPrefixTrailingTooShort(t *testing.T) {
	c := NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{},
		WithTrailingBytes([]byte{0xEB, 0xEB, 0xEB}),
	)
	// length=1, but trailing=3 → not enough data
	mc := newMockConn([]byte{0x00, 0x01, 0x00})
	_, err := c.Decode(mc, nil)
	if err == nil {
		t.Fatal("expect error for trailing bytes too short, got nil")
	}
}

func TestLengthPrefixPartialLeadingBytes(t *testing.T) {
	c := NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{},
		WithLeadingBytes([]byte{0xEB, 0xEB}),
	)
	// only 1 byte, can't even peek prefix+length=4 bytes
	mc := newMockConn([]byte{0xEB})
	_, err := c.Decode(mc, nil)
	if !errors.Is(err, ErrIncompletePacket) {
		t.Fatalf("want ErrIncompletePacket, got %v", err)
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// --- stripBytes tests ---

func TestLengthPrefixStripBytesDefault(t *testing.T) {
	// Default stripBytes == headerLen: serializer gets body only
	c := NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{}) // headerLen=2
	out, _ := c.Encode(context.Background(), []byte("hi"), nil)
	// [0x00 0x02][hi]
	expect := []byte{0x00, 0x02, 'h', 'i'}
	if !bytesEqual(out, expect) {
		t.Fatalf("encode = %x, want %x", out, expect)
	}

	mc := newMockConn(out)
	msg, err := c.Decode(mc, nil)
	if err != nil || string(msg.([]byte)) != "hi" {
		t.Fatalf("payload = %q, err=%v, want hi", msg, err)
	}
}

func TestLengthPrefixStripBytesZero(t *testing.T) {
	// stripBytes=0: serializer gets full frame (must include length field in output)
	c := NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{},
		WithStripBytes(0),
	)
	// Serializer produces: [0x00 0x02]["hi"] = full frame
	out, err := c.Encode(context.Background(), []byte{0x00, 0x02, 'h', 'i'}, nil)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	expect := []byte{0x00, 0x02, 'h', 'i'}
	if !bytesEqual(out, expect) {
		t.Fatalf("encode = %x, want %x", out, expect)
	}

	// Decode: serializer gets full frame
	mc := newMockConn(out)
	msg, err := c.Decode(mc, nil)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !bytesEqual(msg.([]byte), expect) {
		t.Fatalf("payload = %x, want %x", msg, expect)
	}
}

func TestLengthPrefixStripBytesKeepLength(t *testing.T) {
	// stripBytes = lengthField offset: serializer gets length field + body
	// Protocol: [msgID 2B][length 2B][body]
	c := NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{},
		WithLengthOffset(2),
		WithStripBytes(2),
	)
	// Serializer produces: [length=4][body=4B] → total frame: [msgID=0,0][0x00 0x04]["data"]
	// msgID is in the gap [0..2), filled by leadingBytes or zero; serializer handles [2..)
	payload, err := c.Encode(context.Background(), []byte{0x00, 0x04, 'd', 'a', 't', 'a'}, nil)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	// Frame: [0x00,0x00][0x00,0x04]["data"] = msgID(zeros from header gap) + length from serializer + body from serializer
	expect := []byte{0x00, 0x00, 0x00, 0x04, 'd', 'a', 't', 'a'}
	if !bytesEqual(payload, expect) {
		t.Fatalf("encode = %x, want %x", payload, expect)
	}

	mc := newMockConn(payload)
	msg, err := c.Decode(mc, nil)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	// Serializer gets: [length=4][body=4B] from stripBytes=2
	if !bytesEqual(msg.([]byte), []byte{0x00, 0x04, 'd', 'a', 't', 'a'}) {
		t.Fatalf("payload = %x", msg)
	}
}

func TestLengthPrefixStripBytesWithLeadingZero(t *testing.T) {
	// stripBytes=0 + WithLeadingBytes: leading goes at pos 0, serializer output also starts at 0
	// Serializer overwrites leading region
	c := NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{},
		WithLeadingBytes([]byte{0xEB, 0xEB}),
		WithStripBytes(0),
	)
	// Serializer output = full frame = [EB EB][0x00 0x03]["foo"]
	out, err := c.Encode(context.Background(), []byte{0xEB, 0xEB, 0x00, 0x03, 'f', 'o', 'o'}, nil)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	expect := []byte{0xEB, 0xEB, 0x00, 0x03, 'f', 'o', 'o'}
	if !bytesEqual(out, expect) {
		t.Fatalf("encode = %x, want %x", out, expect)
	}

	mc := newMockConn(out)
	msg, err := c.Decode(mc, nil)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !bytesEqual(msg.([]byte), expect) {
		t.Fatalf("payload = %x, want %x", msg, expect)
	}
}

func TestLengthPrefixStripBytesTooLarge(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expect panic for stripBytes > headerLen")
		}
	}()
	NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{},
		WithStripBytes(10),
	)
}

func TestLengthPrefixInvalidOptionsPanic(t *testing.T) {
	tests := []struct {
		name string
		opts []LengthPrefixOption
	}{
		{
			name: "negative stripBytes",
			opts: []LengthPrefixOption{WithStripBytes(-1)},
		},
		{
			name: "negative lengthOffset",
			opts: []LengthPrefixOption{WithLengthOffset(-1)},
		},
		{
			name: "negative maxFrameSize",
			opts: []LengthPrefixOption{WithMaxFrameSize(-1)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Fatal("expect panic")
				}
			}()
			NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{}, tt.opts...)
		})
	}
}

func TestLengthPrefixDecodeNegativeSerializedLenReturnsError(t *testing.T) {
	c := NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{})
	c.stripBytes = 3 // simulate an internal invariant violation after construction

	mc := newMockConn([]byte{0x00, 0x00})
	_, err := c.Decode(mc, nil)
	if err == nil {
		t.Fatal("expect error")
	}
	if mc.buf.Len() != 2 {
		t.Fatalf("buffer drained on decode error, left %d", mc.buf.Len())
	}
}

func TestLengthPrefixEncodeInvalidInternalConfigReturnsError(t *testing.T) {
	c := NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{})
	c.stripBytes = -1 // simulate an internal invariant violation after construction

	_, err := c.Encode(context.Background(), []byte("ok"), nil)
	if err == nil {
		t.Fatal("expect error")
	}
}

func TestLengthPrefixStripBytesFullRoundtrip(t *testing.T) {
	// Protocol: [msgID 2B][properties 1B][length 2B][body N]
	c := NewLengthPrefixCodec(2, binary.BigEndian, RawSerializer{},
		WithLengthOffset(3), // skip msgID(2) + props(1)
		WithStripBytes(0),   // keep everything
	)
	// Full frame from serializer: [msgID=01,02][props=03][0x00,0x02]["ok"]
	out, err := c.Encode(context.Background(), []byte{0x01, 0x02, 0x03, 0x00, 0x02, 'o', 'k'}, nil)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	expect := []byte{0x01, 0x02, 0x03, 0x00, 0x02, 'o', 'k'}
	if !bytesEqual(out, expect) {
		t.Fatalf("encode = %x, want %x", out, expect)
	}

	mc := newMockConn(out)
	msg, err := c.Decode(mc, nil)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !bytesEqual(msg.([]byte), expect) {
		t.Fatalf("payload = %x, want %x", msg, expect)
	}
}
