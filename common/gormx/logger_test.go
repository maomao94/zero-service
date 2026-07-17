package gormx

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestGormLoggerTraceSkipsSQLWhenErrorLevelHasNoError(t *testing.T) {
	l := NewGormLogger(LoggerConfig{LogLevel: logger.Error, SlowThreshold: time.Millisecond})
	called := false

	l.Trace(context.Background(), time.Now(), func() (string, int64) {
		called = true
		return "select 1", 1
	}, nil)

	if called {
		t.Fatalf("fc should not be called when error level has no error")
	}
}

func TestGormLoggerTraceSkipsSQLWhenWarnLevelIsNotSlow(t *testing.T) {
	l := NewGormLogger(LoggerConfig{LogLevel: logger.Warn, SlowThreshold: time.Hour})
	called := false

	l.Trace(context.Background(), time.Now(), func() (string, int64) {
		called = true
		return "select 1", 1
	}, nil)

	if called {
		t.Fatalf("fc should not be called when warn level query is not slow")
	}
}

func TestGormLoggerTraceSkipsSQLWhenSilent(t *testing.T) {
	l := NewGormLogger(LoggerConfig{LogLevel: logger.Silent, SlowThreshold: time.Millisecond})
	called := false

	l.Trace(context.Background(), time.Now().Add(-time.Hour), func() (string, int64) {
		called = true
		return "select 1", 1
	}, errors.New("boom"))

	if called {
		t.Fatalf("fc should not be called when logger is silent")
	}
}

func TestGormLoggerTraceSkipsSuccessfulSQLWhenContextDisablesTrace(t *testing.T) {
	l := NewGormLogger(LoggerConfig{LogLevel: logger.Info, SlowThreshold: time.Hour})
	called := false

	l.Trace(WithoutSQLTrace(context.Background()), time.Now(), func() (string, int64) {
		called = true
		return "select 1", 1
	}, nil)

	if called {
		t.Fatalf("fc should not be called for successful sql when context disables sql trace")
	}
}

func TestGormLoggerTraceLogsErrorWhenContextDisablesTrace(t *testing.T) {
	l := NewGormLogger(LoggerConfig{LogLevel: logger.Error, SlowThreshold: time.Hour})
	called := false

	l.Trace(WithoutSQLTrace(context.Background()), time.Now(), func() (string, int64) {
		called = true
		return "select 1", 1
	}, errors.New("boom"))

	if !called {
		t.Fatalf("fc should be called for sql error even when context disables sql trace")
	}
}

func TestGormLoggerTraceLogsSlowSQLWhenContextDisablesTrace(t *testing.T) {
	l := NewGormLogger(LoggerConfig{LogLevel: logger.Error, SlowThreshold: time.Millisecond})
	called := false

	l.Trace(WithoutSQLTrace(context.Background()), time.Now().Add(-time.Hour), func() (string, int64) {
		called = true
		return "select 1", 1
	}, nil)

	if !called {
		t.Fatalf("fc should be called for slow sql even when context disables sql trace")
	}
}

func TestGormLoggerTraceLogsInfoAndSlowWhenInfoLevelSQLIsSlow(t *testing.T) {
	var buf bytes.Buffer
	logx.SetWriter(logx.NewWriter(&buf))
	defer logx.Reset()

	l := NewGormLogger(LoggerConfig{LogLevel: logger.Info, SlowThreshold: time.Millisecond})
	l.Trace(context.Background(), time.Now().Add(-time.Hour), func() (string, int64) {
		return "select 1", 1
	}, nil)

	output := buf.String()
	if strings.Count(output, "select 1") != 2 {
		t.Fatalf("expected info and slow logs, got: %s", output)
	}
	if !strings.Contains(output, "[SLOW]") {
		t.Fatalf("expected slow log, got: %s", output)
	}
}

func TestGormLoggerTraceCallsSQLWhenErrorLevelHasError(t *testing.T) {
	l := NewGormLogger(LoggerConfig{LogLevel: logger.Error, SlowThreshold: time.Hour})
	called := false

	l.Trace(context.Background(), time.Now(), func() (string, int64) {
		called = true
		return "select 1", 1
	}, errors.New("boom"))

	if !called {
		t.Fatalf("fc should be called when error level has error")
	}
}

func TestGormLoggerTraceSkipsRecordNotFoundAtErrorLevel(t *testing.T) {
	l := NewGormLogger(LoggerConfig{LogLevel: logger.Error, SlowThreshold: time.Hour, IgnoreRecordNotFoundError: true})
	called := false

	l.Trace(context.Background(), time.Now(), func() (string, int64) {
		called = true
		return "select 1", 1
	}, gorm.ErrRecordNotFound)

	if called {
		t.Fatalf("fc should not be called for record not found at error level when ignored")
	}
}

func TestGormLoggerTraceCallsRecordNotFoundAtErrorLevelByDefault(t *testing.T) {
	l := NewGormLogger(LoggerConfig{LogLevel: logger.Error, SlowThreshold: time.Hour})
	called := false

	l.Trace(context.Background(), time.Now(), func() (string, int64) {
		called = true
		return "select 1", 1
	}, gorm.ErrRecordNotFound)

	if !called {
		t.Fatalf("fc should be called for record not found at error level by default")
	}
}

