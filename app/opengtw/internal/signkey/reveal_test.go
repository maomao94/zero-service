package signkey

import (
	"context"
	"errors"
	"testing"

	"zero-service/app/opengtw/internal/security"
	"zero-service/common/gormx"
	"zero-service/model/gormmodel"
)

const (
	revealTestMasterKey = "0123456789abcdef0123456789abcdef"
	revealTestAppID     = "app_0123456789abcdef0123456789abcdef"
)

func newRevealStores(t *testing.T) (*gormmodel.OpenAppStore, *gormmodel.OpenAppSecretStore) {
	t.Helper()
	db := gormx.MustOpenWithConf(gormx.Config{
		DataSource:   "file:" + t.Name() + "?mode=memory&cache=shared&parseTime=true&_loc=auto",
		MaxIdleConns: 1,
		MaxOpenConns: 1,
	})
	db.MustAutoMigrate(&gormmodel.OpenApp{}, &gormmodel.OpenAppSecret{})
	return gormmodel.NewOpenAppStore(db), gormmodel.NewOpenAppSecretStore(db)
}

func seedRevealApp(t *testing.T, appStore *gormmodel.OpenAppStore, secretStore *gormmodel.OpenAppSecretStore, appID string, signRequired bool, masterKey string) string {
	t.Helper()

	signKey, err := security.NewSignKey()
	if err != nil {
		t.Fatalf("new sign key: %v", err)
	}
	cipher, err := security.EncryptSignKey(masterKey, signKey)
	if err != nil {
		t.Fatalf("encrypt sign key: %v", err)
	}
	if err := appStore.Create(t.Context(), &gormmodel.OpenApp{
		AppId:        appID,
		Name:         "测试应用",
		Status:       1,
		SignRequired: signRequired,
	}); err != nil {
		t.Fatalf("create app: %v", err)
	}
	if err := secretStore.Create(t.Context(), &gormmodel.OpenAppSecret{
		AppId:         appID,
		SecretHash:    security.HashSecret("sec_test"),
		SignKeyCipher: cipher,
		Status:        1,
	}); err != nil {
		t.Fatalf("create secret: %v", err)
	}
	return signKey
}

func TestRevealReturnsActiveSignKey(t *testing.T) {
	appStore, secretStore := newRevealStores(t)
	want := seedRevealApp(t, appStore, secretStore, revealTestAppID, true, revealTestMasterKey)

	got, err := Reveal(context.Background(), appStore, secretStore, revealTestMasterKey, revealTestAppID)
	if err != nil {
		t.Fatalf("reveal: %v", err)
	}
	if got != want {
		t.Fatalf("sign key = %q, want %q", got, want)
	}
}

func TestRevealPrefersActiveRecord(t *testing.T) {
	appStore, secretStore := newRevealStores(t)
	want := seedRevealApp(t, appStore, secretStore, revealTestAppID, true, revealTestMasterKey)

	// 模拟重置后的旧密钥：status=2 的行即使 id 更大，也不能被当成当前密钥。
	oldKey, err := security.NewSignKey()
	if err != nil {
		t.Fatalf("new old sign key: %v", err)
	}
	oldCipher, err := security.EncryptSignKey(revealTestMasterKey, oldKey)
	if err != nil {
		t.Fatalf("encrypt old sign key: %v", err)
	}
	if err := secretStore.Create(t.Context(), &gormmodel.OpenAppSecret{
		AppId:         revealTestAppID,
		SecretHash:    security.HashSecret("sec_old"),
		SignKeyCipher: oldCipher,
		Status:        2,
	}); err != nil {
		t.Fatalf("create old secret: %v", err)
	}

	got, err := Reveal(context.Background(), appStore, secretStore, revealTestMasterKey, revealTestAppID)
	if err != nil {
		t.Fatalf("reveal: %v", err)
	}
	if got != want {
		t.Fatalf("sign key = %q, want active %q", got, want)
	}
}

func TestRevealErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("empty app id", func(t *testing.T) {
		appStore, secretStore := newRevealStores(t)
		_, err := Reveal(ctx, appStore, secretStore, revealTestMasterKey, "  ")
		if !errors.Is(err, ErrAppIDRequired) {
			t.Fatalf("error = %v, want ErrAppIDRequired", err)
		}
	})

	t.Run("app not found", func(t *testing.T) {
		appStore, secretStore := newRevealStores(t)
		_, err := Reveal(ctx, appStore, secretStore, revealTestMasterKey, revealTestAppID)
		if !errors.Is(err, ErrAppNotFound) {
			t.Fatalf("error = %v, want ErrAppNotFound", err)
		}
	})

	t.Run("sign not enabled", func(t *testing.T) {
		appStore, secretStore := newRevealStores(t)
		seedRevealApp(t, appStore, secretStore, revealTestAppID, false, revealTestMasterKey)
		_, err := Reveal(ctx, appStore, secretStore, revealTestMasterKey, revealTestAppID)
		if !errors.Is(err, ErrNotEnabled) {
			t.Fatalf("error = %v, want ErrNotEnabled", err)
		}
	})

	t.Run("encryption key missing", func(t *testing.T) {
		appStore, secretStore := newRevealStores(t)
		seedRevealApp(t, appStore, secretStore, revealTestAppID, true, revealTestMasterKey)
		_, err := Reveal(ctx, appStore, secretStore, "short", revealTestAppID)
		if !errors.Is(err, ErrEncryptionKeyMissing) {
			t.Fatalf("error = %v, want ErrEncryptionKeyMissing", err)
		}
	})

	t.Run("active key missing", func(t *testing.T) {
		appStore, secretStore := newRevealStores(t)
		if err := appStore.Create(t.Context(), &gormmodel.OpenApp{
			AppId:        revealTestAppID,
			Name:         "无密钥应用",
			Status:       1,
			SignRequired: true,
		}); err != nil {
			t.Fatalf("create app: %v", err)
		}
		_, err := Reveal(ctx, appStore, secretStore, revealTestMasterKey, revealTestAppID)
		if !errors.Is(err, ErrKeyMissing) {
			t.Fatalf("error = %v, want ErrKeyMissing", err)
		}
	})

	t.Run("wrong master key", func(t *testing.T) {
		appStore, secretStore := newRevealStores(t)
		seedRevealApp(t, appStore, secretStore, revealTestAppID, true, revealTestMasterKey)
		_, err := Reveal(ctx, appStore, secretStore, "another-master-key-32-bytes-long!!", revealTestAppID)
		if !errors.Is(err, ErrDecrypt) {
			t.Fatalf("error = %v, want ErrDecrypt", err)
		}
	})
}
