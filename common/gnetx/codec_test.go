package gnetx

import (
	"encoding/binary"
	"errors"
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
	out, err := c.Encode([]byte("hi"), nil)
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
	out, err := c.Encode([]byte("hi"), nil)
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
	_, err := c.Encode(big, nil)
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

// TestLengthPrefixCodecEncodeMaxFits 验证恰好等于容量的 payload 能正常编码。
func TestLengthPrefixCodecEncodeMaxFits(t *testing.T) {
	c := NewLengthPrefixCodec(1, binary.BigEndian, RawSerializer{}) // 1 字节字段，max 255
	payload := make([]byte, 255)
	out, err := c.Encode(payload, nil)
	if err != nil {
		t.Fatalf("Encode 255-byte payload with 1-byte field should succeed: %v", err)
	}
	if len(out) != 256 || out[0] != 255 {
		t.Fatalf("encode result unexpected: len=%d hdr=%d", len(out), out[0])
	}
}
