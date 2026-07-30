package execdelay

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"zero-service/app/trigger/internal/planscope"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

func TestResolve_nilConfig_delayed(t *testing.T) {
	now := carbon.Parse("2026-04-20 12:00:00")
	r := Resolve(nil, "msg", "base", now, ModeDelayed)
	if len(r.Warnings) != 1 || r.Warnings[0] != WarnMissingDelayed {
		t.Fatalf("warnings=%v", r.Warnings)
	}
	if r.ReasonStem != "base" {
		t.Fatalf("reasonStem=%q", r.ReasonStem)
	}
	want := now.AddMinutes(5).ToDateTimeString()
	if r.NextTrigger != want {
		t.Fatalf("next=%q want %q", r.NextTrigger, want)
	}
}

func TestResolve_validConfig(t *testing.T) {
	now := carbon.Parse("2026-04-20 12:00:00")
	cfg := testCfg{next: "2026-04-21 08:00:00", reason: "r1"}
	r := Resolve(cfg, "m", "base", now, ModeDelayed)
	if len(r.Warnings) != 0 {
		t.Fatalf("warnings=%v", r.Warnings)
	}
	if r.NextTrigger != "2026-04-21 08:00:00" {
		t.Fatalf("next=%q", r.NextTrigger)
	}
	if r.ReasonStem != "r1, m" {
		t.Fatalf("stem=%q", r.ReasonStem)
	}
}

func TestLogWarningsPrefixesCronScopeOnly(t *testing.T) {
	tests := []struct {
		name       string
		scope      planscope.Scope
		wantPrefix bool
	}{
		{name: "cron", scope: planscope.TriggerScope(nil, nil), wantPrefix: true},
		{name: "callback", scope: planscope.CallbackScope(nil, nil, nil), wantPrefix: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logx.SetWriter(logx.NewWriter(&buf))
			defer logx.Reset()

			LogWarnings(context.Background(), tt.scope, Result{Warnings: []Warning{WarnMissingDelayed}})
			got := buf.String()
			if strings.Contains(got, "[cron-plan]") != tt.wantPrefix {
				t.Fatalf("log=%q, wantPrefix=%v", got, tt.wantPrefix)
			}
			if !strings.Contains(got, "延期重试") {
				t.Fatalf("log=%q, want delayed warning", got)
			}
		})
	}
}

func TestFinalReasonDistinguishesMode(t *testing.T) {
	delayed := FinalReason("base", "2026-04-21 08:00:00", ModeDelayed)
	if delayed != "base, 下次重试时间: 2026-04-21 08:00:00" {
		t.Fatalf("delayed=%q", delayed)
	}
	ongoing := FinalReason("base", "2026-04-21 08:00:00", ModeOngoing)
	if ongoing != "base, 下次进度复查时间: 2026-04-21 08:00:00" {
		t.Fatalf("ongoing=%q", ongoing)
	}
}

type testCfg struct {
	next, reason string
}

func (c testCfg) GetNextTriggerTime() string { return c.next }
func (c testCfg) GetDelayReason() string     { return c.reason }
