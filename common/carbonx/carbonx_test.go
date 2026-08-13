package carbonx

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/dromara/carbon/v2"
)

func TestFromTime(t *testing.T) {
	t.Parallel()

	nepal := time.FixedZone("NPT", 5*60*60+45*60)
	value := time.Date(2026, time.August, 13, 9, 8, 7, 654321000, nepal)

	tests := []struct {
		name       string
		timezone   []string
		wantText   string
		wantOffset int
	}{
		{
			name:       "preserve input location",
			wantText:   "2026-08-13 09:08:07",
			wantOffset: 5*60*60 + 45*60,
		},
		{
			name:       "explicitly convert timezone",
			timezone:   []string{carbon.Shanghai},
			wantText:   "2026-08-13 11:23:07",
			wantOffset: 8 * 60 * 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromTime(value, tt.timezone...)
			if got.Error != nil {
				t.Fatalf("FromTime() error = %v", got.Error)
			}
			if text := got.ToDateTimeString(); text != tt.wantText {
				t.Fatalf("FromTime() text = %q, want %q", text, tt.wantText)
			}
			_, offset := got.StdTime().Zone()
			if offset != tt.wantOffset {
				t.Fatalf("FromTime() offset = %d, want %d", offset, tt.wantOffset)
			}
			if !got.StdTime().Equal(value) {
				t.Fatal("FromTime() changed the represented instant")
			}
		})
	}
}

func TestStartOfSecond(t *testing.T) {
	t.Parallel()

	value := time.Date(2026, time.August, 13, 9, 8, 7, 654321000, time.FixedZone("NPT", 20700))
	got := FromTimeStartOfSecond(value)
	if nanosecond := got.StdTime().Nanosecond(); nanosecond != 0 {
		t.Fatalf("FromTimeStartOfSecond() nanosecond = %d, want 0", nanosecond)
	}
	if location := got.StdTime().Location(); location != value.Location() {
		t.Fatalf("FromTimeStartOfSecond() location = %v, want %v", location, value.Location())
	}

	now := NowStartOfSecond().StdTime()
	if nanosecond := now.Nanosecond(); nanosecond != 0 {
		t.Fatalf("NowStartOfSecond() nanosecond = %d, want 0", nanosecond)
	}
}

func TestNowDateTimeFormatting(t *testing.T) {
	location, err := time.LoadLocation(carbon.Shanghai)
	if err != nil {
		t.Fatalf("load Carbon default timezone: %v", err)
	}

	tests := []struct {
		name      string
		now       func() string
		pattern   string
		precision time.Duration
	}{
		{
			name:      "seconds",
			now:       NowDateTime,
			pattern:   `^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`,
			precision: time.Second,
		},
		{
			name:      "milliseconds",
			now:       NowDateTimeMilli,
			pattern:   `^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d{1,3})?$`,
			precision: time.Millisecond,
		},
		{
			name:      "microseconds",
			now:       NowDateTimeMicro,
			pattern:   `^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d{1,6})?$`,
			precision: time.Microsecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatPattern := regexp.MustCompile(tt.pattern)
			before := time.Now()
			text := tt.now()
			after := time.Now()

			if !formatPattern.MatchString(text) {
				t.Fatalf("now() = %q, want Carbon %s format", text, tt.name)
			}

			parsed, parseErr := time.ParseInLocation("2006-01-02 15:04:05.999999", text, location)
			if parseErr != nil {
				t.Fatalf("parse now() output %q: %v", text, parseErr)
			}
			if parsed.Before(before.Truncate(tt.precision)) || parsed.After(after) {
				t.Fatalf("now() = %q (%v), want current instant between %v and %v in %s", text, parsed, before, after, carbon.Shanghai)
			}
		})
	}
}

