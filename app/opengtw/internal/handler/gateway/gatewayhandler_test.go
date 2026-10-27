package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"zero-service/app/openapi/openapi"
	"zero-service/app/opengtw/internal/middleware"
	"zero-service/app/opengtw/internal/svc"
	"zero-service/third_party/extproto"

	xerrors "github.com/zeromicro/x/errors"
	xhttp "github.com/zeromicro/x/http"
	"google.golang.org/grpc"
)

type fakeOpenApiClient struct {
	req *openapi.InvokeReq
	res *openapi.InvokeRes
	err error
}

func (f *fakeOpenApiClient) Invoke(ctx context.Context, in *openapi.InvokeReq, opts ...grpc.CallOption) (*openapi.InvokeRes, error) {
	f.req = in
	return f.res, f.err
}

func newGatewayTestContext(client *fakeOpenApiClient) *svc.ServiceContext {
	return &svc.ServiceContext{OpenApiCli: client}
}

func doGatewayRequest(t *testing.T, client *fakeOpenApiClient, body string) *httptest.ResponseRecorder {
	t.Helper()
	svcCtx := newGatewayTestContext(client)
	handler := GatewayHandler(svcCtx)

	req := httptest.NewRequest(http.MethodPost, "/v1/gateway", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(middleware.WithAppID(req.Context(), "app_test"))
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func TestGatewaySuccessEnvelope(t *testing.T) {
	client := &fakeOpenApiClient{res: &openapi.InvokeRes{
		Data: `{"items":[],"total":0}`,
	}}
	rec := doGatewayRequest(t, client, `{"method":"pod.list","data":{"limit":100},"tid":"req-1"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Request-Id"); got != "req-1" {
		t.Fatalf("X-Request-Id = %q", got)
	}

	var body struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != 0 || body.Msg != "ok" {
		t.Fatalf("unexpected envelope: %+v", body)
	}
	if string(body.Data) != `{"items":[],"total":0}` {
		t.Fatalf("data = %s", body.Data)
	}
}

func TestGatewayKeepsRawData(t *testing.T) {
	client := &fakeOpenApiClient{res: &openapi.InvokeRes{Data: `{}`}}
	raw := `{"method":"pod.list","data":{"limit":"100","offset":0}}`
	rec := doGatewayRequest(t, client, raw)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if string(client.req.Data) != `{"limit":"100","offset":0}` {
		t.Fatalf("forwarded data = %s", client.req.Data)
	}
	if client.req.AppId != "app_test" {
		t.Fatalf("app id = %q", client.req.AppId)
	}
}

func TestGatewayUnknownMethod(t *testing.T) {
	client := &fakeOpenApiClient{err: xerrors.New(int(extproto.Code__1_30_METHOD_NOT_OPEN), "method 未开放")}
	rec := doGatewayRequest(t, client, `{"method":"pod.unknown","data":{}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body xhttp.BaseResponse[any]
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != int(extproto.Code__1_30_METHOD_NOT_OPEN) {
		t.Fatalf("business code = %d", body.Code)
	}
}

func TestGatewayFoldsInfrastructureError(t *testing.T) {
	client := &fakeOpenApiClient{err: xerrors.New(int(extproto.Code__1_00_RPC), "dial tcp: connection refused")}
	rec := doGatewayRequest(t, client, `{"method":"pod.list","data":{}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body xhttp.BaseResponse[any]
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != int(extproto.Code__1_00_UNKNOWN) || body.Msg != "系统繁忙" {
		t.Fatalf("unexpected folded error: %+v", body)
	}
}

func TestGatewayMissingMethod(t *testing.T) {
	client := &fakeOpenApiClient{}
	rec := doGatewayRequest(t, client, `{"data":{}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if client.req != nil {
		t.Fatal("openapi should not be called when method is missing")
	}
}

func TestGatewayInvalidJSONReturnsParseError(t *testing.T) {
	client := &fakeOpenApiClient{}
	rec := doGatewayRequest(t, client, `not-json`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body xhttp.BaseResponse[any]
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != int(extproto.Code__1_00_PARAM) {
		t.Fatalf("business code = %d, want %d", body.Code, extproto.Code__1_00_PARAM)
	}
	if client.req != nil {
		t.Fatal("openapi should not be called on invalid JSON")
	}
}

func TestHealthHandler(t *testing.T) {
	handler := HealthHandler(&svc.ServiceContext{})
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body xhttp.BaseResponse[struct {
		Status string `json:"status"`
	}]
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != 0 || body.Data.Status != "ok" {
		t.Fatalf("unexpected health response: %+v", body)
	}
}
