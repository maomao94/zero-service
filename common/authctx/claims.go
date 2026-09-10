package authctx

import (
	"context"
	"encoding/json"

	"github.com/duke-git/lancet/v2/convertor"
)

// ExtractFromClaims writes non-empty authentication claims to typed context keys.
func ExtractFromClaims(ctx context.Context, claims map[string]any) context.Context {
	if len(claims) == 0 {
		return ctx
	}
	for _, key := range ContextKeys {
		if v := ClaimString(claims, key); v != "" {
			ctx = WithKey(ctx, key, v)
		}
	}
	return ctx
}

// ApplyClaimMapping copies each external JWT claim (sourceKey) onto the standard
// internal key (targetKey) inside claims. Direction: targetKey <- sourceKey,
// matching the ClaimMapping config (standard key -> external JWT claim key).
func ApplyClaimMapping(claims map[string]any, mapping map[string]string) {
	for targetKey, sourceKey := range mapping {
		if v, ok := claims[sourceKey]; ok {
			claims[targetKey] = v
		}
	}
}

// DefaultClaimAliases maps each standard internal key to the external JWT claim
// names that may carry it. Alias lookup order: canonical key first, then aliases
// in the listed order.
var DefaultClaimAliases = map[string][]string{
	CtxUserIdKey:   {"user_id", "userId", "uid"},
	CtxUserNameKey: {"user_name"},
	CtxDeptCodeKey: {"dept_code"},
	CtxAuthTypeKey: {},
}

// ResolveClaimKey returns the canonical internal key for an external claim name
// (canonical or alias). It returns "" for unknown names.
func ResolveClaimKey(name string) string {
	for _, key := range ContextKeys {
		if name == key {
			return key
		}
	}
	for canonical, aliases := range DefaultClaimAliases {
		for _, alias := range aliases {
			if name == alias {
				return canonical
			}
		}
	}
	return ""
}

// ClaimStringByAliases reads the first non-empty claim among the canonical key
// and its aliases.
func ClaimStringByAliases(claims map[string]any, canonicalKey string) string {
	if v := ClaimString(claims, canonicalKey); v != "" {
		return v
	}
	for _, alias := range DefaultClaimAliases[canonicalKey] {
		if v := ClaimString(claims, alias); v != "" {
			return v
		}
	}
	return ""
}

// ClaimString converts a JWT claim value to its string form. Strings and integer
// types are exact; numeric json.Number/float values use convertor.ToString.
// Missing keys and non-identity types (bool/array/map/…) yield "".
func ClaimString(claims map[string]any, key string) string {
	v, ok := claims[key]
	if !ok {
		return ""
	}
	return normalizeClaimString(v)
}

// normalizeClaimString converts a JWT identity claim value to string via
// convertor.ToString. Only string and numeric types (incl. json.Number from the
// token parser) are accepted; other types (bool/array/map/…) are ignored to keep
// malformed values out of identity keys.
func normalizeClaimString(v any) string {
	switch v.(type) {
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, json.Number:
		return convertor.ToString(v)
	default:
		return ""
	}
}
