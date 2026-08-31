package livekitx

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestNewRejectsMissingConfiguration(t *testing.T) {
	if _, err := New(); err == nil {
		t.Fatal("expected missing configuration error")
	}
}

func TestNewUsesConfiguredClientAndCloseIsIdempotent(t *testing.T) {
	httpClient := &http.Client{}
	client, err := New(
		WithURL("http://127.0.0.1:7880"),
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
	client, err := New(WithURL("http://127.0.0.1:7880"), WithAPIKey("devkey", "secret"), WithHTTPService(service))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if client.Config().HTTPService != service {
		t.Fatal("expected injected HTTP service to be retained")
	}
	_, _ = client.API().Room().ListRooms(context.Background(), nil)
	if !service.called {
		t.Fatal("expected management request to use injected HTTP service")
	}
}
