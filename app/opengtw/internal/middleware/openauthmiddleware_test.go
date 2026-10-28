package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"zero-service/app/opengtw/internal/security"
	"zero-service/common/gormx"
	"zero-service/model/gormmodel"
	"zero-service/third_party/extproto"

	xhttp "github.com/zeromicro/x/http"
)

func openAuthTestDB(t *testing.T) (*gormx.DB, *gormmodel.OpenAppStore, *gormmodel.OpenAppSecretStore) {
	t.Helper()
	db := gormx.MustOpenWithConf(gormx.Config{
		DataSource:   "file:" + t.Name() + "?mode=memory&cache=shared&parseTime=true&_loc=auto",
		MaxIdleConns: 1,
		MaxOpenConns: 1,
	})
	db.MustAutoMigrate(&gormmodel.OpenApp{}, &gormmodel.OpenAppSecret{})
	return db, gormmodel.NewOpenAppStore(db), gormmodel.NewOpenAppSecretStore(db)
}

func seedApp(t *testing.T, db *gormx.DB, appID, name string, status int, secrets ...*gormmodel.OpenAppSecret) {
	t.Helper()
	if err := db.Create(&gormmodel.OpenApp{AppId: appID, Name: name, Status: status}).Error; err != nil {
		t.Fatalf("create app: %v", err)
	}
	for _, secret := range secrets {
		secret.AppId = appID
		if err := db.Create(secret).Error; err != nil {
			t.Fatalf("create secret: %v", err)
		}
	}
}

func doAuth(t *testing.T, cache *CredentialCache, appID, secret string, withBasic bool) (*httptest.ResponseRecorder, bool, string) {
	t.Helper()
	called := false
	appIDInContext := ""
	mw := NewOpenAuthMiddleware(cache, "", nil)
	handler := mw.Handle(func(w http.ResponseWriter, r *http.Request) {
		called = true
		appIDInContext = AppIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/gateway", nil)
	if withBasic {
		req.SetBasicAuth(appID, secret)
	}
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec, called, appIDInContext
}

func decodeResponseCode(t *testing.T, rec *httptest.ResponseRecorder) int {
	t.Helper()
	var body xhttp.BaseResponse[any]
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	return body.Code
}

func TestOpenAuthMiddleware(t *testing.T) {
	db, appStore, secretStore := openAuthTestDB(t)
	now := time.Now()
	activeSecret := "sec_active"
	oldSecret := "sec_old_valid"
	expiredSecret := "sec_expired"

	seedApp(t, db, "app_active", "active", 1,
		&gormmodel.OpenAppSecret{SecretHash: security.HashSecret(activeSecret), Status: 1},
		&gormmodel.OpenAppSecret{SecretHash: security.HashSecret(oldSecret), Status: 2, ExpireAt: now.Add(time.Hour)},
		&gormmodel.OpenAppSecret{SecretHash: security.HashSecret(expiredSecret), Status: 2, ExpireAt: now.Add(-time.Hour)},
	)
	seedApp(t, db, "app_disabled", "disabled", 2,
		&gormmodel.OpenAppSecret{SecretHash: security.HashSecret("sec_disabled"), Status: 1},
	)

	cache := NewCredentialCache(appStore, secretStore, time.Hour)
	cache.MustRefresh(context.Background())

	tests := []struct {
		name       string
		appID      string
		secret     string
		withBasic  bool
		wantStatus int
		wantCode   int
		wantCalled bool
	}{
		{name: "missing credentials", wantStatus: 200, wantCode: int(extproto.Code__1_30_APP_SECRET_INVALID)},
		{name: "unknown app", appID: "app_unknown", secret: "x", withBasic: true, wantStatus: 200, wantCode: int(extproto.Code__1_30_APP_NOT_FOUND)},
		{name: "disabled app", appID: "app_disabled", secret: "sec_disabled", withBasic: true, wantStatus: 200, wantCode: int(extproto.Code__1_30_APP_NOT_FOUND)},
		{name: "wrong secret", appID: "app_active", secret: "sec_wrong", withBasic: true, wantStatus: 200, wantCode: int(extproto.Code__1_30_APP_SECRET_INVALID)},
		{name: "expired secret", appID: "app_active", secret: expiredSecret, withBasic: true, wantStatus: 200, wantCode: int(extproto.Code__1_30_APP_SECRET_INVALID)},
		{name: "active secret", appID: "app_active", secret: activeSecret, withBasic: true, wantStatus: 200, wantCode: 0, wantCalled: true},
		{name: "retained old secret", appID: "app_active", secret: oldSecret, withBasic: true, wantStatus: 200, wantCode: 0, wantCalled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, called, appIDInContext := doAuth(t, cache, tt.appID, tt.secret, tt.withBasic)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if called != tt.wantCalled {
				t.Fatalf("called = %v, want %v", called, tt.wantCalled)
			}
			if tt.wantCalled && appIDInContext != tt.appID {
				t.Fatalf("app id in context = %q, want %q", appIDInContext, tt.appID)
			}
			if !tt.wantCalled {
				if code := decodeResponseCode(t, rec); code != tt.wantCode {
					t.Fatalf("business code = %d, want %d", code, tt.wantCode)
				}
			}
		})
	}
}

func TestCredentialCacheRefreshDropsDisabledApp(t *testing.T) {
	db, appStore, secretStore := openAuthTestDB(t)
	seedApp(t, db, "app_active", "active", 1,
		&gormmodel.OpenAppSecret{SecretHash: security.HashSecret("sec_a"), Status: 1})

	cache := NewCredentialCache(appStore, secretStore, time.Hour)
	cache.MustRefresh(context.Background())
	if _, ok := cache.Get("app_active"); !ok {
		t.Fatal("active app should be cached")
	}

	if err := db.Model(&gormmodel.OpenApp{}).Where("app_id = ?", "app_active").Update("status", 2).Error; err != nil {
		t.Fatalf("disable app: %v", err)
	}
	cache.MustRefresh(context.Background())
	if _, ok := cache.Get("app_active"); ok {
		t.Fatal("disabled app should be removed from cache")
	}
}
