package main

import (
	"testing"

	"zero-service/app/opengtw/internal/config"
	"zero-service/app/opengtw/internal/security"
	"zero-service/common/gormx"
	"zero-service/model/gormmodel"
)

func TestRevealKeyFromSQLite(t *testing.T) {
	const (
		masterKey = "0123456789abcdef0123456789abcdef"
		appID     = "app_0123456789abcdef0123456789abcdef"
	)

	cfg := config.Config{
		DB: gormx.Config{
			DataSource:   "file:" + t.TempDir() + "/keytool.db?parseTime=true&_loc=auto",
			MaxIdleConns: 1,
			MaxOpenConns: 1,
		},
		SignKeyEncryptionKey: masterKey,
	}

	db, err := gormx.OpenWithConf(cfg.DB)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.MustAutoMigrate(&gormmodel.OpenApp{}, &gormmodel.OpenAppSecret{})

	signKey, err := security.NewSignKey()
	if err != nil {
		t.Fatalf("new sign key: %v", err)
	}
	cipher, err := security.EncryptSignKey(masterKey, signKey)
	if err != nil {
		t.Fatalf("encrypt sign key: %v", err)
	}
	if err := gormmodel.NewOpenAppStore(db).Create(t.Context(), &gormmodel.OpenApp{
		AppId:        appID,
		Name:         "keytool 测试",
		Status:       1,
		SignRequired: true,
	}); err != nil {
		t.Fatalf("create app: %v", err)
	}
	if err := gormmodel.NewOpenAppSecretStore(db).Create(t.Context(), &gormmodel.OpenAppSecret{
		AppId:         appID,
		SecretHash:    security.HashSecret("sec_test"),
		SignKeyCipher: cipher,
		Status:        1,
	}); err != nil {
		t.Fatalf("create secret: %v", err)
	}
	closeDB(db)

	got, err := revealKey(cfg, appID)
	if err != nil {
		t.Fatalf("reveal key: %v", err)
	}
	if got != signKey {
		t.Fatalf("sign key = %q, want %q", got, signKey)
	}
}
