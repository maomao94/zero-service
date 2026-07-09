package config

import (
	"testing"
	"time"

	"zero-service/common/ftps"
)

func TestModelSyncFTPSConfigToFTPSConfig(t *testing.T) {
	in := ModelSyncFTPSConfig{
		Address:            "ftps.example.invalid:990",
		Username:           "user",
		Password:           "pass",
		RemoteDir:          "/models",
		TLSMode:            "explicit",
		InsecureSkipVerify: true,
		Timeout:            15 * time.Second,
		DisableEPSV:        true,
		UseTemporaryFile:   true,
	}
	out := in.ToFTPSConfig()
	if out.Address != in.Address || out.Username != in.Username || out.Password != in.Password {
		t.Fatalf("basic config mismatch: %+v", out)
	}
	if out.RemoteDir != in.RemoteDir || out.TLSMode != ftps.TLSModeExplicit || out.Timeout != in.Timeout {
		t.Fatalf("ftps config mismatch: %+v", out)
	}
	if !out.InsecureSkipVerify || !out.DisableEPSV || !out.UseTemporaryFile {
		t.Fatalf("boolean config mismatch: %+v", out)
	}
}
