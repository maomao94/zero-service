package gormx

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/trace"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWithContextOnlyPropagatesContext(t *testing.T) {
	dialector := gorm.Dialector(sqlite.Open("file:" + t.Name() + "?mode=memory&cache=shared"))
	db, err := OpenWithDialector(&dialector, WithoutOpenTelemetry())
	if err != nil {
		t.Fatalf("open db error = %v", err)
	}

	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: [16]byte{1},
		SpanID:  [8]byte{1},
	}))
	gormDB := db.WithContext(ctx).DB
	if gormDB.Statement.Context != ctx {
		t.Fatalf("context should be propagated to gorm statement")
	}
	if _, ok := gormDB.Get("trace_id"); ok {
		t.Fatalf("trace_id should not be stored in gorm session")
	}
}
