package security

import "testing"

func TestSignKeyEncryptionRoundTrip(t *testing.T) {
	master := "change-me-sign-key-encryption-key-32-bytes-minimum"
	plain := "sk_test_signing_key"
	ciphertext, err := EncryptSignKey(master, plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	got, err := DecryptSignKey(master, ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != plain {
		t.Fatalf("got %q, want %q", got, plain)
	}
}

func TestSignKeyEncryptionRejectsTampering(t *testing.T) {
	master := "change-me-sign-key-encryption-key-32-bytes-minimum"
	ciphertext, err := EncryptSignKey(master, "sk_test_signing_key")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	last := ciphertext[len(ciphertext)-1]
	if last == '0' {
		last = '1'
	} else {
		last = '0'
	}
	tampered := ciphertext[:len(ciphertext)-1] + string(last)
	if _, err := DecryptSignKey(master, tampered); err == nil {
		t.Fatal("tampered ciphertext should fail authentication")
	}
}
