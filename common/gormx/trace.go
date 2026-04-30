package gormx

import (
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

const openTelemetryPluginName = "otelgorm"

type TraceConfig struct {
	Disabled           bool   `json:",optional,default=false"`
	DBName             string `json:",optional"`
	WithMetrics        bool   `json:",optional,default=false"`
	WithQueryVariables bool   `json:",optional,default=false"`
}

func defaultOpenTelemetryConfig() TraceConfig {
	return TraceConfig{}
}

func WithOpenTelemetry() Option {
	return func(o *dbOptions) {
		o.openTelemetry.Disabled = false
	}
}

func WithoutOpenTelemetry() Option {
	return func(o *dbOptions) {
		o.openTelemetry.Disabled = true
	}
}

func WithOpenTelemetryConfig(cfg TraceConfig) Option {
	return func(o *dbOptions) {
		o.openTelemetry = cfg
	}
}

func registerOpenTelemetry(db *gorm.DB, cfg TraceConfig) error {
	if cfg.Disabled {
		return nil
	}

	opts := make([]tracing.Option, 0, 3)
	if cfg.DBName != "" {
		opts = append(opts, tracing.WithDBName(cfg.DBName))
	}
	if !cfg.WithMetrics {
		opts = append(opts, tracing.WithoutMetrics())
	}
	if !cfg.WithQueryVariables {
		opts = append(opts, tracing.WithoutQueryVariables())
	}

	return db.Use(tracing.NewPlugin(opts...))
}
