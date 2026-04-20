package execdelay

import (
	"context"
	"fmt"

	"zero-service/app/trigger/internal/planscope"

	"github.com/dromara/carbon/v2"
)

type Config interface {
	GetNextTriggerTime() string
	GetDelayReason() string
}

type Warning string

const (
	WarnMissingDelayed Warning = "missing_config_delayed"
	WarnMissingOngoing Warning = "missing_config_ongoing"
	WarnInvalidNext    Warning = "invalid_next_trigger_time"
	WarnNextInPast     Warning = "next_trigger_in_past"
)

type Mode uint8

const (
	ModeDelayed Mode = 1
	ModeOngoing Mode = 2
)

type Result struct {
	NextTrigger string
	ReasonStem  string
	Warnings    []Warning
	InvalidRaw  string
	PastNext    string
	PastNow     string
}

const defaultFallbackMinutes = 5

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

func LogWarnings(ctx context.Context, scope planscope.Scope, r Result) {
	for _, w := range r.Warnings {
		switch w {
		case WarnMissingDelayed:
			scope.Logger(ctx).Error("延期重试（delayed）：下游未带 delay_config，将使用默认延后间隔")
		case WarnMissingOngoing:
			scope.Logger(ctx).Debug("进行中（ongoing）：无 delay_config，使用默认下次触发时间")
		case WarnInvalidNext:
			scope.Logger(ctx).Errorf("延期/进行中：下游给出的 next_trigger_time 无法解析 raw=%q", r.InvalidRaw)
		case WarnNextInPast:
			scope.Logger(ctx).Errorf("延期/进行中：下游给出的 next_trigger_time 早于当前时间 next=%s now=%s", r.PastNext, r.PastNow)
		}
	}
}

func FinalReason(reasonStem, nextTrigger string) string {
	return fmt.Sprintf("%s, 下次触发时间: %s", reasonStem, nextTrigger)
}
