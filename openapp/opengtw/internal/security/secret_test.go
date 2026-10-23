package security

import (
	"strings"
	"testing"
)

func TestNewAppIDAndSecret(t *testing.T) {
	appID, err := NewAppID()
	if err != nil {
		t.Fatalf("NewAppID error = %v", err)
	}
	if !strings.HasPrefix(appID, "app_") || len(appID) != len("app_")+12 {
		t.Fatalf("unexpected app id: %q", appID)
	}

	secret, err := NewSecret()
	if err != nil {
		t.Fatalf("NewSecret error = %v", err)
	}
	if !strings.HasPrefix(secret, "sec_") || len(secret) != len("sec_")+32 {
		t.Fatalf("unexpected secret: %q", secret)
	}

	other, err := NewSecret()
	if err != nil {
		t.Fatalf("NewSecret error = %v", err)
	}
	if secret == other {
		t.Fatal("two secrets should not be equal")
	}
}

func TestHashAndVerifySecret(t *testing.T) {
	secret := "sec_0123456789abcdef0123456789abcdef"
	hash := HashSecret(secret)
	if len(hash) != 64 {
		t.Fatalf("hash length = %d", len(hash))
	}
	if !VerifySecret(hash, secret) {
		t.Fatal("VerifySecret should succeed")
	}
	if VerifySecret(hash, secret+"x") {
		t.Fatal("VerifySecret should fail for another secret")
	}
	if VerifySecret("not-a-hash", secret) {
		t.Fatal("VerifySecret should fail for invalid hash")
	}
}
