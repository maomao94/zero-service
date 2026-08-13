package authctx

import (
	"context"
	"reflect"
	"testing"
)

func TestContextKeyContractAndGetters(t *testing.T) {
	wantKeys := []string{"authorization", "user-id", "user-name", "dept-code", "auth-type"}
	if !reflect.DeepEqual(ContextKeys, wantKeys) {
		t.Fatalf("ContextKeys = %#v, want %#v", ContextKeys, wantKeys)
	}
	for _, key := range ContextKeys {
		if reflect.TypeOf(key).Kind() != reflect.String {
			t.Fatalf("context key %q has dynamic type %T, want string", key, key)
		}
	}

	// Typed setter/getter contract.
	ctx := context.Background()
	ctx = WithUserID(ctx, "user-1")
	ctx = WithUserName(ctx, "alice")
	ctx = WithDeptCode(ctx, "dept-1")
	ctx = WithAuthorization(ctx, "Bearer token")
	ctx = WithAuthType(ctx, "user")
	if got := GetUserId(ctx); got != "user-1" {
		t.Fatalf("GetUserId() = %q", got)
	}
	if got := GetUserName(ctx); got != "alice" {
		t.Fatalf("GetUserName() = %q", got)
	}
	if got := GetDeptCode(ctx); got != "dept-1" {
		t.Fatalf("GetDeptCode() = %q", got)
	}
	if got := GetAuthorization(ctx); got != "Bearer token" {
		t.Fatalf("GetAuthorization() = %q", got)
	}
	if got := GetAuthType(ctx); got != "user" {
		t.Fatalf("GetAuthType() = %q", got)
	}

	// String-key writes (go-zero JWT namespace) are NOT readable by getters:
	// typed keys only, no compatibility fallback.
	strCtx := context.Background()
	strCtx = context.WithValue(strCtx, string("user-id"), "string-1")
	strCtx = context.WithValue(strCtx, CtxUserNameKey, "string-alice")
	strCtx = context.WithValue(strCtx, CtxDeptCodeKey, "string-dept")
	strCtx = context.WithValue(strCtx, CtxAuthorizationKey, "String token")
	strCtx = context.WithValue(strCtx, CtxAuthTypeKey, "string-user")
	if got := GetUserId(strCtx); got != "" {
		t.Fatalf("string-key GetUserId() = %q, want empty (no fallback)", got)
	}
	if got := GetUserName(strCtx); got != "" {
		t.Fatalf("string-key GetUserName() = %q, want empty (no fallback)", got)
	}
	if got := GetDeptCode(strCtx); got != "" {
		t.Fatalf("string-key GetDeptCode() = %q, want empty (no fallback)", got)
	}
	if got := GetAuthorization(strCtx); got != "" {
		t.Fatalf("string-key GetAuthorization() = %q, want empty (no fallback)", got)
	}
	if got := GetAuthType(strCtx); got != "" {
		t.Fatalf("string-key GetAuthType() = %q, want empty (no fallback)", got)
	}

	// Non-string values yield "".
	if got := GetUserId(context.WithValue(context.Background(), CtxUserIdKey, 1)); got != "" {
		t.Fatalf("non-string GetUserId() = %q", got)
	}
	if got := GetAuthType(context.WithValue(context.Background(), CtxAuthTypeKey, 1)); got != "" {
		t.Fatalf("non-string GetAuthType() = %q", got)
	}

	// Wire-name dispatch helpers.
	byKey := WithKey(context.Background(), CtxDeptCodeKey, "dept-2")
	if got := GetByKey(byKey, CtxDeptCodeKey); got != "dept-2" {
		t.Fatalf("GetByKey(dept-code) = %q", got)
	}
	if got := GetByKey(byKey, "unknown"); got != "" {
		t.Fatalf("GetByKey(unknown) = %q", got)
	}
	if got := GetByKey(context.Background(), CtxUserIdKey); got != "" {
		t.Fatalf("GetByKey(missing) = %q", got)
	}
	if got := GetByKey(WithKey(context.Background(), "unknown", "x"), CtxUserIdKey); got != "" {
		t.Fatalf("WithKey(unknown) wrote a value: %q", got)
	}
}

func TestBridgeJWTClaims(t *testing.T) {
	t.Run("dash claim copied directly", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, string("user-id"), "u1")
		ctx = context.WithValue(ctx, string("user-name"), "alice")
		got := BridgeJWTClaims(ctx, nil)
		if v := GetUserId(got); v != "u1" {
			t.Fatalf("GetUserId() = %q", v)
		}
		if v := GetUserName(got); v != "alice" {
			t.Fatalf("GetUserName() = %q", v)
		}
	})

	t.Run("underscore claim mapped to internal typed key", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, string("user_id"), "u1")
		got := BridgeJWTClaims(ctx, map[string]string{CtxUserIdKey: "user_id"})
		if v := GetUserId(got); v != "u1" {
			t.Fatalf("GetUserId() = %q", v)
		}
	})

	t.Run("numeric user-id normalized to string", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			value any
			want  string
		}{
			{"int64", int64(42), "42"},
			{"int", 42, "42"},
			{"integer float64", float64(42), "42"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				ctx := context.WithValue(context.Background(), string("user-id"), tc.value)
				got := BridgeJWTClaims(ctx, nil)
				if v := GetUserId(got); v != tc.want {
					t.Fatalf("GetUserId() = %q, want %q", v, tc.want)
				}
			})
		}
	})

	t.Run("bool and array claims ignored", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, string("user-id"), true)
		ctx = context.WithValue(ctx, string("user-name"), []string{"a"})
		got := BridgeJWTClaims(ctx, nil)
		if v := GetUserId(got); v != "" {
			t.Fatalf("GetUserId() = %q, want empty", v)
		}
		if v := GetUserName(got); v != "" {
			t.Fatalf("GetUserName() = %q, want empty", v)
		}
	})

	t.Run("empty mapping and no claims leaves context unchanged", func(t *testing.T) {
		got := BridgeJWTClaims(context.Background(), nil)
		if v := GetUserId(got); v != "" {
			t.Fatalf("GetUserId() = %q", v)
		}
		if v := GetUserName(got); v != "" {
			t.Fatalf("GetUserName() = %q", v)
		}
		if v := GetAuthType(got); v != "" {
			t.Fatalf("GetAuthType() = %q", v)
		}
	})

	t.Run("existing typed value wins over string key", func(t *testing.T) {
		ctx := WithUserID(context.Background(), "typed-1")
		ctx = context.WithValue(ctx, string("user-id"), "string-1")
		got := BridgeJWTClaims(ctx, nil)
		if v := GetUserId(got); v != "typed-1" {
			t.Fatalf("GetUserId() = %q, want typed-1", v)
		}
	})

	t.Run("existing typed value wins over underscore mapping", func(t *testing.T) {
		ctx := WithUserID(context.Background(), "typed-1")
		ctx = context.WithValue(ctx, string("user_id"), "string-1")
		got := BridgeJWTClaims(ctx, map[string]string{CtxUserIdKey: "user_id"})
		if v := GetUserId(got); v != "typed-1" {
			t.Fatalf("GetUserId() = %q, want typed-1", v)
		}
	})
}
