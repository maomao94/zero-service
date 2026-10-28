package app

import (
	"errors"
	"testing"

	"zero-service/app/opengtw/internal/security"
	"zero-service/app/opengtw/internal/svc"
	"zero-service/app/opengtw/internal/types"
	"zero-service/model/gormmodel"
	"zero-service/third_party/extproto"

	xerrors "github.com/zeromicro/x/errors"
)

const revealLogicMasterKey = "0123456789abcdef0123456789abcdef"

func newRevealLogicContext(t *testing.T) *svc.ServiceContext {
	t.Helper()
	svcCtx, _ := newAdminTestContext(t)
	svcCtx.Config.SignKeyEncryptionKey = revealLogicMasterKey
	svcCtx.Config.SignKeyRevealEnabled = true
	return svcCtx
}

func seedRevealLogicApp(t *testing.T, svcCtx *svc.ServiceContext, appID string, signRequired bool) string {
	t.Helper()

	signKey, err := security.NewSignKey()
	if err != nil {
		t.Fatalf("new sign key: %v", err)
	}
	cipher, err := security.EncryptSignKey(revealLogicMasterKey, signKey)
	if err != nil {
		t.Fatalf("encrypt sign key: %v", err)
	}
	if err := svcCtx.AppStore.Create(t.Context(), &gormmodel.OpenApp{
		AppId:        appID,
		Name:         "测试应用",
		Status:       1,
		SignRequired: signRequired,
	}); err != nil {
		t.Fatalf("create app: %v", err)
	}
	if err := svcCtx.SecretStore.Create(t.Context(), &gormmodel.OpenAppSecret{
		AppId:         appID,
		SecretHash:    security.HashSecret("sec_test"),
		SignKeyCipher: cipher,
		Status:        1,
	}); err != nil {
		t.Fatalf("create secret: %v", err)
	}
	return signKey
}

func errorCode(err error) int {
	var codeMsg *xerrors.CodeMsg
	if errors.As(err, &codeMsg) && codeMsg != nil {
		return codeMsg.Code
	}
	return -1
}

func TestRevealSignKeyLogic(t *testing.T) {
	svcCtx := newRevealLogicContext(t)
	const appID = "app_0123456789abcdef0123456789abcdef"
	want := seedRevealLogicApp(t, svcCtx, appID, true)

	resp, err := NewRevealSignKeyLogic(t.Context(), svcCtx).RevealSignKey(&types.RevealSignKeyRequest{AppId: appID})
	if err != nil {
		t.Fatalf("reveal: %v", err)
	}
	if resp.AppId != appID || resp.SignKey != want || !resp.SignRequired {
		t.Fatalf("unexpected reply: %+v", resp)
	}
	if resp.RevealedAt == "" {
		t.Fatal("revealedAt should not be empty")
	}
}

func TestRevealSignKeyLogicRejectsDisabledFeature(t *testing.T) {
	svcCtx := newRevealLogicContext(t)
	svcCtx.Config.SignKeyRevealEnabled = false

	_, err := NewRevealSignKeyLogic(t.Context(), svcCtx).RevealSignKey(&types.RevealSignKeyRequest{AppId: "app_x"})
	if code := errorCode(err); code != int(extproto.Code__1_00_PARAM_INVALID) {
		t.Fatalf("code = %d, want %d (err=%v)", code, extproto.Code__1_00_PARAM_INVALID, err)
	}
}

func TestRevealSignKeyLogicValidation(t *testing.T) {
	tests := []struct {
		name string
		req  *types.RevealSignKeyRequest
		code extproto.Code
	}{
		{name: "nil request", req: nil, code: extproto.Code__1_00_PARAM_MISSING},
		{name: "empty app id", req: &types.RevealSignKeyRequest{}, code: extproto.Code__1_00_PARAM_MISSING},
		{name: "app not found", req: &types.RevealSignKeyRequest{AppId: "app_missing"}, code: extproto.Code__1_00_RECORD_NOT_EXIST},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svcCtx := newRevealLogicContext(t)
			_, err := NewRevealSignKeyLogic(t.Context(), svcCtx).RevealSignKey(tt.req)
			if code := errorCode(err); code != int(tt.code) {
				t.Fatalf("code = %d, want %d (err=%v)", code, tt.code, err)
			}
		})
	}
}

func TestRevealSignKeyLogicRejectsUnsignedApp(t *testing.T) {
	svcCtx := newRevealLogicContext(t)
	const appID = "app_0123456789abcdef0123456789abcdef"
	seedRevealLogicApp(t, svcCtx, appID, false)

	_, err := NewRevealSignKeyLogic(t.Context(), svcCtx).RevealSignKey(&types.RevealSignKeyRequest{AppId: appID})
	if code := errorCode(err); code != int(extproto.Code__1_00_PARAM_INVALID) {
		t.Fatalf("code = %d, want %d (err=%v)", code, extproto.Code__1_00_PARAM_INVALID, err)
	}
}

func TestRevealSignKeyLogicReportsDecryptFailureAsInternal(t *testing.T) {
	svcCtx := newRevealLogicContext(t)
	const appID = "app_0123456789abcdef0123456789abcdef"
	seedRevealLogicApp(t, svcCtx, appID, true)
	svcCtx.Config.SignKeyEncryptionKey = "another-master-key-32-bytes-long!!"

	_, err := NewRevealSignKeyLogic(t.Context(), svcCtx).RevealSignKey(&types.RevealSignKeyRequest{AppId: appID})
	if code := errorCode(err); code != int(extproto.Code__1_00_INTERNAL) {
		t.Fatalf("code = %d, want %d (err=%v)", code, extproto.Code__1_00_INTERNAL, err)
	}
}
