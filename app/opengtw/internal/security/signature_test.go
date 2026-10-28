package security

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const (
	testAppID     = "app_0123456789abcdef0123456789abcdef"
	testSignKey   = "sk_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	testBody      = `{"method":"pod.list","data":{"limit":100,"offset":0},"tid":"req-001","timestamp":1761120000000,"nonce":"nonce-001"}`
	testTimestamp = int64(1761120000000)
	testNonce     = "nonce-001"
	testBodyHash  = "ba0ebcbe2c20ea9befd138f072721e89b2592800080f04e3f04efd52839f2a7d"
	testSign      = "a21688f6f7bd082ead7fa5b1a34eb48899210c7702d808d6413e3197cbea13ba"
)

func signatureFixture() *SignatureInput {
	return &SignatureInput{
		AppID:     testAppID,
		Timestamp: testTimestamp,
		Nonce:     testNonce,
		Path:      SignatureGatewayPath,
		BodyHash:  HashBody([]byte(testBody)),
	}
}

func TestHashBody(t *testing.T) {
	if got := HashBody([]byte(testBody)); got != testBodyHash {
		t.Fatalf("body hash = %q, want %q", got, testBodyHash)
	}
	if got := HashBody(nil); got != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatalf("empty body hash = %q", got)
	}
}

func TestParseSignHeader(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		wantVersion string
		wantSign    string
		wantErr     error
	}{
		{name: "versioned", value: "v1:abc", wantVersion: "v1", wantSign: "abc"},
		{name: "uppercase version", value: "V1:abc", wantVersion: "v1", wantSign: "abc"},
		{name: "bare sign defaults to v1", value: "abc", wantVersion: "v1", wantSign: "abc"},
		{name: "spaces", value: " v1 : abc ", wantVersion: "v1", wantSign: "abc"},
		{name: "empty", value: "", wantErr: ErrSignatureMissing},
		{name: "empty sign", value: "v1:", wantErr: ErrSignatureMissing},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, sign, err := ParseSignHeader(tt.value)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if version != tt.wantVersion || sign != tt.wantSign {
				t.Fatalf("got %q/%q, want %q/%q", version, sign, tt.wantVersion, tt.wantSign)
			}
		})
	}
}

func TestCanonicalString(t *testing.T) {
	const want = "v1\n" +
		"app_0123456789abcdef0123456789abcdef\n" +
		"/v1/gateway\n" +
		"ba0ebcbe2c20ea9befd138f072721e89b2592800080f04e3f04efd52839f2a7d"
	if got := signatureFixture().CanonicalString(); got != want {
		t.Fatalf("canonical string = %q, want %q", got, want)
	}
}

func TestSignAndVerify(t *testing.T) {
	req := signatureFixture()
	req.Sign = req.ComputeSign(testSignKey)
	if req.Sign != testSign {
		t.Fatalf("sign = %q, want %q", req.Sign, testSign)
	}

	now := time.UnixMilli(req.Timestamp)
	if err := req.Validate(now); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := req.Verify([]string{testSignKey}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	// 重置密钥后的 24 小时窗口内会有多把 key，命中任意一把即可。
	if err := req.Verify([]string{"sk_old", testSignKey}); err != nil {
		t.Fatalf("verify with key ring: %v", err)
	}
	// 大写签名也应被接受（协议输出小写，这里做容错）。
	req.Sign = strings.ToUpper(req.Sign)
	if err := req.Verify([]string{testSignKey}); err != nil {
		t.Fatalf("verify uppercase sign: %v", err)
	}
}

func TestValidateRejects(t *testing.T) {
	now := time.UnixMilli(testTimestamp)
	tests := []struct {
		name   string
		mutate func(req *SignatureInput)
		want   error
	}{
		{
			name:   "expired timestamp",
			mutate: func(req *SignatureInput) { req.Timestamp -= int64(SignatureTimeWindow/time.Millisecond) + 1 },
			want:   ErrSignatureExpired,
		},
		{
			name:   "future timestamp",
			mutate: func(req *SignatureInput) { req.Timestamp += int64(SignatureTimeWindow/time.Millisecond) + 1 },
			want:   ErrSignatureExpired,
		},
		{
			name:   "nonce with separator",
			mutate: func(req *SignatureInput) { req.Nonce = "nonce&path=/v1/gateway" },
			want:   ErrSignatureMissing,
		},
		{
			name:   "missing app id",
			mutate: func(req *SignatureInput) { req.AppID = "" },
			want:   ErrSignatureMissing,
		},
		{
			name:   "missing timestamp",
			mutate: func(req *SignatureInput) { req.Timestamp = 0 },
			want:   ErrSignatureMissing,
		},
		{
			name:   "missing nonce",
			mutate: func(req *SignatureInput) { req.Nonce = "" },
			want:   ErrSignatureMissing,
		},
		{
			name:   "missing sign",
			mutate: func(req *SignatureInput) { req.Sign = "" },
			want:   ErrSignatureMissing,
		},
		{
			name:   "missing body hash",
			mutate: func(req *SignatureInput) { req.BodyHash = "" },
			want:   ErrSignatureMissing,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := signatureFixture()
			req.Sign = req.ComputeSign(testSignKey)
			tt.mutate(req)
			if err := req.Validate(now); !errors.Is(err, tt.want) {
				t.Fatalf("validate error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestVerifyRejects(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(req *SignatureInput)
		keys   []string
	}{
		{
			name:   "wrong key",
			mutate: func(req *SignatureInput) {},
			keys:   []string{"sk_wrong"},
		},
		{
			name:   "tampered body",
			mutate: func(req *SignatureInput) { req.BodyHash = HashBody([]byte(`{"method":"pod.delete"}`)) },
			keys:   []string{testSignKey},
		},
		{
			name:   "tampered app id",
			mutate: func(req *SignatureInput) { req.AppID = "app_other" },
			keys:   []string{testSignKey},
		},
		{
			name:   "tampered path",
			mutate: func(req *SignatureInput) { req.Path = "/v1/other" },
			keys:   []string{testSignKey},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := signatureFixture()
			req.Sign = req.ComputeSign(testSignKey)
			tt.mutate(req)
			if err := req.Verify(tt.keys); !errors.Is(err, ErrSignatureMismatch) {
				t.Fatalf("verify error = %v, want ErrSignatureMismatch", err)
			}
		})
	}
}
