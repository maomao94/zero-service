package mcpx

import (
	"context"
	"crypto/rand"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zeromicro/go-zero/core/logx"
)

// postAuthInfo holds auth information captured from SSE POST requests.
type postAuthInfo struct {
	TokenInfo *auth.TokenInfo
	Header    http.Header
}

// authSSESession represents a single SSE session with per-request auth capture.
type authSSESession struct {
	transport *sdkmcp.SSEServerTransport
	authInfo  atomic.Pointer[postAuthInfo]
}

// authSSEHandler is a custom SSE handler that properly injects RequestExtra
// (TokenInfo + Header) from POST requests into MCP tool handler context.
//
// The standard SDK SSEHandler does not propagate POST-request auth context
// because it only pushes the JSON-RPC message body through a channel,
// discarding the HTTP request context. This handler bridges that gap by:
//  1. Capturing TokenInfo from the POST request's context (set by auth middleware)
//  2. Injecting it as RequestExtra when the message is read by the transport
type authSSEHandler struct {
	getServer func(*http.Request) *sdkmcp.Server

	mu       sync.Mutex
	sessions map[string]*authSSESession
}

func newAuthSSEHandler(getServer func(*http.Request) *sdkmcp.Server) *authSSEHandler {
	return &authSSEHandler{
		getServer: getServer,
		sessions:  make(map[string]*authSSESession),
	}
}

func (h *authSSEHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	sessionID := req.URL.Query().Get("sessionid")

	if req.Method == http.MethodPost {
		if sessionID == "" {
			http.Error(w, "sessionid must be provided", http.StatusBadRequest)
			return
		}
		h.mu.Lock()
		sess := h.sessions[sessionID]
		h.mu.Unlock()
		if sess == nil {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}

		// Capture POST request's auth info before delegating to SDK transport.
		// This must happen before ServeHTTP pushes the message to the channel,
		// ensuring the Read side sees the correct TokenInfo.
		if ti := auth.TokenInfoFromContext(req.Context()); ti != nil {
			sess.authInfo.Store(&postAuthInfo{
				TokenInfo: ti,
				Header:    req.Header.Clone(),
			})
		}

		sess.transport.ServeHTTP(w, req)
		return
	}

	if req.Method != http.MethodGet {
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// GET: create a new SSE session
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sessionID = sseRandText()
	endpoint, err := req.URL.Parse("?sessionid=" + sessionID)
	if err != nil {
		http.Error(w, "internal error: failed to create endpoint", http.StatusInternalServerError)
		return
	}

	transport := &sdkmcp.SSEServerTransport{Endpoint: endpoint.RequestURI(), Response: w}
	sess := &authSSESession{transport: transport}

	h.mu.Lock()
	h.sessions[sessionID] = sess
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.sessions, sessionID)
		h.mu.Unlock()
	}()

	server := h.getServer(req)
	if server == nil {
		http.Error(w, "no server available", http.StatusBadRequest)
		return
	}

	// Wrap transport to inject RequestExtra on every Read.
	wrapped := &authSSETransport{inner: transport, sess: sess}

	logx.Infof("New SSE connection from %s, sessionId=%s", req.RemoteAddr, sessionID)
	ss, err := server.Connect(req.Context(), wrapped, nil)
	if err != nil {
		http.Error(w, "connection failed", http.StatusInternalServerError)
		return
	}
	defer ss.Close()

	// Block until the client disconnects.
	<-req.Context().Done()
}

// authSSETransport wraps SSEServerTransport to inject RequestExtra on Read.
type authSSETransport struct {
	inner *sdkmcp.SSEServerTransport
	sess  *authSSESession
}

func (t *authSSETransport) Connect(ctx context.Context) (sdkmcp.Connection, error) {
	conn, err := t.inner.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &authSSEConn{Connection: conn, sess: t.sess}, nil
}

// authSSEConn wraps a Connection, injecting RequestExtra from captured POST auth info.
type authSSEConn struct {
	sdkmcp.Connection
	sess *authSSESession
}

func (c *authSSEConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	msg, err := c.Connection.Read(ctx)
	if err != nil {
		return nil, err
	}
	if req, ok := msg.(*jsonrpc.Request); ok {
		re := &sdkmcp.RequestExtra{}
		if info := c.sess.authInfo.Load(); info != nil {
			re.TokenInfo = info.TokenInfo
			re.Header = info.Header
		}
		req.Extra = re
	}
	return msg, nil
}

// sseRandText generates a random session ID for SSE connections.
func sseRandText() string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	src := make([]byte, 26)
	rand.Read(src)
	for i := range src {
		src[i] = alphabet[src[i]%32]
	}
	return string(src)
}
