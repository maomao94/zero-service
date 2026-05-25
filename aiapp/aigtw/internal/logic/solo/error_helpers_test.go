package solo

import (
	"errors"
	"net/http"
	"testing"

	"zero-service/common/gtwx"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestInvalidRequestErrorIsOpenAIHTTP400(t *testing.T) {
	err := invalidRequestError("bad request")

	var openAIErr *gtwx.OpenAIError
	if !errors.As(err, &openAIErr) {
		t.Fatalf("invalidRequestError() = %T, want *gtwx.OpenAIError", err)
	}
	if openAIErr.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("HTTPStatus = %d, want %d", openAIErr.HTTPStatus, http.StatusBadRequest)
	}
	if openAIErr.ErrorMsg.Type != "invalid_request_error" || openAIErr.ErrorMsg.Code != "invalid_request" {
		t.Fatalf("OpenAI error = %#v, want invalid_request_error/invalid_request", openAIErr.ErrorMsg)
	}
}

func TestUnauthenticatedErrorIsGrpcStatus(t *testing.T) {
	err := unauthenticatedError("missing user id")

	if code := status.Code(err); code != codes.Unauthenticated {
		t.Fatalf("status.Code() = %v, want %v", code, codes.Unauthenticated)
	}
	if got := err.Error(); got == "missing user id" {
		t.Fatalf("unauthenticatedError() returned plain error string %q", got)
	}
}
