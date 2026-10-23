package security

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

const (
	appIDPrefix  = "app_"
	secretPrefix = "sec_"
)

// NewAppID 生成 app_ + 12 位 hex 的应用 ID。
func NewAppID() (string, error) {
	return prefixedHex(appIDPrefix, 6)
}

// NewSecret 生成 sec_ + 32 位 hex 的密钥。
func NewSecret() (string, error) {
	return prefixedHex(secretPrefix, 16)
}

// HashSecret 返回密钥的 sha256 十六进制摘要。
func HashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// VerifySecret 以常数时间比较密钥与摘要。
func VerifySecret(hash, secret string) bool {
	expected, err := hex.DecodeString(hash)
	if err != nil || len(expected) != sha256.Size {
		return false
	}
	actual := sha256.Sum256([]byte(secret))
	return subtle.ConstantTimeCompare(expected, actual[:]) == 1
}

func prefixedHex(prefix string, size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(buf), nil
}
