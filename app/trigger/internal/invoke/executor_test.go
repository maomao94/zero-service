package invoke

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"zero-service/app/trigger/internal/svc"

	"github.com/zeromicro/go-zero/rest/httpc"
)

func newTestSvcCtx() *svc.ServiceContext {
	return &svc.ServiceContext{
		Httpc: httpc.NewService("invoke-test"),
	}
}

func TestRun_AllSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	sc := newTestSvcCtx()
	tasks := make([]*Task, 3)
	for i := 0; i < 3; i++ {
		tasks[i] = &Task{
			ID:         fmt.Sprintf("task-%d", i),
			Protocol:   "http",
			HTTPMethod: "POST",
			URL:        ts.URL,
			Body:       []byte(`{"data":"test"}`),
		}
	}

	results := Run(context.Background(), sc, tasks, 0, false)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for i, r := range results {
		if !r.Success {
			t.Errorf("result[%d]: expected Success=true, got false, error=%s", i, r.Error)
		}
		if r.StatusCode != 200 {
			t.Errorf("result[%d]: expected StatusCode=200, got %d", i, r.StatusCode)
		}
		if len(r.Data) == 0 {
			t.Errorf("result[%d]: expected Data to be non-empty", i)
		}
	}
}

func TestRun_PartialFailure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/ok") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok":true}`))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"ok":false}`))
		}
	}))
	defer ts.Close()

	sc := newTestSvcCtx()
	tasks := []*Task{
		{ID: "ok-1", Protocol: "http", HTTPMethod: "POST", URL: ts.URL + "/ok/1"},
		{ID: "ok-2", Protocol: "http", HTTPMethod: "POST", URL: ts.URL + "/ok/2"},
		{ID: "fail-1", Protocol: "http", HTTPMethod: "POST", URL: ts.URL + "/fail"},
	}

	results := Run(context.Background(), sc, tasks, 0, false)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	successCount := 0
	failCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		} else {
			failCount++
		}
	}

	if successCount != 2 {
		t.Errorf("expected 2 successes, got %d", successCount)
	}
	if failCount != 1 {
		t.Errorf("expected 1 failure, got %d", failCount)
	}
}

func TestRun_TimeoutCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	sc := newTestSvcCtx()
	tasks := []*Task{
		{
			ID:         "timeout-1",
			Protocol:   "http",
			HTTPMethod: "POST",
			URL:        ts.URL,
			Timeout:    100,
		},
	}

	results := Run(context.Background(), sc, tasks, 0, false)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Success {
		t.Errorf("expected Success=false, got true")
	}
	if !strings.Contains(r.Error, "deadline") && !strings.Contains(r.Error, "timeout") {
		t.Errorf("expected error to contain 'deadline' or 'timeout', got %q", r.Error)
	}
}

func TestRun_ConcurrencyControl(t *testing.T) {
	var current int64
	var maxConcurrent int64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := atomic.AddInt64(&current, 1)
		for {
			old := atomic.LoadInt64(&maxConcurrent)
			if c <= old || atomic.CompareAndSwapInt64(&maxConcurrent, old, c) {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
		atomic.AddInt64(&current, -1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	sc := newTestSvcCtx()
	tasks := make([]*Task, 5)
	for i := 0; i < 5; i++ {
		tasks[i] = &Task{
			ID:         fmt.Sprintf("conc-%d", i),
			Protocol:   "http",
			HTTPMethod: "POST",
			URL:        ts.URL,
		}
	}

	results := Run(context.Background(), sc, tasks, 2, false)

	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}
	for i, r := range results {
		if !r.Success {
			t.Errorf("result[%d]: expected Success=true, got false, error=%s", i, r.Error)
		}
	}

	mc := atomic.LoadInt64(&maxConcurrent)
	if mc > 2 {
		t.Errorf("expected max concurrent <= 2, got %d", mc)
	}
}
