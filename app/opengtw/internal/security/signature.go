package security

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/cryptor"
)

const (
	// SignatureTimeWindow 是允许的客户端与服务端时间差；超过窗口的请求即使签名正确也会被拒绝。
	SignatureTimeWindow = 5 * time.Minute
	// SignatureGatewayPath 是开放面统一转发接口的路径，作为 canonical string 的 path 字段。
	SignatureGatewayPath = "/v1/gateway"
	// SignatureVersion 是当前签名协议版本；协议变更时递增，canonical 第一行随之变化。
	SignatureVersion = "v1"
	// SignatureMaxNonceLength 限制 nonce 长度，避免 Redis key 过大。
	SignatureMaxNonceLength = 64
)

var (
	// ErrSignatureMissing 表示缺少 appId/timestamp/nonce/sign 等必填签名要素。
	ErrSignatureMissing = errors.New("签名参数缺失")
	// ErrSignatureExpired 表示请求时间戳超出允许窗口。
	ErrSignatureExpired = errors.New("签名时间戳超出允许窗口")
	// ErrSignatureMismatch 表示签名与任一有效 signKey 都不匹配。
	ErrSignatureMismatch = errors.New("签名不匹配")
	// ErrSignatureVersion 表示客户端使用了服务端不支持的签名协议版本。
	ErrSignatureVersion = errors.New("签名版本不支持")
)

// SignatureInput 是一次请求的签名要素。
//
// timestamp、nonce 位于 body 中，由 bodyHash 覆盖；这里保留它们只用于时间窗和
// nonce 格式校验，不再重复进入 canonical，避免同一信息签两次。
type SignatureInput struct {
	// AppID 来自 Basic 鉴权用户名，是服务端可信值。
	AppID string
	// Timestamp 是 body 中的客户端毫秒时间戳，服务端允许 ±5 分钟。
	Timestamp int64
	// Nonce 是 body 中的客户端随机串，服务端用 Redis 占用，防止同一请求被重放。
	Nonce string
	// Path 是请求实际路径，当前固定为 /v1/gateway。
	Path string
	// BodyHash 是原始 body 字节的 SHA-256 小写十六进制摘要。
	BodyHash string
	// Sign 是 HMAC-SHA256(canonical, signKey) 的小写十六进制结果。
	Sign string
}

// ParseSignHeader 解析 X-Sign 头的 "v1:<hex>" 格式；省略版本时按 v1 处理。
func ParseSignHeader(value string) (version, sign string, err error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", ErrSignatureMissing
	}
	if idx := strings.IndexByte(value, ':'); idx >= 0 {
		// 版本是协议 token，按大小写不敏感处理，避免 V1/v1 这种无意义失败。
		version = strings.ToLower(strings.TrimSpace(value[:idx]))
		sign = strings.TrimSpace(value[idx+1:])
	} else {
		version = SignatureVersion
		sign = value
	}
	if version == "" || sign == "" {
		return "", "", ErrSignatureMissing
	}
	return version, sign, nil
}

// Validate 校验必填项、nonce 格式与时间窗。
//
// appId 来自 Basic 鉴权；timestamp、nonce 来自 body，且已被 bodyHash 覆盖。
func (s *SignatureInput) Validate(now time.Time) error {
	if s == nil {
		return ErrSignatureMissing
	}
	if s.AppID == "" || s.Timestamp <= 0 || s.Nonce == "" || s.Path == "" || s.BodyHash == "" || s.Sign == "" {
		return ErrSignatureMissing
	}
	if !validNonce(s.Nonce) {
		return fmt.Errorf("%w: nonce 必须是 1-%d 位字母、数字、点、下划线或短横线", ErrSignatureMissing, SignatureMaxNonceLength)
	}
	delta := now.Sub(time.UnixMilli(s.Timestamp))
	if delta > SignatureTimeWindow || delta < -SignatureTimeWindow {
		return ErrSignatureExpired
	}
	return nil
}

// CanonicalString 生成待签名的规范化字符串。
//
// 协议固定为 v1，字段之间用换行分隔：
//
//	v1
//	{appId}
//	{path}
//	{bodyHash}
//
// body 中的 method/data/tid/timestamp/nonce 都由 bodyHash 覆盖，不需要重复出现在
// canonical 中；以后 body 新增字段也会自动被签名。
func (s *SignatureInput) CanonicalString() string {
	return strings.Join([]string{
		SignatureVersion,
		s.AppID,
		s.Path,
		s.BodyHash,
	}, "\n")
}

// HashBody 返回原始 body 的 SHA-256 小写十六进制摘要。
//
// 客户端必须对“实际发送的 body 字节”计算摘要，而不是重新序列化后的 JSON；
// 服务端读取原始 body 后直接计算，因此不依赖 data 的字段顺序或空白格式。
func HashBody(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// ComputeSign 用 signKey 对请求做 HMAC-SHA256，返回小写十六进制签名。
//
// 客户端可以复用本方法生成 sign；服务端解密出 signKey 后重新计算 HMAC 比对。
func (s *SignatureInput) ComputeSign(signKey string) string {
	return cryptor.HmacSha256(s.CanonicalString(), signKey)
}

// Verify 校验签名是否命中任意一把有效 signKey。
//
// signKeys 允许多把：重置密钥后的 24 小时窗口内，新旧签名密钥同时有效。
// 比较使用 subtle.ConstantTimeCompare，避免通过响应时间猜测签名。
func (s *SignatureInput) Verify(signKeys []string) error {
	if s == nil || s.Sign == "" {
		return ErrSignatureMissing
	}
	// 签名统一按小写十六进制传输；这里兼容客户端误发大写的情况。
	expected := []byte(strings.ToLower(strings.TrimSpace(s.Sign)))
	for _, signKey := range signKeys {
		if signKey == "" {
			continue
		}
		actual := []byte(s.ComputeSign(signKey))
		if subtle.ConstantTimeCompare(actual, expected) == 1 {
			return nil
		}
	}
	return ErrSignatureMismatch
}

// validNonce 限制 nonce 为安全字符集，保证换行分隔的 canonical string 无歧义。
func validNonce(nonce string) bool {
	if len(nonce) == 0 || len(nonce) > SignatureMaxNonceLength {
		return false
	}
	for i := 0; i < len(nonce); i++ {
		c := nonce[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.', c == '_', c == '-':
		default:
			return false
		}
	}
	return true
}
