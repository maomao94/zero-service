package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"zero-service/common/gormx"
	"zero-service/model/gormmodel"
	"zero-service/openapp/opengtw/internal/middleware"
	"zero-service/openapp/opengtw/internal/svc"
	"zero-service/openapp/opengtw/internal/types"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	xhttp "github.com/zeromicro/x/http"
)

func newAdminTestContext(t *testing.T) (*svc.ServiceContext, *middleware.CredentialCache) {
	t.Helper()
	db := gormx.MustOpenWithConf(gormx.Config{
		DataSource:   "file:" + t.Name() + "?mode=memory&cache=shared&parseTime=true&_loc=auto",
		MaxIdleConns: 1,
		MaxOpenConns: 1,
	})
	db.MustAutoMigrate(&gormmodel.OpenApp{}, &gormmodel.OpenAppSecret{})

	appStore := gormmodel.NewOpenAppStore(db)
	secretStore := gormmodel.NewOpenAppSecretStore(db)
	cache := middleware.NewCredentialCache(appStore, secretStore, time.Hour)
	cache.MustRefresh(t.Context())

	return &svc.ServiceContext{
		DB:              db,
		AppStore:        appStore,
		SecretStore:     secretStore,
		CredentialCache: cache,
	}, cache
}

func authenticateWithCache(cache *middleware.CredentialCache, appID, secret string) (int, int) {
	called := false
	mw := middleware.NewOpenAuthMiddleware(cache)
	handler := mw.Handle(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/gateway", nil)
	req.SetBasicAuth(appID, secret)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if called {
		return rec.Code, 0
	}
	var body xhttp.BaseResponse[any]
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		return rec.Code, -1
	}
	return rec.Code, body.Code
}

func TestAdminFlowCreateCallResetDisable(t *testing.T) {
	svcCtx, cache := newAdminTestContext(t)
	ctx := t.Context()
	logger := logx.WithContext(ctx)

	created, err := NewCreateAppLogic(ctx, svcCtx).CreateApp(&types.CreateAppRequest{Name: "示例应用", Remark: "podman"})
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	if !strings.HasPrefix(created.AppId, "app_") || len(created.AppId) != 16 {
		t.Fatalf("unexpected app id: %q", created.AppId)
	}
	if !strings.HasPrefix(created.Secret, "sec_") || len(created.Secret) != 36 {
		t.Fatalf("unexpected secret: %q", created.Secret)
	}
	if created.Status != 1 {
		t.Fatalf("status = %d, want 1", created.Status)
	}

	if status, code := authenticateWithCache(cache, created.AppId, created.Secret); status != 200 || code != 0 {
		t.Fatalf("auth after create = %d/%d", status, code)
	}

	oldSecret := created.Secret
	reset, err := NewResetAppSecretLogic(ctx, svcCtx).ResetAppSecret(&types.ResetAppSecretRequest{AppId: created.AppId})
	if err != nil {
		t.Fatalf("reset secret: %v", err)
	}
	if reset.Secret == oldSecret {
		t.Fatal("new secret should differ from old one")
	}
	if reset.OldSecretExpireAt == "" {
		t.Fatal("old secret expire time should not be empty")
	}

	if status, code := authenticateWithCache(cache, created.AppId, reset.Secret); status != 200 || code != 0 {
		t.Fatalf("new secret auth = %d/%d", status, code)
	}
	if status, code := authenticateWithCache(cache, created.AppId, oldSecret); status != 200 || code != 0 {
		t.Fatalf("old secret should still work within 24h, got %d/%d", status, code)
	}

	list, err := NewListAppsLogic(ctx, svcCtx).ListApps(&types.ListAppsRequest{})
	if err != nil {
		t.Fatalf("list apps: %v", err)
	}
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].AppId != created.AppId {
		t.Fatalf("unexpected list: %+v", list)
	}

	if _, err := NewDisableAppLogic(ctx, svcCtx).DisableApp(&types.DisableAppRequest{AppId: created.AppId, Status: 2}); err != nil {
		t.Fatalf("disable app: %v", err)
	}
	for _, secret := range []string{oldSecret, reset.Secret} {
		status, code := authenticateWithCache(cache, created.AppId, secret)
		if status != http.StatusOK || code != int(extproto.Code__1_30_APP_NOT_FOUND) {
			t.Fatalf("auth after disable = %d/%d", status, code)
		}
	}

	logger.Info("admin flow ok")
}

func TestAdminFlowValidation(t *testing.T) {
	svcCtx, _ := newAdminTestContext(t)
	ctx := t.Context()

	if _, err := NewCreateAppLogic(ctx, svcCtx).CreateApp(&types.CreateAppRequest{}); err == nil {
		t.Fatal("empty name should fail")
	}
	if _, err := NewDisableAppLogic(ctx, svcCtx).DisableApp(&types.DisableAppRequest{AppId: "app_missing", Status: 3}); err == nil {
		t.Fatal("invalid status should fail")
	}
	if _, err := NewDisableAppLogic(ctx, svcCtx).DisableApp(&types.DisableAppRequest{AppId: "app_missing", Status: 2}); err == nil {
		t.Fatal("missing app should fail")
	}
	if _, err := NewResetAppSecretLogic(ctx, svcCtx).ResetAppSecret(&types.ResetAppSecretRequest{AppId: "app_missing"}); err == nil {
		t.Fatal("reset missing app should fail")
	}
}
