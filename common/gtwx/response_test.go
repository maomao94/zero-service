package gtwx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"zero-service/common/djisdk"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	xerrors "github.com/zeromicro/x/errors"
	xhttp "github.com/zeromicro/x/http"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func decodeEnvelope(t *testing.T, rr *httptest.ResponseRecorder) xhttp.BaseResponse[any] {
	t.Helper()
	var body xhttp.BaseResponse[any]
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not an envelope: %v, body=%s", err, rr.Body.String())
	}
	return body
}

func TestWriteErrorAlwaysHTTP200(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantMsg  string
	}{
		{name: "code msg", err: xerrors.New(130102, "密钥无效"), wantCode: 130102, wantMsg: "密钥无效"},
		{name: "grpc unauthorized", err: tool.NewErrorByPbCode(extproto.Code__1_00_UNAUTHORIZED), wantCode: 100401, wantMsg: "未认证"},
		{name: "grpc not found", err: tool.NewErrorByPbCode(extproto.Code__1_00_RECORD_NOT_EXIST), wantCode: 100202, wantMsg: "记录不存在"},
		{name: "fold internal", err: xerrors.New(int(extproto.Code__1_00_INTERNAL), "db password leaked"), wantCode: 100999, wantMsg: "系统繁忙"},
		{name: "plain error", err: errors.New("boom"), wantCode: 100999, wantMsg: "系统繁忙"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			WriteError(context.Background(), rr, tc.err)

			if rr.Code != http.StatusOK {
				t.Fatalf("want HTTP 200, got %d", rr.Code)
			}
			body := decodeEnvelope(t, rr)
			if body.Code != tc.wantCode || body.Msg != tc.wantMsg {
				t.Fatalf("want code=%d msg=%q, got code=%d msg=%q", tc.wantCode, tc.wantMsg, body.Code, body.Msg)
			}
		})
	}
}

func TestWriteErrorNeverEmitsSuccessCode(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteError(context.Background(), rr, xerrors.New(0, "mistaken success"))

	body := decodeEnvelope(t, rr)
	if body.Code == 0 {
		t.Fatal("error response must not carry success code 0")
	}
}

// 未在 extproto 登记的码不属于网关信封的错误码空间：设备结果由业务字段
// （如 CommonRes.reasonCode）承载，第三方平台故障由适配层映射成平台码。
func TestWriteErrorFoldsVendorCode(t *testing.T) {
	djiErr := djisdk.NewDJIError(312022)
	rr := httptest.NewRecorder()

	WriteError(context.Background(), rr, fmt.Errorf("command failed: %w", djiErr))

	if rr.Code != http.StatusOK {
		t.Fatalf("want HTTP 200, got %d", rr.Code)
	}
	body := decodeEnvelope(t, rr)
	if body.Code != int(extproto.Code__1_00_UNKNOWN) || body.Msg != "系统繁忙" {
		t.Fatalf("want 100999/系统繁忙, got code=%d msg=%q", body.Code, body.Msg)
	}
}

func TestWriteErrorFoldsVendorCodeOverGRPC(t *testing.T) {
	// 模拟跨 gRPC 后只剩 ErrorInfo.reason 的场景：312022 不在 extproto 中。
	st, err := status.New(codes.FailedPrecondition, "设备升级中").
		WithDetails(&errdetails.ErrorInfo{Reason: "312022", Domain: "dji"})
	if err != nil {
		t.Fatalf("build status: %v", err)
	}
	rr := httptest.NewRecorder()

	WriteError(context.Background(), rr, st.Err())

	body := decodeEnvelope(t, rr)
	if body.Code != int(extproto.Code__1_00_UNKNOWN) || body.Msg != "系统繁忙" {
		t.Fatalf("want 100999/系统繁忙, got code=%d msg=%q", body.Code, body.Msg)
	}
}

// 开发者自定义的码只要不在 extproto 中，就必须降级为 100999。
func TestWriteErrorRejectsUndefinedCode(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteError(context.Background(), rr, xerrors.New(100555, "made up"))

	body := decodeEnvelope(t, rr)
	if body.Code != int(extproto.Code__1_00_UNKNOWN) {
		t.Fatalf("want 100999, got %d", body.Code)
	}
}

// 开放平台码必须原样返回，不能被降级。
func TestWriteErrorKeepsOpenPlatformCode(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteError(context.Background(), rr, xerrors.New(int(extproto.Code__1_30_APP_NOT_FOUND), "应用不存在或已停用"))

	body := decodeEnvelope(t, rr)
	if body.Code != int(extproto.Code__1_30_APP_NOT_FOUND) {
		t.Fatalf("want %d, got %d", extproto.Code__1_30_APP_NOT_FOUND, body.Code)
	}
}

func TestWriteOK(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteOK(context.Background(), rr, map[string]string{"status": "ok"})

	if rr.Code != http.StatusOK {
		t.Fatalf("want HTTP 200, got %d", rr.Code)
	}
	body := decodeEnvelope(t, rr)
	if body.Code != 0 || body.Msg != "ok" {
		t.Fatalf("unexpected success envelope: %+v", body)
	}
	var response struct {
		Timestamp int64 `json:"timestamp"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode timestamp: %v", err)
	}
	if response.Timestamp <= 0 {
		t.Fatalf("timestamp should be positive, got %d", response.Timestamp)
	}
}

func TestGatewayFallbackHandlers(t *testing.T) {
	cases := []struct {
		name     string
		handler  http.HandlerFunc
		wantCode int
	}{
		{name: "not found", handler: NotFoundHandler, wantCode: int(extproto.Code__1_00_ROUTE_NOT_FOUND)},
		{name: "method not allowed", handler: MethodNotAllowedHandler, wantCode: int(extproto.Code__1_00_METHOD_NOT_ALLOWED)},
		{name: "unauthorized", handler: func(w http.ResponseWriter, r *http.Request) {
			UnauthorizedHandler(w, r, errors.New("jwt expired"))
		}, wantCode: int(extproto.Code__1_00_UNAUTHORIZED)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			rr := httptest.NewRecorder()
			tc.handler(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("want HTTP 200, got %d", rr.Code)
			}
			if rr.Header().Get("X-Request-Id") == "" {
				t.Fatal("X-Request-Id should be set")
			}
			body := decodeEnvelope(t, rr)
			if body.Code != tc.wantCode {
				t.Fatalf("want code=%d, got %d", tc.wantCode, body.Code)
			}
		})
	}
}

func TestEnsureRequestIDReusesIncomingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Request-Id", "req-from-client")
	rr := httptest.NewRecorder()

	ensureRequestID(rr, req)

	if got := rr.Header().Get("X-Request-Id"); got != "req-from-client" {
		t.Fatalf("want incoming request id, got %q", got)
	}
}
