package livekitx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/livekit/protocol/livekit"
)

func TestNewRejectsMissingConfiguration(t *testing.T) {
	if _, err := New(); err == nil {
		t.Fatal("expected missing configuration error")
	}
}

func TestNewUsesConfiguredClientAndCloseIsIdempotent(t *testing.T) {
	httpClient := &http.Client{}
	client, err := New(
		WithURL("https://127.0.0.1:7880"),
		WithAPIKey("devkey", "secret"),
		WithHTTPClient(httpClient),
	)
	if err != nil {
		t.Fatal(err)
	}
	if client == nil || client.API() == nil {
		t.Fatal("expected initialized LiveKit client")
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestConfigDoesNotOwnRequestTimeout(t *testing.T) {
	if _, ok := any(Config{}).(interface{ RequestTimeout() }); ok {
		t.Fatal("Config must not expose a request timeout")
	}
}

type recordingHTTPService struct{ called bool }

func (s *recordingHTTPService) Do(context.Context, string, string, any) (*http.Response, error) {
	s.called = true
	return nil, nil
}

func (s *recordingHTTPService) DoRequest(r *http.Request) (*http.Response, error) {
	s.called = true
	return nil, errors.New("injected transport")
}

func TestNewUsesInjectedHTTPService(t *testing.T) {
	service := &recordingHTTPService{}
	client, err := New(WithURL("https://127.0.0.1:7880"), WithAPIKey("devkey", "secret"), WithHTTPService(service))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if client.Config().HTTPClient == nil {
		t.Fatal("expected injected transport to be retained")
	}
	_, _ = client.API().Room().ListRooms(context.Background(), nil)
	if !service.called {
		t.Fatal("expected management request to use injected HTTP service")
	}
}

// TestInternalTransportToleratesSelfSignedCert 验证未注入传输时 SDK 内部
// client 对自签证书（httptest TLS server）可直接调用，调用方无需安装证书。
func TestInternalTransportToleratesSelfSignedCert(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/protobuf")
		_, _ = w.Write([]byte{})
	}))
	defer srv.Close()

	client, err := New(WithURL(srv.URL), WithAPIKey("devkey", "secret"))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if client.Config().HTTPClient != nil {
		t.Fatal("expected no injected transport")
	}
	resp, err := client.API().Room().ListRooms(context.Background(), &livekit.ListRoomsRequest{})
	if err != nil {
		t.Fatalf("internal transport must tolerate self-signed cert: %v", err)
	}
	if resp == nil {
		t.Fatal("expected empty ListRoomsResponse")
	}
}
