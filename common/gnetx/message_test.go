package gnetx

import "testing"

// 测试用 opt-in 接口 stub 类型，包级定义。
type testReq struct{ serial int }

func (r testReq) TID() string { return "tid-" + itoa(r.serial) }

type testResp struct{ respSerial int }

func (r testResp) ResponseTID() string { return "tid-" + itoa(r.respSerial) }

type testMsg struct{ id int }

func (m testMsg) MessageID() int { return m.id }

type testClient struct{ cid string }

func (c testClient) ClientID() string { return c.cid }

func TestMessageInterfaces(t *testing.T) {
	var (
		corr Correlatable       = testReq{serial: 1}
		resp Response           = testResp{respSerial: 1}
		id   Identifiable       = testMsg{id: 42}
		cli  ClientIdentifiable = testClient{cid: "dev-001"}
	)

	if corr.TID() != "tid-1" {
		t.Fatalf("Correlatable.TID = %q, want tid-1", corr.TID())
	}
	if resp.ResponseTID() != "tid-1" {
		t.Fatalf("Response.ResponseTID = %q, want tid-1", resp.ResponseTID())
	}
	if id.MessageID() != 42 {
		t.Fatalf("Identifiable.MessageID = %d, want 42", id.MessageID())
	}
	if cli.ClientID() != "dev-001" {
		t.Fatalf("ClientIdentifiable.ClientID = %q, want dev-001", cli.ClientID())
	}

	// 未实现接口的消息应断言失败
	plain := struct{ x int }{x: 1}
	if _, ok := any(plain).(Correlatable); ok {
		t.Fatal("plain struct should not satisfy Correlatable")
	}
	if _, ok := any(plain).(Identifiable); ok {
		t.Fatal("plain struct should not satisfy Identifiable")
	}
}

func TestErrorsAreSentinel(t *testing.T) {
	errs := []error{
		ErrIncompletePacket,
		ErrFrameTooLarge,
		ErrSessionClosed,
		ErrNoHandler,
		ErrPendingNotFound,
		errRawSerializerType,
	}
	for _, e := range errs {
		if e == nil {
			t.Fatal("sentinel error must not be nil")
		}
	}
}
