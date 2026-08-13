package authctx

import (
	"context"
	"encoding/json"
	"testing"
)

func TestClaimsContract(t *testing.T) {
	claims := map[string]any{
		"external_user": float64(42),
		CtxUserNameKey:  "alice",
		CtxDeptCodeKey:  1.5,
		CtxAuthTypeKey:  true,
	}
	ApplyClaimMapping(claims, map[string]string{CtxUserIdKey: "external_user"})
	if got := claims[CtxUserIdKey]; got != float64(42) {
		t.Fatalf("mapped claim = %#v", got)
	}

	ctx := ExtractFromClaims(context.Background(), claims)
	if got := GetUserId(ctx); got != "42" {
		t.Fatalf("numeric user id = %q", got)
	}
	if got := GetDeptCode(ctx); got != "1.5" {
		t.Fatalf("fractional claim = %q", got)
	}
	if got := GetAuthType(ctx); got != "" {
		t.Fatalf("bool claim = %#v, want empty", got)
	}
}

func TestClaimMappingDoesNotDeleteOrSynthesizeClaims(t *testing.T) {
	claims := map[string]any{CtxUserIdKey: "original"}
	ApplyClaimMapping(claims, map[string]string{CtxUserIdKey: "missing"})
	if got := claims[CtxUserIdKey]; got != "original" {
		t.Fatalf("missing external key changed claim to %#v", got)
	}
	if got := ClaimString(claims, "missing"); got != "" {
		t.Fatalf("missing ClaimString() = %q", got)
	}
	if got := ClaimString(map[string]any{"nil": nil}, "nil"); got != "" {
		t.Fatalf("nil ClaimString() = %q, want empty", got)
	}
}

func TestNormalizeClaimString(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		// string passes through.
		{"string", "42", "42"},
		// Signed integer types are exact base-10.
		{"int", int(42), "42"},
		{"int8", int8(42), "42"},
		{"int16", int16(42), "42"},
		{"int32", int32(42), "42"},
		{"int64", int64(42), "42"},
		{"negative int", int(-42), "-42"},
		// Unsigned integer types are exact base-10.
		{"uint", uint(42), "42"},
		{"uint8", uint8(42), "42"},
		{"uint16", uint16(42), "42"},
		{"uint32", uint32(42), "42"},
		{"uint64", uint64(42), "42"},
		// Floats use convertor.ToString (user ids are normally small integers,
		// so fractional/oversized values are simply converted, not rejected).
		{"float32 integer", float32(42), "42"},
		{"float64 integer", float64(42), "42"},
		{"float64 negative integer", float64(-42), "-42"},
		{"float64 boundary 2^53", float64(9007199254740992), "9007199254740992"},
		{"float64 fractional", float64(1.5), "1.5"},
		{"float64 oversized", float64(1e19), "10000000000000000000"},
		// json.Number (from the token parser) converts via convertor.ToString;
		// legal literals keep their exact decimal form.
		{"json.Number integer", json.Number("42"), "42"},
		{"json.Number exact large", json.Number("9007199254740993"), "9007199254740993"},
		{"json.Number fractional", json.Number("1.5"), "1.5"},
		{"json.Number negative", json.Number("-42"), "-42"},
		{"json.Number exponent", json.Number("1e3"), "1e3"},
		{"json.Number non-numeric", json.Number("abc"), ""},
		{"json.Number empty", json.Number(""), "0"},
		// Everything else is skipped.
		{"bool", true, ""},
		{"slice", []string{"a"}, ""},
		{"map", map[string]any{"k": "v"}, ""},
		{"nil", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeClaimString(tt.value); got != tt.want {
				t.Fatalf("normalizeClaimString(%#v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
