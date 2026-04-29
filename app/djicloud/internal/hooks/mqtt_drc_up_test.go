package hooks

import (
	"context"
	"testing"
	"time"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/collection"
)

func TestNewDrcUpHandlerDoesNotRefreshOnline(t *testing.T) {
	onlineCache, err := collection.NewCache(time.Minute)
	if err != nil {
		t.Fatalf("NewCache online error = %v", err)
	}

	msg := &djisdk.DrcUpMessage{
		Timestamp: 123456,
		Method:    djisdk.MethodDrcHeartBeat,
		Seq:       7,
	}
	parsed := &djisdk.DrcHeartBeatUpData{Timestamp: 123450}

	if err := NewDrcUpHandler(onlineCache)(context.Background(), "gateway-1", msg, parsed); err != nil {
		t.Fatalf("NewDrcUpHandler() error = %v", err)
	}

	if IsOnline(onlineCache, "gateway-1") {
		t.Fatal("expected drc/up not to refresh online cache")
	}
}
