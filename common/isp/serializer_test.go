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

func TestMessageResponseTIDOnlyForResponses(t *testing.T) {
	tests := []struct {
		name    string
		msg     *Message
		wantTID string
	}{
		{
			name: "generic response without items",
			msg: &Message{
				RecvSeq: 123,
				Type:    TypeSystem,
				Command: CommandGenericResponseWithoutItems,
			},
			wantTID: "123",
		},
		{
			name: "generic response with items",
			msg: &Message{
				RecvSeq: 456,
				Type:    TypeSystem,
				Command: CommandGenericResponseWithItems,
			},
			wantTID: "456",
		},
		{
			name: "heartbeat command is not a response",
			msg: &Message{
				RecvSeq: 789,
				Type:    TypeSystem,
				Command: CommandHeartbeat,
			},
		},
		{
			name: "proactive report is not a response",
			msg: &Message{
				RecvSeq: 789,
				Type:    TypePatrolDeviceRunData,
				Command: CommandReport,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.ResponseTID(); got != tt.wantTID {
				t.Fatalf("ResponseTID() = %q, want %q", got, tt.wantTID)
			}
		})
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

func TestXMLParseModelUpdateMultiItems(t *testing.T) {
	xmlBizData := `<PatrolDevice>
    <SendCode>testDog</SendCode>
    <ReceiveCode>Server01</ReceiveCode>
    <Code>变电站编码</Code>
    <Type>11</Type>
    <Time>2025-01-09 12:00:00</Time>
    <Items>
        <Item time="2025-01-09 12:00:00" type="1" file_path="/path/to/deviceA_model.xml"/>
        <Item time="2025-01-09 12:05:00" type="2" file_path="/path/to/region_host_model.xml"/>
        <Item time="2025-01-09 12:10:00" type="3" file_path="/path/to/robot_model.xml"/>
        <Item time="2025-01-09 12:15:00" type="4" file_path="/path/to/camera_model.xml"/>
        <Item time="2025-01-09 12:20:00" type="5" file_path="/path/to/drone_model.xml"/>
        <Item time="2025-01-09 12:25:00" type="6" file_path="/path/to/voice_model.xml"/>
        <Item time="2025-01-09 12:30:00" type="7" file_path="/path/to/task_model.xml"/>
        <Item time="2025-01-09 12:35:00" type="8" file_path="/path/to/maintenance_config.xml"/>
        <Item time="2025-01-09 12:40:00" type="9" file_path="/path/to/map_file.xml"/>
        <Item time="2025-01-09 12:45:00" type="10" file_path="/path/to/maintenance_record.xml"/>
        <Item time="2025-01-09 12:50:00" type="11" file_path="/path/to/linkage_config.xml"/>
        <Item time="2025-01-09 12:55:00" type="12" file_path="/path/to/alarm_threshold_model.xml"/>
    </Items>
</PatrolDevice>`

	msg, err := ParseXML([]byte(xmlBizData))
	if err != nil {
		t.Fatal(err)
	}
	if msg.Type != TypeModelUpdateReport {
		t.Fatalf("type=%d, want %d", msg.Type, TypeModelUpdateReport)
	}
	if msg.Command != 0 {
		t.Fatalf("command=%d, want 0 (report)", msg.Command)
	}
	if msg.MessageID() != MessageIDModelUpdateReport {
		t.Fatalf("messageID=%#x, want %#x", msg.MessageID(), MessageIDModelUpdateReport)
	}
	if len(msg.Items) != 12 {
		t.Fatalf("items=%d, want 12", len(msg.Items))
	}
	for i, item := range msg.Items {
		typ := item["type"]
		fp := item["file_path"]
		if typ == "" || fp == "" {
			t.Fatalf("item[%d] type=%s file_path=%s", i, typ, fp)
		}
		t.Logf("item[%d]: type=%s file_path=%s", i, typ, fp)
	}
}