func TestFormatDateTimePrecision(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     time.Time
		format    func(time.Time, ...string) string
		carbonFmt func(*carbon.Carbon) string
		want      string
	}{
		{
			name:      "seconds",
			value:     time.Date(2026, time.August, 13, 9, 8, 7, 123456789, time.UTC),
			format:    FormatDateTime,
			carbonFmt: func(value *carbon.Carbon) string { return value.ToDateTimeString() },
			want:      "2026-08-13 09:08:07",
		},
		{
			name:      "milliseconds trim trailing zeros",
			value:     time.Date(2026, time.August, 13, 9, 8, 7, 120340000, time.UTC),
			format:    FormatDateTimeMilli,
			carbonFmt: func(value *carbon.Carbon) string { return value.ToDateTimeMilliString() },
			want:      "2026-08-13 09:08:07.12",
		},
		{
			name:      "milliseconds truncate finer precision",
			value:     time.Date(2026, time.August, 13, 9, 8, 7, 123456789, time.UTC),
			format:    FormatDateTimeMilli,
			carbonFmt: func(value *carbon.Carbon) string { return value.ToDateTimeMilliString() },
			want:      "2026-08-13 09:08:07.123",
		},
		{
			name:      "microseconds trim trailing zeros",
			value:     time.Date(2026, time.August, 13, 9, 8, 7, 120340000, time.UTC),
			format:    FormatDateTimeMicro,
			carbonFmt: func(value *carbon.Carbon) string { return value.ToDateTimeMicroString() },
			want:      "2026-08-13 09:08:07.12034",
		},
		{
			name:      "microseconds truncate nanoseconds",
			value:     time.Date(2026, time.August, 13, 9, 8, 7, 123456789, time.UTC),
			format:    FormatDateTimeMicro,
			carbonFmt: func(value *carbon.Carbon) string { return value.ToDateTimeMicroString() },
			want:      "2026-08-13 09:08:07.123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.format(tt.value)
			if got != tt.want {
				t.Fatalf("format() = %q, want %q", got, tt.want)
			}
			legacy := tt.carbonFmt(carbon.CreateFromStdTime(tt.value))
			if got != legacy {
				t.Fatalf("format() = %q, direct Carbon output = %q", got, legacy)
			}
		})
	}
}

func TestFormatDateTimeTimezone(t *testing.T) {
	t.Parallel()

	value := time.Date(2026, time.August, 13, 1, 2, 3, 123456000, time.UTC)
	tests := []struct {
		name          string
		format        func(time.Time, ...string) string
		wantPreserved string
		wantConverted string
	}{
		{name: "seconds", format: FormatDateTime, wantPreserved: "2026-08-13 01:02:03", wantConverted: "2026-08-13 09:02:03"},
		{name: "milliseconds", format: FormatDateTimeMilli, wantPreserved: "2026-08-13 01:02:03.123", wantConverted: "2026-08-13 09:02:03.123"},
		{name: "microseconds", format: FormatDateTimeMicro, wantPreserved: "2026-08-13 01:02:03.123456", wantConverted: "2026-08-13 09:02:03.123456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.format(value); got != tt.wantPreserved {
				t.Fatalf("format() = %q, want input location output %q", got, tt.wantPreserved)
			}
			if got := tt.format(value, carbon.Shanghai); got != tt.wantConverted {
				t.Fatalf("format(Shanghai) = %q, want %q", got, tt.wantConverted)
			}
		})
	}
}

func TestTimezoneErrorsFollowCarbonSemantics(t *testing.T) {
	t.Parallel()

	value := time.Date(2026, time.August, 13, 1, 2, 3, 0, time.UTC)
	from := FromTime(value, "not/a-timezone")
	if from.Error == nil {
		t.Fatal("FromTime() error = nil, want invalid timezone error")
	}

	tests := []struct {
		name   string
		format func(time.Time, ...string) string
	}{
		{name: "seconds", format: FormatDateTime},
		{name: "milliseconds", format: FormatDateTimeMilli},
		{name: "microseconds", format: FormatDateTimeMicro},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.format(value, "not/a-timezone"); got != "" {
				t.Fatalf("format() = %q, want empty output for invalid timezone", got)
			}
		})
	}
}

