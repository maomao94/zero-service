package svc

import "testing"

func TestHasIdentity(t *testing.T) {
	tests := []struct {
		name   string
		claims map[string]any
		want   bool
	}{
		{"canonical", map[string]any{"user-id": "u1"}, true},
		{"snake", map[string]any{"user_id": "u1"}, true},
		{"camel", map[string]any{"userId": "u1"}, true},
		{"uid", map[string]any{"uid": "u1"}, true},
		{"numeric", map[string]any{"user_id": float64(42)}, true},
		{"device token", map[string]any{"user-id": "dev-1", "auth-type": "device"}, true},
		{"missing identity", map[string]any{"auth-type": "user"}, false},
		{"empty identity", map[string]any{"user_id": ""}, false},
		{"nil claims", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasIdentity(tt.claims); got != tt.want {
				t.Fatalf("hasIdentity(%#v) = %t, want %t", tt.claims, got, tt.want)
			}
		})
	}
}
