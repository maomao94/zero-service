package security

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	testAppID   = "app_0123456789abcdef0123456789abcdef"
	testSignKey = "sk_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

func signedRequestFixture() *SignedRequest {
	return &SignedRequest{
		AppID:     testAppID,
		Method:    "pod.list",
		Timestamp: 1761120000000,
		Nonce:     "nonce-001",
		TID:       "req-001",
		Data:      json.RawMessage(`{"limit":100,"offset":0}`),
	}
}

func TestCanonicalString(t *testing.T) {
	req := signedRequestFixture()
	const want = "appId=app_0123456789abcdef0123456789abcdef" +
		"&method=pod.list" +
		"&timestamp=1761120000000" +
		"&nonce=nonce-001" +
		"&tid=req-001" +
		"&dataHash=79cca47b16046efc521110b1af7825c6ea1893f0513b5314c7d4cc8cc4bed0fb" +
		"&path=/v1/gateway"
	if got := req.CanonicalString(testAppID, SignatureGatewayPath); got != want {
		t.Fatalf("canonical string = %q, want %q", got, want)
	}
}

func TestSignAndVerify(t *testing.T) {
	req := signedRequestFixture()
	req.Sign = req.ComputeSign(testAppID, SignatureGatewayPath, testSignKey)
	if req.Sign != "bb16d25c20f94ee46aa8db4b6ecb0b12585120374edaa347845488770d0e2900" {
		t.Fatalf("sign = %q", req.Sign)
	}

	now := time.UnixMilli(req.Timestamp)
	if err := req.Validate(testAppID, now); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := req.Verify(testAppID, SignatureGatewayPath, []string{testSignKey}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	// 重置密钥后的 24 小时窗口内会有多把 key，命中任意一把即可。
	if err := req.Verify(testAppID, SignatureGatewayPath, []string{"sk_old", testSignKey}); err != nil {
		t.Fatalf("verify with key ring: %v", err)
	}
	// 大写签名也应被接受（协议输出小写，这里做容错）。
	req.Sign = strings.ToUpper(req.Sign)
	if err := req.Verify(testAppID, SignatureGatewayPath, []string{testSignKey}); err != nil {
		t.Fatalf("verify uppercase sign: %v", err)
	}
}

func TestValidateRejects(t *testing.T) {
	now := time.UnixMilli(1761120000000)
	tests := []struct {
		name   string
		mutate func(req *SignedRequest)
		want   error
	}{
		{
			name: "app id mismatch",
			mutate: func(req *SignedRequest) {
				req.AppID = "app_other"
			},
			want: ErrSignatureMismatch,
		},
		{
			name: "expired timestamp",
			mutate: func(req *SignedRequest) {
				req.Timestamp -= int64(SignatureTimeWindow/time.Millisecond) + 1
			},
			want: ErrSignatureExpired,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := signedRequestFixture()
			req.Sign = req.ComputeSign(testAppID, SignatureGatewayPath, testSignKey)
			tt.mutate(req)
			if err := req.Validate(testAppID, now); !errors.Is(err, tt.want) {
				t.Fatalf("validate error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestVerifyRejects(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(req *SignedRequest)
		keys   []string
	}{
		{
			name:   "wrong key",
			mutate: func(req *SignedRequest) {},
			keys:   []string{"sk_wrong"},
		},
		{
			name: "tampered data",
			mutate: func(req *SignedRequest) {
				req.Data = json.RawMessage(`{"limit":1,"offset":0}`)
			},
			keys: []string{testSignKey},
		},
		{
			name: "tampered method",
			mutate: func(req *SignedRequest) {
				req.Method = "pod.delete"
			},
			keys: []string{testSignKey},
		},
		{
			name: "tampered tid",
			mutate: func(req *SignedRequest) {
				req.TID = "req-other"
			},
			keys: []string{testSignKey},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := signedRequestFixture()
			req.Sign = req.ComputeSign(testAppID, SignatureGatewayPath, testSignKey)
			tt.mutate(req)
			if err := req.Verify(testAppID, SignatureGatewayPath, tt.keys); !errors.Is(err, ErrSignatureMismatch) {
				t.Fatalf("verify error = %v, want ErrSignatureMismatch", err)
			}
		})
	}
}

func TestValidateMissingFields(t *testing.T) {
	tests := []string{"method", "timestamp", "nonce", "sign"}
	for _, field := range tests {
		t.Run(field, func(t *testing.T) {
			req := signedRequestFixture()
			req.Sign = req.ComputeSign(testAppID, SignatureGatewayPath, testSignKey)
			switch field {
			case "method":
				req.Method = ""
			case "timestamp":
				req.Timestamp = 0
			case "nonce":
				req.Nonce = ""
			case "sign":
				req.Sign = ""
			}
			if err := req.Validate(testAppID, time.Now()); !errors.Is(err, ErrSignatureMissing) {
				t.Fatalf("error = %v, want ErrSignatureMissing", err)
			}
		})
	}
}

func TestParseSignedRequestKeepsRawData(t *testing.T) {
	body := []byte(`{
		"appId": "app_0123456789abcdef0123456789abcdef",
		"method": "pod.list",
		"timestamp": 1761120000000,
		"nonce": "nonce-001",
		"tid": "req-001",
		"data": { "limit" : 100 },
		"sign": "abc"
	}`)
	req, err := ParseSignedRequest(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// dataHash 依赖原始字节：这里必须保留空格和键顺序，不能重新序列化。
	if got := string(req.Data); got != `{ "limit" : 100 }` {
		t.Fatalf("raw data = %q", got)
	}
}