func TestGormLoggerTraceCallsRecordNotFoundAtInfoLevel(t *testing.T) {
	l := NewGormLogger(LoggerConfig{LogLevel: logger.Info, SlowThreshold: time.Hour})
	called := false

	l.Trace(context.Background(), time.Now(), func() (string, int64) {
		called = true
		return "select 1", 1
	}, gorm.ErrRecordNotFound)

	if !called {
		t.Fatalf("fc should be called for record not found at info level")
	}
}

func TestGormLoggerTraceCallsSQLWhenWarnLevelIsSlow(t *testing.T) {
	l := NewGormLogger(LoggerConfig{LogLevel: logger.Warn, SlowThreshold: time.Millisecond})
	called := false

	l.Trace(context.Background(), time.Now().Add(-time.Hour), func() (string, int64) {
		called = true
		return "select 1", 1
	}, nil)

	if !called {
		t.Fatalf("fc should be called when warn level query is slow")
	}
}

func TestGormLoggerTraceCallsSQLWhenInfoLevel(t *testing.T) {
	l := NewGormLogger(LoggerConfig{LogLevel: logger.Info, SlowThreshold: time.Hour})
	called := false

	l.Trace(context.Background(), time.Now(), func() (string, int64) {
		called = true
		return "select 1", 1
	}, nil)

	if !called {
		t.Fatalf("fc should be called when info level logs normal query")
	}
}

func TestGormLoggerLogModeDoesNotMutateOriginal(t *testing.T) {
	original := NewGormLogger(LoggerConfig{LogLevel: logger.Error, SlowThreshold: time.Second})
	changed := original.LogMode(logger.Info)

	originalCalled := false
	original.Trace(context.Background(), time.Now(), func() (string, int64) {
		originalCalled = true
		return "select 1", 1
	}, nil)
	if originalCalled {
		t.Fatalf("original logger should keep error level")
	}

	changedCalled := false
	changed.Trace(context.Background(), time.Now(), func() (string, int64) {
		changedCalled = true
		return "select 1", 1
	}, nil)
	if !changedCalled {
		t.Fatalf("changed logger should use info level")
	}
}

func TestGormLoggerParamsFilterShowsQueryParametersByDefault(t *testing.T) {
	l := NewGormLogger(LoggerConfig{})
	sql, params := l.(*gormLogger).ParamsFilter(context.Background(), "select * from users where phone = ?", "13800000000")

	if sql != "select * from users where phone = ?" {
		t.Fatalf("sql = %q", sql)
	}
	if len(params) != 1 || params[0] != "13800000000" {
		t.Fatalf("params = %#v", params)
	}
}

func TestGormLoggerParamsFilterUsesGormParameterizedQueries(t *testing.T) {
	l := NewGormLogger(LoggerConfig{ParameterizedQueries: true})
	sql, params := l.(*gormLogger).ParamsFilter(context.Background(), "select * from users where phone = ?", "13800000000")

	if sql != "select * from users where phone = ?" {
		t.Fatalf("sql = %q", sql)
	}
	if params != nil {
		t.Fatalf("params = %#v, want nil", params)
	}
}

func TestGormLoggerParamsFilterIgnoresNonFullSQLContext(t *testing.T) {
	l := NewGormLogger(LoggerConfig{ParameterizedQueries: true})
	ctx := context.WithValue(context.Background(), "other_key", true)
	sql, params := l.(*gormLogger).ParamsFilter(ctx, "select * from users where phone = ?", "13800000000")

	if sql != "select * from users where phone = ?" {
		t.Fatalf("sql = %q", sql)
	}
	if params != nil {
		t.Fatalf("params should be nil for non-FullSQL context")
	}
}

func TestGormLoggerParamsFilterDefaultGormLoggerRedactsQueryParameters(t *testing.T) {
	l := DefaultGormLogger()
	_, params := l.(*gormLogger).ParamsFilter(context.Background(), "select * from users where phone = ?", "13800000000")

	if len(params) != 0 {
		t.Fatalf("expected params to be redacted, got %#v", params)
	}
}

func TestWithFullSQLContextSetsContextValue(t *testing.T) {
	ctx := WithFullSQL(context.Background())
	v, ok := ctx.Value(fullSQLKey{}).(bool)
	if !ok || !v {
		t.Fatalf("fullSQLKey = %v/%v, want true/true", v, ok)
	}
}

func TestQuietGormLoggerSilencesAllOutput(t *testing.T) {
	l := QuietGormLogger()
	called := false

	l.Trace(context.Background(), time.Now().Add(-time.Hour), func() (string, int64) {
		called = true
		return "select 1", 1
	}, errors.New("boom"))

	if called {
		t.Fatalf("fc should not be called for quiet logger")
	}
}

func TestFormatRows(t *testing.T) {
	if got := formatRows(-1); got != "-" {
		t.Fatalf("formatRows(-1) = %q, want -", got)
	}
	if got := formatRows(10); got != "10" {
		t.Fatalf("formatRows(10) = %q, want 10", got)
	}
}
