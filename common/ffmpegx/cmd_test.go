package ffmpegx

import (
	"context"
	"slices"
	"testing"
)

func TestBuildRelayCmdEnablesStdoutProgress(t *testing.T) {
	cmd := BuildRelayCmd(context.Background(), "rtmp://source/live/input", "rtmp://target/live/output")
	want := []string{"-progress", "pipe:1", "-stats_period", "5"}
	if !slices.Equal(cmd.Args[1:5], want) {
		t.Fatalf("relay command prefix = %q, want %q", cmd.Args[1:5], want)
	}
}
