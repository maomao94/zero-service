package invoke

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"time"
	"zero-service/app/trigger/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type HTTPInvoker struct{}

func (h *HTTPInvoker) Execute(ctx context.Context, svcCtx *svc.ServiceContext, task *Task) *Result {
	start := time.Now()
	result := &Result{ID: task.ID}

	method := task.HTTPMethod
	if method == "" {
		method = http.MethodPost
	}

	var bodyReader io.Reader
	if len(task.Body) > 0 {
		bodyReader = bytes.NewReader(task.Body)
	}

	req, err := http.NewRequestWithContext(ctx, method, task.URL, bodyReader)
	if err != nil {
		result.Error = err.Error()
		result.StatusCode = http.StatusBadRequest
		result.CostMs = time.Since(start).Milliseconds()
		return result
	}

	for k, v := range task.Headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Content-Type") == "" && len(task.Body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := svcCtx.Httpc.DoRequest(req)
	if err != nil {
		result.Error = err.Error()
		result.CostMs = time.Since(start).Milliseconds()
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			result.StatusCode = http.StatusRequestTimeout
		} else {
			result.StatusCode = http.StatusBadGateway
		}
		logx.WithContext(ctx).Errorf("invoke http failed: id=%s url=%s err=%v", task.ID, task.URL, err)
		return result
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = err.Error()
		result.StatusCode = int32(resp.StatusCode)
		result.CostMs = time.Since(start).Milliseconds()
		return result
	}

	result.StatusCode = int32(resp.StatusCode)
	result.Data = data
	result.CostMs = time.Since(start).Milliseconds()
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300

	return result
}
