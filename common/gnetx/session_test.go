package gnetx

import (
	"testing"
)

func TestSessionManagerGetAllCount(t *testing.T) {
	mgr := NewSessionManager(nil)
	mc1 := newMockConn(nil)
	mc2 := newMockConn(nil)
	sess1 := newSession("id1", mc1, newTestCodec(), mgr, false)
	sess2 := newSession("id2", mc2, newTestCodec(), mgr, false)
	mgr.add(sess1)
	mgr.add(sess2)

	if c := mgr.Count(); c != 2 {
		t.Fatalf("Count = %d, want 2", c)
	}
	if mgr.Get("id1") != sess1 {
		t.Fatal("Get by id failed")
	}
	if mgr.Get("id2") != sess2 {
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
	sess1 := newSession("id1", mc1, newTestCodec(), mgr, false)
	sess2 := newSession("id2", mc2, newTestCodec(), mgr, false)
	mgr.add(sess1)
	mgr.add(sess2)

	sess1.Register("alias-x")
	if mgr.Get("alias-x") != sess1 {
		t.Fatal("alias lookup should return sess1")
	}

	sess2.Register("alias-x")
	if mgr.Get("alias-x") != sess2 {
		t.Fatal("alias should now point to sess2")
	}
	if !sess1.isClosed() {
		t.Fatal("sess1 should be closed (kicked) after alias conflict")
	}
}

func TestSessionManagerRemove(t *testing.T) {
	mgr := NewSessionManager(nil)
	mc := newMockConn(nil)
	sess := newSession("id1", mc, newTestCodec(), mgr, false)
	mgr.add(sess)
	sess.Register("alias-1")

	if mgr.Count() != 1 {
		t.Fatalf("Count = %d, want 1", mgr.Count())
	}

	sess.Close()

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
	sess := newSession("id1", mc, newTestCodec(), nil, false)

	sess.SetAttribute("key1", "value1")
	sess.SetAttribute(42, 100)

	if v := sess.Attribute("key1"); v != "value1" {
		t.Fatalf("Attribute key1 = %v", v)
	}
	if v := sess.Attribute(42); v != 100 {
		t.Fatalf("Attribute 42 = %v", v)
	}
	if sess.Attribute("noexist") != nil {
		t.Fatal("Attribute should be nil for unset key")
	}

	sess.DeleteAttribute("key1")
	if sess.Attribute("key1") != nil {
		t.Fatal("Attribute should be nil after delete")
	}
	if v := sess.Attribute(42); v != 100 {
		t.Fatal("Delete key1 should not affect key 42")
	}
}

func TestSessionCloseIdempotent(t *testing.T) {
	mc := newMockConn(nil)
	sess := newSession("id1", mc, newTestCodec(), nil, false)

	if err := sess.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := sess.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if !sess.isClosed() {
		t.Fatal("session should be closed")
	}
}

func TestSessionIsClient(t *testing.T) {
	mc := newMockConn(nil)
	sessServer := newSession("s1", mc, newTestCodec(), nil, false)
	sessClient := newSession("c1", mc, newTestCodec(), nil, true)

	if sessServer.IsClient() {
		t.Fatal("server session should not be client")
	}
	if !sessClient.IsClient() {
		t.Fatal("client session should be client")
	}
}

func TestSessionCreatedAt(t *testing.T) {
	mc := newMockConn(nil)
	sess := newSession("id1", mc, newTestCodec(), nil, false)
	if sess.CreatedAt().IsZero() {
		t.Fatal("CreatedAt should not be zero")
	}
}

func TestSessionRemoteAddrLocalAddr(t *testing.T) {
	mc := newMockConn(nil)
	sess := newSession("id1", mc, newTestCodec(), nil, false)

	if sess.RemoteAddr() == nil {
		t.Fatal("RemoteAddr should not be nil")
	}
	if sess.LocalAddr() == nil {
		t.Fatal("LocalAddr should not be nil")
	}
}

func TestSessionIDAndAlias(t *testing.T) {
	mc := newMockConn(nil)
	sess := newSession("my-id", mc, newTestCodec(), nil, false)

	if sess.ID() != "my-id" {
		t.Fatalf("ID = %q, want my-id", sess.ID())
	}
	if sess.Alias() != "" {
		t.Fatal("Alias should be empty before Register")
	}

	mgr := NewSessionManager(nil)
	sess2 := newSession("id2", newMockConn(nil), newTestCodec(), mgr, false)
	mgr.add(sess2)
	sess2.Register("device-1")
	if sess2.Alias() != "device-1" {
		t.Fatalf("Alias = %q, want device-1", sess2.Alias())
	}
}

func TestSessionLastActiveAt(t *testing.T) {
	mc := newMockConn(nil)
	sess := newSession("id1", mc, newTestCodec(), nil, false)

	before := sess.LastActiveAt()
	if before.IsZero() {
		t.Fatal("LastActiveAt should not be zero")
	}

	sess.touch()
	after := sess.LastActiveAt()
	if after.Before(before) {
		t.Fatal("touch should advance LastActiveAt")
	}
}
