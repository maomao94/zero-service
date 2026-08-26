package ffmpegx

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestWatchOutputCallsBackForEveryLine(t *testing.T) {
	var lines []string
	if err := WatchOutput(io.NopCloser(strings.NewReader("frame=1\nprogress=continue\nordinary\n")), func(line string) {
		lines = append(lines, line)
	}); err != nil {
		t.Fatalf("WatchOutput: %v", err)
	}
	if got := strings.Join(lines, ","); got != "frame=1,progress=continue,ordinary" {
		t.Fatalf("lines = %q", got)
	}
}

func TestWatchOutputReturnsReadError(t *testing.T) {
	want := errors.New("read failed")
	if err := WatchOutput(errorReadCloser{err: want}, func(string) {}); !errors.Is(err, want) {
		t.Fatalf("WatchOutput error = %v, want %v", err, want)
	}
}

type errorReadCloser struct{ err error }

func (r errorReadCloser) Read([]byte) (int, error) { return 0, r.err }
func (errorReadCloser) Close() error               { return nil }
