package gormx

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestOpenWithDialectorEnablesOpenTelemetryByDefault(t *testing.T) {
	dialector := gorm.Dialector(sqlite.Open("file:" + t.Name() + "?mode=memory&cache=shared"))
	db, err := OpenWithDialector(&dialector)
	if err != nil {
		t.Fatalf("open db error = %v", err)
	}

	if _, ok := db.DB.Config.Plugins[openTelemetryPluginName]; !ok {
		t.Fatalf("open telemetry plugin should be enabled by default")
	}
}

func TestOpenWithDialectorCanDisableOpenTelemetry(t *testing.T) {
	dialector := gorm.Dialector(sqlite.Open("file:" + t.Name() + "?mode=memory&cache=shared"))
	db, err := OpenWithDialector(&dialector, WithoutOpenTelemetry())
	if err != nil {
		t.Fatalf("open db error = %v", err)
	}

	if _, ok := db.DB.Config.Plugins[openTelemetryPluginName]; ok {
		t.Fatalf("open telemetry plugin should be disabled")
	}
}

func TestOpenWithConfCanDisableOpenTelemetry(t *testing.T) {
	dialector := gorm.Dialector(sqlite.Open("file:" + t.Name() + "?mode=memory&cache=shared"))
	db, err := Open("", func(o *dbOptions) { o.dialector = &dialector }, WithOpenTelemetryConfig(TraceConfig{Disabled: true}))
	if err != nil {
		t.Fatalf("open db error = %v", err)
	}

	if _, ok := db.DB.Config.Plugins[openTelemetryPluginName]; ok {
		t.Fatalf("open telemetry plugin should be disabled by config")
	}
}

func TestOpenTelemetryConfigDefaultsToSafeTracing(t *testing.T) {
	cfg := defaultOpenTelemetryConfig()

	if cfg.Disabled {
		t.Fatalf("open telemetry should be enabled by default")
	}
	if cfg.WithMetrics {
		t.Fatalf("metrics should be disabled by default")
	}
	if cfg.WithQueryVariables {
		t.Fatalf("query variables should be disabled by default")
	}
}

func TestOpenTelemetryConfigCanBeDisabledByConfig(t *testing.T) {
	opts := &dbOptions{}
	WithOpenTelemetryConfig(TraceConfig{Disabled: true})(opts)

	if !opts.openTelemetry.Disabled {
		t.Fatalf("open telemetry should be disabled by config")
	}
}
