package livekitx

import (
	"testing"
	"time"

	"github.com/livekit/protocol/auth"
)

func TestNewJoinTokenRequiresRoomAndIdentity(t *testing.T) {
	if _, err := NewJoinToken(JoinTokenOptions{APIKey: "devkey", APISecret: "secret"}); err == nil {
		t.Fatal("expected required identity and room error")
	}
}

func TestNewJoinTokenCreatesMinimalJoinGrant(t *testing.T) {
	token, err := NewJoinToken(JoinTokenOptions{
		APIKey:       "devkey",
		APISecret:    "secret",
		Room:         "room-a",
		Identity:     "user-a",
		Name:         "User A",
		ValidFor:     time.Hour,
		CanPublish:   true,
		CanSubscribe: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("expected signed token")
	}
	verifier, err := auth.ParseAPIToken(token)
	if err != nil {
		t.Fatal(err)
	}
	claims, grants, err := verifier.Verify("secret")
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "user-a" || grants.Video.Room != "room-a" || !grants.Video.RoomJoin {
		t.Fatalf("unexpected claims: %+v, grants: %+v", claims, grants)
	}
	if !grants.Video.GetCanPublish() || !grants.Video.GetCanSubscribe() {
		t.Fatal("expected publish and subscribe permissions")
	}
}

func TestNewJoinTokenDefaults(t *testing.T) {
	token, err := NewJoinToken(JoinTokenOptions{
		APIKey:    "devkey",
		APISecret: "secret",
		Room:      "room-a",
		Identity:  "user-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := auth.ParseAPIToken(token)
	if err != nil {
		t.Fatal(err)
	}
	claims, _, err := verifier.Verify("secret")
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "user-a" {
		t.Fatalf("unexpected subject: %s", claims.Subject)
	}
}
