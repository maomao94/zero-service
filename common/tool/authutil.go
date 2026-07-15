package tool

import (
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v4"
)

func stripBearerPrefixFromTokenString(tok string) (string, error) {
	// Should be a bearer token
	if len(tok) > 6 && strings.ToUpper(tok[0:7]) == "BEARER " {
		return tok[7:], nil
	}
	return tok, nil
}

// ParseToken 解析并验证JWT token，支持所有签名算法，与go-zero保持一致
func ParseToken(tokenString string, secrets ...string) (jwt.MapClaims, error) {
	if len(secrets) == 0 {
		return nil, fmt.Errorf("at least one secret is required")
	}
	tokenString, tokenErr := stripBearerPrefixFromTokenString(tokenString)
	if tokenErr != nil {
		return nil, tokenErr
	}
	var lastErr error
	for _, secret := range secrets {
		token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil {
			lastErr = err
			continue
		}
		if token.Valid {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				return claims, nil
			}
		}
		lastErr = fmt.Errorf("invalid token")
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("invalid token")
}
