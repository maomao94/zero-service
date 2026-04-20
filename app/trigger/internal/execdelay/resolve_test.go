package execdelay

import (
	"testing"

	"github.com/dromara/carbon/v2"
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

type testCfg struct {
	next, reason string
}

func (c testCfg) GetNextTriggerTime() string { return c.next }
func (c testCfg) GetDelayReason() string     { return c.reason }
