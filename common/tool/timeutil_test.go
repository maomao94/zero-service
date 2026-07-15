package tool

import (
	"testing"
	"time"
)

func TestNowStartOfSecondClearsSubSecond(t *testing.T) {
	now := NowStartOfSecond().StdTime()
	if now.Nanosecond() != 0 {
		t.Fatalf("nanosecond = %d, want 0", now.Nanosecond())
	}
}

func TestCarbonFromTimeStartOfSecondClearsSubSecond(t *testing.T) {
	in := time.Date(2026, 7, 15, 10, 20, 30, 789123456, time.Local)
	out := CarbonFromTimeStartOfSecond(in)

	if got := out.StdTime().Nanosecond(); got != 0 {
		t.Fatalf("nanosecond = %d, want 0", got)
	}
	if got := out.ToShortDateTimeString(); got != "20260715102030" {
		t.Fatalf("compact datetime = %q, want 20260715102030", got)
	}
}
