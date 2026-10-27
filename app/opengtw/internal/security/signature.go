package security

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/cryptor"
)

const (
	// SignatureTimeWindow 是允许的客户端与服务端时间差；超过窗口的请求即使签名正确也会被拒绝。
	SignatureTimeWindow = 5 * time.Minute
	// SignatureGatewayPath 是开放面统一转发接口的路径，作为 canonical string 的 path 字段。
	SignatureGatewayPath = "/v1/gateway"
)

var (
	// ErrSignatureMissing 表示缺少 appId/method/timestamp/nonce/sign 等必填签名要素。
	ErrSignatureMissing = errors.New("签名参数缺失")
	// ErrSignatureExpired 表示请求时间戳超出允许窗口。
	ErrSignatureExpired = errors.New("签名时间戳超出允许窗口")
	// ErrSignatureMismatch 表示签名与任一有效 signKey 都不匹配。
	ErrSignatureMismatch = errors.New("签名不匹配")
)

// SignedRequest 是 POST /v1/gateway 请求体中与签名相关的字段。
//
// 这里刻意独立定义一份结构体，而不是复用 goctl 生成的 types.GatewayRequest：
// 签名是跨语言协议，字段名、缺省行为和顺序都需要稳定；goctl 重新生成不会影响本结构。
//
// Data 使用 json.RawMessage，保留客户端实际发送的 JSON 字节（包括键顺序和空白），
// 因为 dataHash 必须对原始字节计算。先反序列化成 map 再重新序列化会改变字节，导致签名不一致。
type SignedRequest struct {
	// AppID 可选；填写时必须与 Basic 鉴权用户名完全一致（含 app_ 前缀）。
	AppID string `json:"appId,omitempty"`
	// Method 是开放 method，例如 pod.list。
	Method string `json:"method"`
	// Timestamp 是客户端毫秒时间戳，服务端允许 ±5 分钟。
	Timestamp int64 `json:"timestamp,omitempty"`
	// Nonce 是客户端随机串，服务端用 Redis 占用，防止同一请求被重放。
	Nonce string `json:"nonce,omitempty"`
	// TID 是链路 ID；未传时网关会生成，但签名时空串也参与 canonical。
	TID string `json:"tid,omitempty"`
	// Data 是该 method 的 protojson 原文；空值等价于 {}，但签名时按实际字节计算。
	Data json.RawMessage `json:"data,omitempty"`
	// Sign 是 HMAC-SHA256(canonical, signKey) 的小写十六进制结果。
	Sign string `json:"sign,omitempty"`
}

// ParseSignedRequest 解析 HTTP body，保留 Data 的原始字节。
func ParseSignedRequest(body []byte) (*SignedRequest, error) {
	var req SignedRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("解析签名请求失败: %w", err)
	}
	return &req, nil
}

// Validate 校验签名必填项、appId 一致性和时间窗。
//
// appID 来自 Basic 鉴权用户名，是服务端可信值；body 里的 appId 只是冗余校验，
// 即使客户端不传，canonical string 也始终使用 Basic 用户名。
func (r *SignedRequest) Validate(appID string, now time.Time) error {
	if r == nil {
		return ErrSignatureMissing
	}
	if r.AppID != "" && r.AppID != appID {
		return fmt.Errorf("%w: appId 与 Basic 凭据不一致", ErrSignatureMismatch)
	}
	if r.Method == "" || r.Timestamp <= 0 || r.Nonce == "" || r.Sign == "" {
		return ErrSignatureMissing
	}
	delta := now.Sub(time.UnixMilli(r.Timestamp))
	if delta > SignatureTimeWindow || delta < -SignatureTimeWindow {
		return ErrSignatureExpired
	}
	return nil
}

// CanonicalString 生成待签名的规范化字符串。
//
// 顺序和分隔符是协议的一部分，客户端与服务端必须完全一致：
//
//	appId={appID}&method={method}&timestamp={timestamp}&nonce={nonce}&tid={tid}&dataHash={sha256(data)}&path={path}
//
// 注意 appID 使用服务端鉴权得到的值，path 使用请求实际路径（当前固定为 /v1/gateway）。
func (r *SignedRequest) CanonicalString(appID, path string) string {
	return "appId=" + appID +
		"&method=" + r.Method +
		"&timestamp=" + strconv.FormatInt(r.Timestamp, 10) +
		"&nonce=" + r.Nonce +
		"&tid=" + r.TID +
		"&dataHash=" + HashData(string(r.Data)) +
		"&path=" + path
}

// HashData 返回 data 原文的 SHA-256 小写十六进制摘要。
func HashData(data string) string {
	return cryptor.Sha256(data)
}

// ComputeSign 用 signKey 对请求做 HMAC-SHA256，返回小写十六进制签名。
//
// 客户端可以复用本方法生成 sign；服务端不落库明文 signKey 的哈希，
// 而是解密出 signKey 后重新计算 HMAC 比对。
func (r *SignedRequest) ComputeSign(appID, path, signKey string) string {
	return cryptor.HmacSha256(r.CanonicalString(appID, path), signKey)
}

// Verify 校验签名是否命中任意一把有效 signKey。
//
// signKeys 允许多把：重置密钥后的 24 小时窗口内，新旧签名密钥同时有效。
// 比较使用 subtle.ConstantTimeCompare，避免通过响应时间猜测签名。
func (r *SignedRequest) Verify(appID, path string, signKeys []string) error {
	if r == nil || r.Sign == "" {
		return ErrSignatureMissing
	}
	// 签名统一按小写十六进制传输；这里兼容客户端误发大写的情况。
	expected := []byte(strings.ToLower(strings.TrimSpace(r.Sign)))
	for _, signKey := range signKeys {
		if signKey == "" {
			continue
		}
		actual := []byte(r.ComputeSign(appID, path, signKey))
		if subtle.ConstantTimeCompare(actual, expected) == 1 {
			return nil
		}
	}
	return ErrSignatureMismatch
}
