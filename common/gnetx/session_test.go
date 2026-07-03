package gnetx

import (
	"context"
	"errors"
	"testing"
	"time"

	"zero-service/common/antsx"
)

func TestSessionManagerGetAllCount(t *testing.T) {
	mgr := NewSessionManager(nil)
	mc1 := newMockConn(nil)
	mc2 := newMockConn(nil)
	cn1 := newSession("id1", mc1, newTestCodec(), mgr, nil)
	cn2 := newSession("id2", mc2, newTestCodec(), mgr, nil)
	mgr.add(cn1)
	mgr.add(cn2)

	if c := mgr.Count(); c != 2 {
		t.Fatalf("Count = %d, want 2", c)
	}
	if mgr.Get("id1") != cn1 {
		t.Fatal("Get by id failed")
	}
	if mgr.Get("id2") != cn2 {
		t.Fatal("Get by id failed")
	}
	if mgr.Get("nonexistent") != nil {
		t.Fatal("Get nonexistent should return nil")
	}

	all := mgr.All()
	if len(all) != 2 {
		t.Fatalf("All len = %d, want 2", len(all))
	}
}

func TestSessionManagerAliasConflict(t *testing.T) {
	mgr := NewSessionManager(nil)
	mc1 := newMockConn(nil)
	mc2 := newMockConn(nil)
	cn1 := newSession("id1", mc1, newTestCodec(), mgr, nil)
	cn2 := newSession("id2", mc2, newTestCodec(), mgr, nil)
	mgr.add(cn1)
	mgr.add(cn2)

	cn1.Register("alias-x")
	if mgr.Get("alias-x") != cn1 {
		t.Fatal("alias lookup should return cn1")
	}

	cn2.Register("alias-x")
	if mgr.Get("alias-x") != cn2 {
		t.Fatal("alias should now point to cn2")
	}
	if !cn1.isClosed() {
		t.Fatal("cn1 should be closed (kicked) after alias conflict")
	}
}

func TestSessionManagerRemove(t *testing.T) {
	mgr := NewSessionManager(nil)
	mc := newMockConn(nil)
	cn := newSession("id1", mc, newTestCodec(), mgr, nil)
	mgr.add(cn)
	cn.Register("alias-1")

	if mgr.Count() != 1 {
		t.Fatalf("Count = %d, want 1", mgr.Count())
	}

	cn.Close()

	if mgr.Count() != 0 {
		t.Fatalf("Count after close = %d, want 0", mgr.Count())
	}
	if mgr.Get("id1") != nil {
		t.Fatal("Get after close should return nil")
	}
	if mgr.Get("alias-1") != nil {
		t.Fatal("Get alias after close should return nil")
	}
}

func TestSessionAttributes(t *testing.T) {
	mc := newMockConn(nil)
	cn := newSession("id1", mc, newTestCodec(), nil, nil)

	cn.SetAttribute("key1", "value1")
	cn.SetAttribute(42, 100)

	if v := cn.Attribute("key1"); v != "value1" {
		t.Fatalf("Attribute key1 = %v", v)
	}
	if v := cn.Attribute(42); v != 100 {
		t.Fatalf("Attribute 42 = %v", v)
	}
	if cn.Attribute("noexist") != nil {
		t.Fatal("Attribute should be nil for unset key")
	}

	cn.DeleteAttribute("key1")
	if cn.Attribute("key1") != nil {
		t.Fatal("Attribute should be nil after delete")
	}
	if v := cn.Attribute(42); v != 100 {
		t.Fatal("Delete key1 should not affect key 42")
	}
}

func TestSessionCloseIdempotent(t *testing.T) {
	mc := newMockConn(nil)
	cn := newSession("id1", mc, newTestCodec(), nil, nil)

	if err := cn.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := cn.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if !cn.isClosed() {
		t.Fatal("conn should be closed")
	}
}

func TestSessionCreatedAt(t *testing.T) {
	mc := newMockConn(nil)
	cn := newSession("id1", mc, newTestCodec(), nil, nil)
	if cn.CreatedAt().IsZero() {
		t.Fatal("CreatedAt should not be zero")
	}
}

