package gnetx

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/panjf2000/gnet/v2"

	"zero-service/common/antsx"
)

type ctxTestKey struct{}

type ctxCaptureCodec struct {
	got context.Context
}

func (c *ctxCaptureCodec) Decode(gnet.Conn, Conn) (any, error) { return nil, nil }

func (c *ctxCaptureCodec) Encode(ctx context.Context, _ any, _ Conn) ([]byte, error) {
	c.got = ctx
	return []byte("ok"), nil
}

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

	if err := cn.Send(context.Background(), &echoMsg{Body: "x"}); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("Send on closed: want ErrSessionClosed, got %v", err)
	}
}

func TestSessionSendPassesContextToCodec(t *testing.T) {
	codec := &ctxCaptureCodec{}
	cn := newSession("id1", newMockConn(nil), codec, nil, nil)
	ctx := context.WithValue(context.Background(), ctxTestKey{}, "value")

	if err := cn.Send(ctx, &echoMsg{Body: "x"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got := codec.got.Value(ctxTestKey{}); got != "value" {
		t.Fatalf("Encode ctx value = %v, want value", got)
	}
}

func TestServerWriteReplyPassesContextToCodec(t *testing.T) {
	codec := &ctxCaptureCodec{}
	cn := newSession("id1", newMockConn(nil), codec, nil, nil)
	srv := &Server{opts: ServerOptions{Codec: codec}}
	ctx := context.WithValue(context.Background(), ctxTestKey{}, "reply")

	if err := srv.writeReply(ctx, cn, &echoMsg{Body: "x"}); err != nil {
		t.Fatalf("writeReply: %v", err)
	}
	if got := codec.got.Value(ctxTestKey{}); got != "reply" {
		t.Fatalf("reply Encode ctx value = %v, want reply", got)
	}
}

func TestSessionRequestOnClosed(t *testing.T) {
	mc := newMockConn(nil)
	cn := newSession("id1", mc, newTestCodec(), nil, nil)
	cn.Close()

	_, err := cn.Request(context.Background(), &pingReq{Serial: 1}, time.Second)
	if !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("Request on closed: want ErrSessionClosed, got %v", err)
	}
}

func TestSessionRequestNilReplyPool(t *testing.T) {
	mc := newMockConn(nil)
	cn := newSession("id1", mc, newTestCodec(), nil, nil)

	_, err := cn.Request(context.Background(), &pingReq{Serial: 1}, time.Second)
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

func TestSessionNextSendSeqStartsAtConfiguredValue(t *testing.T) {
	cn := newSession("id1", newMockConn(nil), newTestCodec(), nil, nil, 7)

	if seq := cn.NextSendSeq(); seq != 7 {
		t.Fatalf("first NextSendSeq = %d, want 7", seq)
	}
	if seq := cn.NextSendSeq(); seq != 8 {
		t.Fatalf("second NextSendSeq = %d, want 8", seq)
	}
}

func TestSequenceStartAppliedByConnectionOwners(t *testing.T) {
	serverConn := newMockConn(nil)
	srv := &Server{
		opts: ServerOptions{Codec: newTestCodec(), SequenceStart: 10},
		mgr:  NewSessionManager(nil),
	}
	srv.OnOpen(serverConn)
	serverSession := serverConn.Context().(*session)
	if seq := serverSession.NextSendSeq(); seq != 10 {
		t.Fatalf("server first seq = %d, want 10", seq)
	}

	clientConn := newMockConn(nil)
	cli := &Client{opts: ClientOptions{Codec: newTestCodec(), SequenceStart: 20}}
	cli.OnOpen(clientConn)
	clientSession := clientConn.Context().(*session)
	if seq := clientSession.NextSendSeq(); seq != 20 {
		t.Fatalf("client first seq = %d, want 20", seq)
	}

	dialerConn := newMockConn(nil)
	dialer := &Dialer{opts: ClientOptions{Codec: newTestCodec(), SequenceStart: 30}}
	dialer.OnOpen(dialerConn)
	dialerSession := dialerConn.Context().(*session)
	if seq := dialerSession.NextSendSeq(); seq != 30 {
		t.Fatalf("dialer first seq = %d, want 30", seq)
	}
}

func TestNextSendSeqConcurrent(t *testing.T) {
	cn := newSession("id1", newMockConn(nil), newTestCodec(), nil, nil)
	const goroutines = 16
	const calls = 1000
	total := goroutines * calls

	seen := make(map[uint64]bool, total)
	var mu sync.Mutex
	var wg sync.WaitGroup

	var errOnce sync.Once
	var firstErr error
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < calls; i++ {
				seq := cn.NextSendSeq()
				mu.Lock()
				if seen[seq] {
					errOnce.Do(func() { firstErr = errors.New("duplicate seq") })
					mu.Unlock()
					return
				}
				seen[seq] = true
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if firstErr != nil {
		t.Fatal(firstErr)
	}

	for i := uint64(0); i < uint64(total); i++ {
		if !seen[i] {
			t.Fatalf("missing seq %d", i)
		}
	}
}

func TestNextSendSeqViaConnInterface(t *testing.T) {
	cn := newSession("id1", newMockConn(nil), newTestCodec(), nil, nil)
	var conn Conn = cn

	if seq := conn.NextSendSeq(); seq != 0 {
		t.Fatalf("Conn.NextSendSeq = %d, want 0", seq)
	}
	if seq := conn.NextSendSeq(); seq != 1 {
		t.Fatalf("Conn.NextSendSeq = %d, want 1", seq)
	}
}

func TestNextSendSeqViaServerConn(t *testing.T) {
	srv := &Server{
		opts: ServerOptions{Codec: newTestCodec(), SequenceStart: 5},
		mgr:  NewSessionManager(nil),
	}
	conn := newMockConn(nil)
	srv.OnOpen(conn)
	srv.mgr.add(conn.Context().(*session))

	serverConn := srv.mgr.Get(conn.Context().(*session).ID()).(ServerConn)
	if seq := serverConn.NextSendSeq(); seq != 5 {
		t.Fatalf("ServerConn.NextSendSeq = %d, want 5", seq)
	}
	if seq := serverConn.NextSendSeq(); seq != 6 {
		t.Fatalf("ServerConn.NextSendSeq = %d, want 6", seq)
	}
}

func TestNextSendSeqViaClientConn(t *testing.T) {
	conn := newMockConn(nil)
	cli := &Client{opts: ClientOptions{Codec: newTestCodec(), SequenceStart: 100}}
	cli.OnOpen(conn)

	clientConn := conn.Context().(*session)
	if seq := clientConn.NextSendSeq(); seq != 100 {
		t.Fatalf("ClientConn.NextSendSeq = %d, want 100", seq)
	}
	if seq := clientConn.NextSendSeq(); seq != 101 {
		t.Fatalf("ClientConn.NextSendSeq = %d, want 101", seq)
	}
}

func TestNextSendSeqViaDialerConn(t *testing.T) {
	conn := newMockConn(nil)
	d := &Dialer{opts: ClientOptions{Codec: newTestCodec(), SequenceStart: 200}}
	d.OnOpen(conn)

	dialerConn := conn.Context().(*session)
	if seq := dialerConn.NextSendSeq(); seq != 200 {
		t.Fatalf("DialerConn.NextSendSeq = %d, want 200", seq)
	}
	if seq := dialerConn.NextSendSeq(); seq != 201 {
		t.Fatalf("DialerConn.NextSendSeq = %d, want 201", seq)
	}
}
