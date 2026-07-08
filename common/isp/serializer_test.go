package isp

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestMessageIDHelpers(t *testing.T) {
	if got := EncodeMessageID(251, 1); got != 0xfb0001 {
		t.Fatalf("register message id = %#x", got)
	}
	if got := MessageIDHeartbeat; got != 0xfb0002 {
		t.Fatalf("heartbeat message id = %#x", got)
	}
	if got := MessageIDPatrolDeviceRunData; got != 0x20000 {
		t.Fatalf("run data message id = %#x", got)
	}
	typ, command := DecodeMessageID(0xfb0004)
	if typ != 251 || command != 4 {
		t.Fatalf("decoded message id = %d/%d", typ, command)
	}
}

func TestXMLBuildParseDynamicItems(t *testing.T) {
	msg := &Message{
		RootName:    RootPatrolDevice,
		SendCode:    "dog-1",
		ReceiveCode: "server-1",
		Type:        TypePatrolDeviceRunData,
		Code:        "0",
		Command:     CommandReport,
		Time:        "2026-07-08 10:00:00",
		Items:       []Item{{"speed": "1.2", "battery": "88"}},
	}
	raw, err := BuildXML(msg, RootPatrolDevice)
	if err != nil {
		t.Fatal(err)
	}
	// Command=0 时不输出 <Command> 元素
	if bytes.Contains(raw, []byte("<Command>")) {
		t.Fatalf("Command=0 should be omitted from XML, got: %s", string(raw))
	}
	parsed, err := ParseXML(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.RootName != RootPatrolDevice || parsed.MessageID() != MessageIDPatrolDeviceRunData {
		t.Fatalf("unexpected parsed message: %#v", parsed)
	}
	if parsed.Command != 0 {
		t.Fatalf("missing Command should default to 0, got %d", parsed.Command)
	}
	if len(parsed.Items) != 1 || parsed.Items[0]["battery"] != "88" || parsed.Items[0]["speed"] != "1.2" {
		t.Fatalf("unexpected parsed items: %#v", parsed.Items)
	}
}

func TestSerializerLittleEndianRoundTrip(t *testing.T) {
	ser := NewSerializer(RootPatrolHost)
	msg := &Message{
		RootName:      RootPatrolHost,
		SendSeq:       0x0102030405060708,
		RecvSeq:       0x1112131415161718,
		SessionSource: SessionSourceClient,
		SendCode:      "client",
		ReceiveCode:   "server",
		Type:          TypeSystem,
		Command:       CommandRegister,
		Items:         []Item{{"name": "dog"}},
	}
	raw, err := ser.Encode(msg)
	if err != nil {
		t.Fatal(err)
	}
	if got := binary.LittleEndian.Uint64(raw[0:8]); got != msg.SendSeq {
		t.Fatalf("send seq endian mismatch: %#x", got)
	}
	if got := binary.LittleEndian.Uint64(raw[8:16]); got != msg.RecvSeq {
		t.Fatalf("recv seq endian mismatch: %#x", got)
	}
	if got := int(binary.LittleEndian.Uint32(raw[17:21])); got != len(raw)-serializerHeaderLen {
		t.Fatalf("xml length = %d, want %d", got, len(raw)-serializerHeaderLen)
	}
	decodedAny, err := ser.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	decoded := decodedAny.(*Message)
	if decoded.SendSeq != msg.SendSeq || decoded.RecvSeq != msg.RecvSeq || decoded.MessageID() != MessageIDRegister {
		t.Fatalf("decoded message mismatch: %#v", decoded)
	}
}

func TestCodecEncodesJavaCompatibleFrame(t *testing.T) {
	msg := &Message{
		RootName:      RootPatrolDevice,
		SendSeq:       1,
		SessionSource: SessionSourceClient,
		SendCode:      "client",
		ReceiveCode:   "server",
		Type:          TypeSystem,
		Command:       CommandHeartbeat,
	}
	frame, err := EncodeFrame(msg, RootPatrolDevice, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(frame[:2], []byte{0xEB, 0x90}) || !bytes.Equal(frame[len(frame)-2:], []byte{0xEB, 0x90}) {
		t.Fatalf("frame flags mismatch: %x", frame)
	}
	if got := binary.LittleEndian.Uint64(frame[2:10]); got != 1 {
		t.Fatalf("frame transmit seq = %d", got)
	}
	xmlLen := int(binary.LittleEndian.Uint32(frame[19:23]))
	if xmlLen != len(frame)-25 {
		t.Fatalf("xml length = %d, want %d", xmlLen, len(frame)-25)
	}
}
