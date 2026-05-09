package netx

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zeromicro/go-zero/rest/httpc"
)

func TestNewHTTPCServiceDefaultHTTPClient(t *testing.T) {
	svc := NewHTTPCService("test-httpc")
	if svc == nil {
		t.Fatal("expected httpc service")
	}
}

func TestNewHTTPClient_DefaultTransport(t *testing.T) {
	client := NewHTTPClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected http transport")
	}
	if transport.Proxy == nil {
		t.Fatal("expected environment proxy")
	}
	if !transport.ForceAttemptHTTP2 {
		t.Fatal("expected HTTP/2 enabled")
	}
	if transport.MaxIdleConns != 100 || transport.MaxIdleConnsPerHost != 10 {
		t.Fatalf("unexpected connection pool config: max=%d perHost=%d", transport.MaxIdleConns, transport.MaxIdleConnsPerHost)
	}
	if transport.IdleConnTimeout == 0 || transport.TLSHandshakeTimeout == 0 || transport.ExpectContinueTimeout == 0 {
		t.Fatal("expected transport timeouts configured")
	}
}

func TestNewClient_DefaultTransport(t *testing.T) {
	c := NewClient()
	eng, ok := c.engine.(*DefaultEngine)
	if !ok {
		t.Fatal("expected default engine")
	}
	transport, ok := eng.client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected http transport")
	}
	if transport.Proxy == nil {
		t.Fatal("expected environment proxy")
	}
	if !transport.ForceAttemptHTTP2 {
		t.Fatal("expected HTTP/2 enabled")
	}
	if transport.MaxIdleConns == 0 || transport.MaxIdleConnsPerHost == 0 {
		t.Fatalf("expected idle connection pooling, got max=%d perHost=%d", transport.MaxIdleConns, transport.MaxIdleConnsPerHost)
	}
	if transport.IdleConnTimeout == 0 || transport.TLSHandshakeTimeout == 0 || transport.ExpectContinueTimeout == 0 {
		t.Fatal("expected transport timeouts configured")
	}
}

func TestNewClient_WithEngine(t *testing.T) {
	svc := httpc.NewService("test")
	c := NewClient(WithEngine(NewHTTPEngine(svc)))
	if _, ok := c.engine.(*HTTPCEngine); !ok {
		t.Fatal("expected HTTPCEngine")
	}
}

func TestNewClient_WithHttpcAndTLS(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"secure":true}`))
	}))
	defer ts.Close()

	tlsCfg := &tls.Config{InsecureSkipVerify: true}
	tlsClient := &http.Client{Transport: &http.Transport{TLSClientConfig: tlsCfg}}
	svc := httpc.NewServiceWithClient("test-tls", tlsClient)
	c := NewClient(WithEngine(NewHTTPEngine(svc)))
	if _, ok := c.engine.(*HTTPCEngine); !ok {
		t.Fatal("expected HTTPCEngine")
	}
	resp, err := c.Do(ctx(t), NewRequest(ts.URL, http.MethodGet))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Errorf("expected success, error: %s", resp.Err)
	}
}
