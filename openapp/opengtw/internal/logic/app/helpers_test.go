package app

import (
	"testing"
	"time"
)

func TestFormatTimeUsesProjectLayout(t *testing.T) {
	value := time.Date(2026, 10, 23, 17, 49, 59, 0, time.Local)
	if got := formatTime(value); got != "2026-10-23 17:49:59" {
		t.Fatalf("formatTime() = %q, want yyyy-MM-dd HH:mm:ss", got)
	}
}

func TestFormatTimeZeroValueIsEmpty(t *testing.T) {
	if got := formatTime(time.Time{}); got != "" {
		t.Fatalf("formatTime(zero) = %q, want empty", got)
	}
}
