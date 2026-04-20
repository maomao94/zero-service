// Package execdelay resolves next trigger time from delay_config for plan_exec_item delayed/ongoing paths.
package execdelay

import (
	"context"
	"fmt"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

// Config matches trigger/streamevent DelayConfigPb (getter-only to avoid import cycles).
type Config interface {
	GetNextTriggerTime() string
	GetDelayReason() string
}

// Warning describes a non-fatal resolve issue; LogWarnings maps these to leveled logs.
type Warning string

const (
	WarnMissingDelayed Warning = "missing_config_delayed"
	WarnMissingOngoing Warning = "missing_config_ongoing"
	WarnInvalidNext    Warning = "invalid_next_trigger_time"
	WarnNextInPast     Warning = "next_trigger_in_past"
)

// Mode distinguishes delayed vs ongoing when delay_config is missing (error vs debug).
type Mode uint8

const (
	ModeDelayed Mode = 1
	ModeOngoing Mode = 2
)

// Result is the pure computation output; Warnings may contain multiple entries.
type Result struct {
	NextTrigger string // yyyy-MM-dd HH:mm:ss
	ReasonStem  string
	Warnings    []Warning
	InvalidRaw  string // WarnInvalidNext
	PastNext    string // WarnNextInPast
	PastNow     string
}

const defaultFallbackMinutes = 5

// Resolve computes next trigger time from delay_config and current time (same rules as legacy cron/callback).
func Resolve(cfg Config, streamMessage, reasonStem string, now *carbon.Carbon, mode Mode) Result {
	if now == nil {
		now = carbon.Now()
	}
	out := Result{
		NextTrigger: now.AddMinutes(defaultFallbackMinutes).ToDateTimeString(),
		ReasonStem:  reasonStem,
	}
	if cfg == nil {
		if mode == ModeDelayed {
			out.Warnings = append(out.Warnings, WarnMissingDelayed)
		} else {
			out.Warnings = append(out.Warnings, WarnMissingOngoing)
		}
		return out
	}
	if dr := cfg.GetDelayReason(); dr != "" {
		out.ReasonStem = fmt.Sprintf("%s, %s", dr, streamMessage)
	}
	raw := cfg.GetNextTriggerTime()
	dt := carbon.ParseByLayout(raw, carbon.DateTimeLayout)
	if dt.Error != nil || dt.IsInvalid() {
		out.Warnings = append(out.Warnings, WarnInvalidNext)
		out.InvalidRaw = raw
		return out
	}
	if dt.Lt(now) {
		out.Warnings = append(out.Warnings, WarnNextInPast)
		out.PastNext = dt.ToDateTimeString()
		out.PastNow = now.ToDateTimeString()
		return out
	}
	out.NextTrigger = dt.ToDateTimeString()
	return out
}

// LogWarnings writes leveled logs for Result.Warnings; scope must be a planscope line (message prefix).
func LogWarnings(ctx context.Context, scope string, r Result) {
	logger := logx.WithContext(ctx)
	for _, w := range r.Warnings {
		switch w {
		case WarnMissingDelayed:
			logger.Errorf("%s delayed 结果但缺少 delay_config，将使用默认延后", scope)
		case WarnMissingOngoing:
			logger.Debugf("%s ongoing 无 delay_config，使用默认下次触发时间", scope)
		case WarnInvalidNext:
			logger.Errorf("%s delayed/ongoing 的 next_trigger_time 非法 raw=%q", scope, r.InvalidRaw)
		case WarnNextInPast:
			logger.Errorf("%s delayed/ongoing 的 next_trigger_time 早于当前 next=%s now=%s", scope, r.PastNext, r.PastNow)
		}
	}
}

// FinalReason appends the human-readable next trigger time to the reason stem.
func FinalReason(reasonStem, nextTrigger string) string {
	return fmt.Sprintf("%s, 下次触发时间: %s", reasonStem, nextTrigger)
}
