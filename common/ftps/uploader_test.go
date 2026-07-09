package ftps

import (
	"testing"
	"time"
)

func TestNewUploaderDefaults(t *testing.T) {
	uploader := NewUploader(Config{})
	cfg := uploader.Config()
	if cfg.Timeout != 30*time.Second {
		t.Fatalf("Timeout = %s, want 30s", cfg.Timeout)
	}
	if cfg.TLSMode != TLSModeImplicit {
		t.Fatalf("TLSMode = %q, want %q", cfg.TLSMode, TLSModeImplicit)
	}
}

func TestNewUploaderPreservesConfig(t *testing.T) {
	in := Config{
		Address:            "ftps.example.invalid:990",
		Username:           "user",
		Password:           "pass",
		RemoteDir:          "/models",
		TLSMode:            TLSModeExplicit,
		InsecureSkipVerify: true,
		Timeout:            15 * time.Second,
		DisableEPSV:        true,
		UseTemporaryFile:   true,
	}
	out := NewUploader(in).Config()
	if out != in {
		t.Fatalf("Config() = %+v, want %+v", out, in)
	}
}
