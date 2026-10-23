package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	xhttp "github.com/zeromicro/x/http"
	"zero-service/third_party/extproto"
)

func TestMeetingAuthRejectsMissingIdentityWithEnvelope(t *testing.T) {
	m := NewMeetingAuthMiddleware(nil)
	handler := m.Handle(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not run without user identity")
	})

	req := httptest.NewRequest(http.MethodGet, "/live/v1/listMeetings", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("want HTTP 200, got %d", rr.Code)
	}
	var body xhttp.BaseResponse[any]
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not an envelope: %v, body=%s", err, rr.Body.String())
	}
	if body.Code != int(extproto.Code__1_00_UNAUTHORIZED) || body.Msg != "缺少用户身份" {
		t.Fatalf("unexpected body: %+v", body)
	}
}