func TestSessionRemoteAddrLocalAddr(t *testing.T) {
	mc := newMockConn(nil)
	cn := newSession("id1", mc, newTestCodec(), nil, nil)

	if cn.RemoteAddr() == nil {
		t.Fatal("RemoteAddr should not be nil")
	}
	if cn.LocalAddr() == nil {
		t.Fatal("LocalAddr should not be nil")
	}
}

func TestSessionIDAndAlias(t *testing.T) {
	mc := newMockConn(nil)
	cn := newSession("my-id", mc, newTestCodec(), nil, nil)

	if cn.ID() != "my-id" {
		t.Fatalf("ID = %q, want my-id", cn.ID())
	}
	if cn.Alias() != "" {
		t.Fatal("Alias should be empty before Register")
	}

	mgr := NewSessionManager(nil)
	cn2 := newSession("id2", newMockConn(nil), newTestCodec(), mgr, nil)
	mgr.add(cn2)
	cn2.Register("device-1")
	if cn2.Alias() != "device-1" {
		t.Fatalf("Alias = %q, want device-1", cn2.Alias())
	}
}

func TestSessionLastActiveAt(t *testing.T) {
	mc := newMockConn(nil)
	cn := newSession("id1", mc, newTestCodec(), nil, nil)

	before := cn.LastActiveAt()
	if before.IsZero() {
		t.Fatal("LastActiveAt should not be zero")
	}

	cn.touch()
	after := cn.LastActiveAt()
	if after.Before(before) {
		t.Fatal("touch should advance LastActiveAt")
	}
}

func TestSessionSendOnClosed(t *testing.T) {
	mc := newMockConn(nil)
	cn := newSession("id1", mc, newTestCodec(), nil, nil)
	cn.Close()

	if err := cn.Send(nil, &echoMsg{Body: "x"}); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("Send on closed: want ErrSessionClosed, got %v", err)
	}
}

func TestSessionRequestOnClosed(t *testing.T) {
	mc := newMockConn(nil)
	cn := newSession("id1", mc, newTestCodec(), nil, nil)
	cn.Close()

	_, err := cn.Request(nil, &pingReq{Serial: 1}, time.Second)
	if !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("Request on closed: want ErrSessionClosed, got %v", err)
	}
}

func TestSessionRequestNilReplyPool(t *testing.T) {
	mc := newMockConn(nil)
	cn := newSession("id1", mc, newTestCodec(), nil, nil)

	_, err := cn.Request(nil, &pingReq{Serial: 1}, time.Second)
	if !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("Request with nil replyPool: want ErrSessionClosed, got %v", err)
	}
}

func TestSessionResolveResponseNilReplyPool(t *testing.T) {
	mc := newMockConn(nil)
	cn := newSession("id1", mc, newTestCodec(), nil, nil)

	if cn.resolveResponse("tid1", &pongResp{RespSerial: 1}) {
		t.Fatal("resolveResponse with nil replyPool should return false")
	}
}

func TestSessionResolveResponse(t *testing.T) {
	mc := newMockConn(nil)
	replyPool := antsx.NewReplyPool[any]()
	defer replyPool.Close()
	cn := newSession("s1", mc, newTestCodec(), nil, replyPool)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() {
		_, _ = cn.Request(ctx, &pingReq{Serial: 10}, time.Second)
	}()

	time.Sleep(50 * time.Millisecond)

	ok := cn.resolveResponse("10", &pongResp{RespSerial: 10, Reply: "ok"})
	if !ok {
		t.Fatal("resolveResponse should return true for matching TID")
	}
}

func TestNewSessionID(t *testing.T) {
	id1 := newSessionID()
	id2 := newSessionID()
	if id1 == "" || id2 == "" {
		t.Fatal("session IDs should not be empty")
	}
	if id1 == id2 {
		t.Fatal("two session IDs should be unique")
	}
}