func TestOptionalDateTimeFormatting(t *testing.T) {
	t.Parallel()

	zero := time.Time{}
	nonZero := time.Date(2026, time.August, 13, 9, 8, 7, 0, time.UTC)
	tests := []struct {
		name  string
		value func() string
		want  string
	}{
		{name: "zero time is formatted by the ordinary API", value: func() string { return FormatDateTime(zero, "UTC") }, want: "0001-01-01 00:00:00"},
		{name: "zero time is formatted by the ordinary millisecond API", value: func() string { return FormatDateTimeMilli(zero, "UTC") }, want: "0001-01-01 00:00:00"},
		{name: "zero time is formatted by the ordinary microsecond API", value: func() string { return FormatDateTimeMicro(zero, "UTC") }, want: "0001-01-01 00:00:00"},
		{name: "zero time is empty", value: func() string { return FormatDateTimeOrEmpty(zero) }, want: ""},
		{name: "non-zero time", value: func() string { return FormatDateTimeOrEmpty(nonZero) }, want: "2026-08-13 09:08:07"},
		{name: "invalid SQL time is empty", value: func() string { return FormatNullDateTime(sql.NullTime{}) }, want: ""},
		{name: "valid SQL time", value: func() string {
			return FormatNullDateTime(sql.NullTime{Time: nonZero, Valid: true})
		}, want: "2026-08-13 09:08:07"},
		{name: "valid SQL zero time is formatted", value: func() string {
			return FormatNullDateTime(sql.NullTime{Time: zero, Valid: true}, "UTC")
		}, want: "0001-01-01 00:00:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value(); got != tt.want {
				t.Fatalf("format() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatDateTimeMicroOrEmpty(t *testing.T) {
	t.Parallel()

	value := time.Date(2026, time.August, 13, 1, 2, 3, 123456000, time.UTC)
	tests := []struct {
		name     string
		value    time.Time
		timezone []string
		want     string
	}{
		{name: "zero time", value: time.Time{}, want: ""},
		{name: "preserve input timezone", value: value, want: "2026-08-13 01:02:03.123456"},
		{name: "explicit timezone", value: value, timezone: []string{carbon.Shanghai}, want: "2026-08-13 09:02:03.123456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatDateTimeMicroOrEmpty(tt.value, tt.timezone...); got != tt.want {
				t.Fatalf("FormatDateTimeMicroOrEmpty() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestToNullTime(t *testing.T) {
	t.Parallel()

	value := time.Date(2026, time.August, 13, 1, 2, 3, 123456789, time.FixedZone("NPT", 20700))
	tests := []struct {
		name  string
		value time.Time
		want  sql.NullTime
	}{
		{name: "zero time is invalid", value: time.Time{}, want: sql.NullTime{Time: time.Time{}, Valid: false}},
		{name: "non-zero time is valid", value: value, want: sql.NullTime{Time: value, Valid: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToNullTime(tt.value)
			if got != tt.want {
				t.Fatalf("ToNullTime() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestNowUnixUnits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		now       func() int64
		fromTime  func(time.Time) int64
		tolerance int64
	}{
		{name: "seconds", now: NowUnix, fromTime: func(value time.Time) int64 { return value.Unix() }, tolerance: 1},
		{name: "milliseconds", now: NowUnixMilli, fromTime: func(value time.Time) int64 { return value.UnixMilli() }, tolerance: 100},
		{name: "microseconds", now: NowUnixMicro, fromTime: func(value time.Time) int64 { return value.UnixMicro() }, tolerance: 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := tt.fromTime(time.Now())
			got := tt.now()
			after := tt.fromTime(time.Now())
			if got < before-tt.tolerance || got > after+tt.tolerance {
				t.Fatalf("now() = %d, want between %d and %d", got, before, after)
			}
		})
	}
}
