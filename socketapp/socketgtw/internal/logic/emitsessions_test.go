package logic

import (
	"context"
	"errors"
	"testing"

	"zero-service/common/socketiox"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

func TestEmitSessionsNoSessions(t *testing.T) {
	runner := threading.NewTaskRunner(1)
	logger := logx.WithContext(context.Background())
	if err := emitSessions(runner, logger, nil, "event", nil, "req"); err != nil {
		t.Fatalf("expect nil error, got %v", err)
	}
}

func TestEmitSessionsDropsWhenPoolFull(t *testing.T) {
	runner := threading.NewTaskRunner(1)
	release := make(chan struct{})
	runner.Schedule(func() { <-release })
	defer close(release)

	logger := logx.WithContext(context.Background())
	sessions := []*socketiox.Session{{}, {}}
	err := emitSessions(runner, logger, sessions, "event", nil, "req")

	var dropErr *emitDropError
	if !errors.As(err, &dropErr) {
		t.Fatalf("expect *emitDropError, got %T: %v", err, err)
	}
	if len(dropErr.Dropped) != 2 {
		t.Fatalf("expect 2 dropped sessions, got %v", dropErr.Dropped)
	}
}

func TestEmitSessionsEmitErrorNotReportedAsDrop(t *testing.T) {
	runner := threading.NewTaskRunner(4)
	logger := logx.WithContext(context.Background())
	// 零值 Session 没有底层 socket，EmitDown 必然失败；这一分支只记录日志，不返回调度失败。
	if err := emitSessions(runner, logger, []*socketiox.Session{{}}, "event", nil, "req"); err != nil {
		t.Fatalf("expect nil error, got %v", err)
	}
}
