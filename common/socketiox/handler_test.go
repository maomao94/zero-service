package socketiox

import (
	"net/http"
	"testing"
)

func TestIsNewSocketConnection(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		target  string
		headers map[string]string
		want    bool
	}{
		{
			name:   "polling handshake",
			method: http.MethodGet,
			target: "/socket.io/?EIO=4&transport=polling",
			want:   true,
		},
		{
			name:   "polling post is not a new connection",
			method: http.MethodPost,
			target: "/socket.io/?EIO=4&transport=polling",
			want:   false,
		},
		{
			name:   "polling with sid reuses session",
			method: http.MethodGet,
			target: "/socket.io/?EIO=4&transport=polling&sid=abc",
			want:   false,
		},
		{
			name:    "websocket direct",
			method:  http.MethodGet,
			target:  "/socket.io/?EIO=4&transport=websocket",
			headers: map[string]string{"Upgrade": "websocket", "Connection": "Upgrade"},
			want:    true,
		},
		{
			name:    "websocket with sid is an upgrade",
			method:  http.MethodGet,
			target:  "/socket.io/?EIO=4&transport=websocket&sid=abc",
			headers: map[string]string{"Upgrade": "websocket", "Connection": "Upgrade"},
			want:    false,
		},
		{
			name:   "websocket without upgrade header",
			method: http.MethodGet,
			target: "/socket.io/?EIO=4&transport=websocket",
			want:   false,
		},
		{
			name:   "unknown transport",
			method: http.MethodGet,
			target: "/socket.io/?EIO=4&transport=xhr",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := http.NewRequest(tt.method, tt.target, nil)
			if err != nil {
				t.Fatal(err)
			}
			for k, v := range tt.headers {
				r.Header.Set(k, v)
			}
			if got := isNewSocketConnection(r); got != tt.want {
				t.Fatalf("isNewSocketConnection = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNewSocketConnectionNil(t *testing.T) {
	if isNewSocketConnection(nil) {
		t.Fatal("nil request should not be treated as a new connection")
	}
}
