package iec104

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/zeromicro/go-zero/core/logx"
)

func TestLogProviderFormatsVariadicArguments(t *testing.T) {
	var buf bytes.Buffer
	logx.SetWriter(logx.NewWriter(&buf))
	defer logx.Reset()

	provider := NewLogProvider(context.Background())
	provider.Error("iec104 client %s:%d failed", "127.0.0.1", 2404)

	got := buf.String()
	if !strings.Contains(got, "iec104 client 127.0.0.1:2404 failed") {
		t.Fatalf("expected formatted log message, got %q", got)
	}
}
