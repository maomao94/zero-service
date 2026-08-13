package grpcx

import (
	"context"
	"encoding/base64"
	"reflect"
	"strings"
	"testing"

	"zero-service/common/authctx"

	"google.golang.org/grpc/metadata"
)

func TestMetadataFieldContract(t *testing.T) {
	want := []metadataField{
		{authctx.CtxAuthorizationKey, "authorization"},
		{authctx.CtxUserIdKey, "x-user-id"},
		{authctx.CtxUserNameKey, "x-user-name"},
		{authctx.CtxDeptCodeKey, "x-dept-code"},
		{authctx.CtxAuthTypeKey, "x-auth-type"},
	}
	if !reflect.DeepEqual(metadataFields, want) {
		t.Fatalf("metadataFields = %#v, want %#v", metadataFields, want)
	}
	for _, field := range metadataFields {
		if field.grpcKey != strings.ToLower(field.grpcKey) {
			t.Errorf("gRPC metadata key %q is not lowercase", field.grpcKey)
		}
	}
}

func TestMetadataRoundTripAllFields(t *testing.T) {
	values := []string{"Bearer token", "user-1", "用户名", "dept-1", "user"}
	ctx := context.Background()
	for i, field := range metadataFields {
		ctx = authctx.WithKey(ctx, field.contextKey, values[i])
	}
	outCtx := InjectToGrpcMD(ctx)
	md, ok := metadata.FromOutgoingContext(outCtx)
	if !ok {
		t.Fatal("outgoing metadata missing")
	}
	encodedUserName := "b64:" + base64.StdEncoding.EncodeToString([]byte("用户名"))
	if got := md.Get(HeaderUserName); !reflect.DeepEqual(got, []string{encodedUserName}) {
		t.Fatalf("encoded user name = %#v", got)
	}

	inCtx := metadata.NewIncomingContext(context.Background(), md)
	roundTrip := ExtractFromGrpcMD(inCtx)
	for i, field := range metadataFields {
		if got := authctx.GetByKey(roundTrip, field.contextKey); got != values[i] {
			t.Errorf("context[%q] = %#v, want %q", field.contextKey, got, values[i])
		}
	}
}

func TestMetadataOverwriteAndFirstValueContract(t *testing.T) {
	original := metadata.Pairs(HeaderAuthorization, "old", "existing", "preserved")
	ctx := metadata.NewOutgoingContext(context.Background(), original)
	ctx = authctx.WithAuthorization(ctx, "new")
	md, _ := metadata.FromOutgoingContext(InjectToGrpcMD(ctx))
	if got := md.Get(HeaderAuthorization); !reflect.DeepEqual(got, []string{"new"}) {
		t.Fatalf("authorization overwrite = %#v", got)
	}
	if got := md.Get("existing"); !reflect.DeepEqual(got, []string{"preserved"}) {
		t.Fatalf("existing metadata = %#v", got)
	}
	if got := original.Get(HeaderAuthorization); !reflect.DeepEqual(got, []string{"old"}) {
		t.Fatalf("source metadata was mutated: %#v", got)
	}

	incoming := metadata.NewIncomingContext(context.Background(), metadata.MD{
		HeaderUserId:   []string{"first", "second"},
		HeaderDeptCode: []string{"", "ignored"},
	})
	extracted := ExtractFromGrpcMD(incoming)
	if got := authctx.GetUserId(extracted); got != "first" {
		t.Fatalf("first metadata value = %q", got)
	}
	if got := authctx.GetDeptCode(extracted); got != "" {
		t.Fatalf("empty first value unexpectedly fell through: %q", got)
	}
}

func TestMetadataFilteringAndPrintableContract(t *testing.T) {
	ctx := context.Background()
	ctx = authctx.WithUserID(ctx, "")
	ctx = context.WithValue(ctx, string("user-name"), 42)
	ctx = authctx.WithDeptCode(ctx, "line\nbreak")
	md, _ := metadata.FromOutgoingContext(InjectToGrpcMD(ctx))
	if got := md.Get(HeaderUserId); len(got) != 0 {
		t.Fatalf("empty value was propagated: %#v", got)
	}
	if got := md.Get(HeaderUserName); len(got) != 0 {
		t.Fatalf("non-string value was propagated: %#v", got)
	}
	want := "b64:" + base64.StdEncoding.EncodeToString([]byte("line\nbreak"))
	if got := md.Get(HeaderDeptCode); !reflect.DeepEqual(got, []string{want}) {
		t.Fatalf("control-character encoding = %#v, want %q", got, want)
	}
}
