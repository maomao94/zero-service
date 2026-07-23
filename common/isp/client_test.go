package isp

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"zero-service/common/gnetx"
)

func TestClientRegistrationBindsClientID(t *testing.T) {
	receivedRecvSeq := make(chan uint64, 1)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	server, err := NewServer(ServerConfig{
		ListenAddr:         addr,
		RootName:           RootPatrolDevice,
		HeartbeatInterval:  1,
		IdleTimeoutSeconds: 30,
	}, func(router *ServerRouter) {
		router.Handle(MessageIDRegister, func(_ context.Context, _ gnetx.Conn, req *Message) (*Message, error) {
			receivedRecvSeq <- req.RecvSeq
			return nil, nil
		})
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	serverDone := make(chan struct{})
	go func() {
		server.Start()
		close(serverDone)
	}()
	defer func() {
		server.Stop()
		select {
		case <-serverDone:
		case <-time.After(3 * time.Second):
			t.Fatal("server did not stop")
		}
	}()

	client, err := NewClient(ClientConfig{
		ServerAddr:        addr,
		SendCode:          "device-001",
		RootName:          RootPatrolDevice,
		HeartbeatInterval: time.Second,
		RequestTimeout:    time.Second,
		ReconnectInterval: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && testClientSession(client) == nil {
		time.Sleep(10 * time.Millisecond)
	}
	if testClientSession(client) == nil {
		t.Fatal("client session did not connect")
	}
	client.sessionAck.Store(ackState{sessionID: "old-session", recvSeq: 99})

	client.tick()
	select {
	case recvSeq := <-receivedRecvSeq:
		if recvSeq != 0 {
			t.Fatalf("registration RecvSeq = %d, want 0 for a new session", recvSeq)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not receive registration")
	}
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		sess := testClientSession(client)
		if client.IsRegistered() && sess != nil && sess.ClientID() == "device-001" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("registration did not bind client ID, registered=%v", client.IsRegistered())
}

func TestClientRegistrationPublishesStateUnderClientLock(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	registerReceived := make(chan struct{}, 1)
	releaseResponse := make(chan struct{})
	server, err := NewServer(ServerConfig{
		ListenAddr:         addr,
		RootName:           RootPatrolDevice,
		HeartbeatInterval:  1,
		IdleTimeoutSeconds: 30,
	}, func(router *ServerRouter) {
		router.Handle(MessageIDRegister, func(_ context.Context, _ gnetx.Conn, _ *Message) (*Message, error) {
			registerReceived <- struct{}{}
			<-releaseResponse
			return nil, nil
		})
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	serverDone := make(chan struct{})
	go func() {
		server.Start()
		close(serverDone)
	}()
	defer func() {
		server.Stop()
		select {
		case <-serverDone:
		case <-time.After(3 * time.Second):
			t.Fatal("server did not stop")
		}
	}()

	client, err := NewClient(ClientConfig{
		ServerAddr:          addr,
		SendCode:            "device-001",
		RegisterReceiveCode: "server-001",
		RootName:            RootPatrolDevice,
		HeartbeatInterval:   time.Second,
		RequestTimeout:      time.Second,
		ReconnectInterval:   50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	sess := waitForSession(t, client, "", 3*time.Second)
	registerDone := make(chan struct{})
	go func() {
		client.doRegister(sess)
		close(registerDone)
	}()
	select {
	case <-registerReceived:
	case <-time.After(3 * time.Second):
		t.Fatal("server did not receive registration")
	}

	client.mu.RLock()
	close(releaseResponse)
	time.Sleep(200 * time.Millisecond)
	if clientID := sess.ClientID(); clientID != "" {
		client.mu.RUnlock()
		t.Fatalf("ClientID became visible before registration state commit: %q", clientID)
	}
	client.mu.RUnlock()

	select {
	case <-registerDone:
	case <-time.After(3 * time.Second):
		t.Fatal("registration did not finish")
	}
	if !client.IsRegistered() {
		t.Fatal("client should be registered after state commit")
	}
	if got := client.ReceiveCode(); got != "server-001" {
		t.Fatalf("ReceiveCode = %q, want server-001", got)
	}
}

func TestClientRegistrationRequiresBoundCurrentSession(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	client, err := NewClient(ClientConfig{
		ServerAddr:        addr,
		SendCode:          "device-001",
		RootName:          RootPatrolDevice,
		ReconnectInterval: time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	if client.IsRegistered() {
		t.Fatal("client without a current session should not be registered")
	}
	if client.Connected() {
		t.Fatal("client without a current session should not be connected")
	}
}

func TestClientRegistrationFailureDoesNotCloseReplacementSession(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	firstRequest := make(chan struct{}, 1)
	releaseHandler := make(chan struct{})
	server, err := NewServer(ServerConfig{
		ListenAddr:         addr,
		RootName:           RootPatrolDevice,
		HeartbeatInterval:  1,
		IdleTimeoutSeconds: 30,
	}, func(router *ServerRouter) {
		router.Handle(MessageIDRegister, func(_ context.Context, _ gnetx.Conn, _ *Message) (*Message, error) {
			select {
			case firstRequest <- struct{}{}:
				<-releaseHandler
			default:
			}
			return nil, nil
		})
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	serverDone := make(chan struct{})
	go func() {
		server.Start()
		close(serverDone)
	}()
	defer func() {
		select {
		case <-releaseHandler:
		default:
			close(releaseHandler)
		}
		server.Stop()
		select {
		case <-serverDone:
		case <-time.After(3 * time.Second):
			t.Fatal("server did not stop")
		}
	}()

	client, err := NewClient(ClientConfig{
		ServerAddr:        addr,
		SendCode:          "device-001",
		RootName:          RootPatrolDevice,
		HeartbeatInterval: time.Second,
		RequestTimeout:    300 * time.Millisecond,
		ReconnectInterval: 30 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	firstSession := waitForSession(t, client, "", 3*time.Second)
	registerDone := make(chan struct{})
	go func() {
		client.doRegister(firstSession)
		close(registerDone)
	}()
	select {
	case <-firstRequest:
	case <-time.After(3 * time.Second):
		t.Fatal("server did not receive registration")
	}

	serverSessions := server.Manager().All()
	if len(serverSessions) != 1 {
		t.Fatalf("server session count = %d, want 1", len(serverSessions))
	}
	if err := serverSessions[0].Close(); err != nil {
		t.Fatalf("close first server session: %v", err)
	}
	replacement := waitForSession(t, client, firstSession.SessionID(), 3*time.Second)

	select {
	case <-registerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("registration request did not finish")
	}
	close(releaseHandler)

	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		current := testClientSession(client)
		if current == nil || current.SessionID() != replacement.SessionID() {
			t.Fatalf("replacement session was closed by stale registration failure: got %v, want %s", current, replacement.SessionID())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestClientTrackRecvSeqIgnoresStaleSession(t *testing.T) {
	client := &Client{}
	client.sessionAck.Store(ackState{sessionID: "current-session", recvSeq: 5})

	client.trackRecvSeq(100, "old-session")
	client.trackRecvSeq(4, "current-session")
	client.trackRecvSeq(6, "current-session")

	got := client.sessionAck.Load().(ackState).recvSeq
	if got != 6 {
		t.Fatalf("sessionAck.recvSeq = %d, want 6", got)
	}
}

func TestClientTrackRecvSeqConcurrent(t *testing.T) {
	client := &Client{}
	client.sessionAck.Store(ackState{sessionID: "current-session"})

	var wg sync.WaitGroup
	for seq := uint64(1); seq <= 100; seq++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.trackRecvSeq(seq, "current-session")
		}()
	}
	wg.Wait()

	got := client.sessionAck.Load().(ackState)
	if got.sessionID != "current-session" || got.recvSeq != 100 {
		t.Fatalf("sessionAck = %+v, want sessionID=current-session recvSeq=100", got)
	}
}

func waitForSession(t *testing.T, client *Client, previousID string, timeout time.Duration) gnetx.ClientConn {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		sess := testClientSession(client)
		if sess != nil && sess.SessionID() != previousID {
			return sess
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("client session did not become available within %s", timeout)
	return nil
}

func testClientSession(client *Client) gnetx.ClientConn {
	return client.transport.Session()
}
