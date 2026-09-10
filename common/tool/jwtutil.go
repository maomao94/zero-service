package tool

import (
	"crypto/subtle"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var (
	ErrInvalidSignKey   = errors.New("invalid sign key")
	ErrSignKeyNotConfig = errors.New("server sign key not configured")
)

// 标准 JWT claims，不允许被自定义 payload 覆盖
const (
	jwtAudience  = "aud"
	jwtExpire    = "exp"
	jwtId        = "jti"
	jwtIssueAt   = "iat"
	jwtIssuer    = "iss"
	jwtNotBefore = "nbf"
	jwtSubject   = "sub"
)

// GenerateUserToken 签发用户域 Token
// secretKey: JWT 签名密钥
// expireSeconds: 过期时间（秒）
// userId: 用户 ID
// userName: 用户名称（可选）
// deptCode: 部门编码（可选）
func GenerateUserToken(secretKey string, expireSeconds int64, userId, userName, deptCode string) (token string, accessExpire, refreshAfter int64, err error) {
	if len(userId) == 0 {
		return "", 0, 0, errors.New("userId is required")
	}
	now := time.Now().Unix()
	expireAt := now + expireSeconds

	claims := jwt.MapClaims{
		"exp":       expireAt,
		"iat":       now,
		"user-id":   userId,
		"auth-type": "user",
	}
	if userName != "" {
		claims["user-name"] = userName
	}
	if deptCode != "" {
		claims["dept-code"] = deptCode
	}

	tokenString, err := signToken(secretKey, claims)
	if err != nil {
		return "", 0, 0, err
	}
	return tokenString, expireAt, now + expireSeconds/2, nil
}

// GenerateDeviceToken 签发设备域 Token
// secretKey: JWT 签名密钥
// expireSeconds: 过期时间（秒）
// deviceId: 设备 ID
// deviceName: 设备名称（可选）
func GenerateDeviceToken(secretKey string, expireSeconds int64, deviceId, deviceName string) (token string, accessExpire, refreshAfter int64, err error) {
	if len(deviceId) == 0 {
		return "", 0, 0, errors.New("deviceId is required")
	}
	now := time.Now().Unix()
	expireAt := now + expireSeconds

	claims := jwt.MapClaims{
		"exp":       expireAt,
		"iat":       now,
		"user-id":   deviceId,
		"auth-type": "device",
	}
	if deviceName != "" {
		claims["user-name"] = deviceName
	}

	tokenString, err := signToken(secretKey, claims)
	if err != nil {
		return "", 0, 0, err
	}
	return tokenString, expireAt, now + expireSeconds/2, nil
}

// GenerateTokenByMap 使用自定义 claims 签发 Token
// secretKey: JWT 签名密钥
// expireSeconds: 过期时间（秒）
// claims: 自定义 claims（自动添加 exp/iat；标准 claims 会被忽略，防止覆盖）
func GenerateTokenByMap(secretKey string, expireSeconds int64, claims map[string]string) (token string, accessExpire, refreshAfter int64, err error) {
	now := time.Now().Unix()
	expireAt := now + expireSeconds

	jwtClaims := jwt.MapClaims{
		jwtExpire:  expireAt,
		jwtIssueAt: now,
	}
	for k, v := range claims {
		if k == "" {
			continue
		}
		switch k {
		case jwtAudience, jwtExpire, jwtId, jwtIssueAt, jwtIssuer, jwtNotBefore, jwtSubject:
			// 忽略标准 claims，防止被 payload 覆盖
		default:
			jwtClaims[k] = v
		}
	}

	tokenString, err := signToken(secretKey, jwtClaims)
	if err != nil {
		return "", 0, 0, err
	}
	return tokenString, expireAt, now + expireSeconds/2, nil
}

// VerifySignKey 验证签发密钥（常量时间比较，防止时序攻击）
// 返回 true 表示密钥匹配
func VerifySignKey(serverKey, clientKey string) bool {
	if len(serverKey) == 0 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(serverKey), []byte(clientKey)) == 1
}

// signToken 签发 JWT Token
func signToken(secretKey string, claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}
