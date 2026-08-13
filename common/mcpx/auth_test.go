package mcpx

import (
	"context"
	"testing"
	"time"

	"zero-service/common/authctx"

	"github.com/golang-jwt/jwt/v4"
)

func TestDualTokenVerifierJWTPropagationContract(t *testing.T) {
	const secret = "test-secret"
	claims := jwt.MapClaims{
		"exp":       float64(time.Now().Add(time.Hour).Unix()),
		"user_id":   float64(42),
		"user-name": "alice",
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	verifier := NewDualTokenVerifier(
		[]string{secret},
		"",
		map[string]string{authctx.CtxUserIdKey: "user_id"},
	)
	info, err := verifier(context.Background(), token, nil)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if info.UserID != "42" {
		t.Fatalf("UserID = %q, want %q", info.UserID, "42")
	}
	if got := info.Extra[authctx.CtxUserIdKey]; got != float64(42) {
		t.Fatalf("mapped user claim = %#v", got)
	}
	if got := info.Extra[authctx.CtxUserNameKey]; got != "alice" {
		t.Fatalf("user-name claim = %#v", got)
	}
	if got := info.Extra[authctx.CtxAuthTypeKey]; got != "user" {
		t.Fatalf("auth-type = %#v", got)
	}
	if got := info.Extra[authctx.CtxAuthorizationKey]; got != token {
		t.Fatalf("authorization = %#v, want original token", got)
	}
}
