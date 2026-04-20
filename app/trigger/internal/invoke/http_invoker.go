package invoke

import (
	"context"
	"net/http"
	"time"
	"zero-service/app/trigger/internal/svc"
	"zero-service/common/netx"

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

	headers := make(http.Header, len(task.Headers))
	for k, v := range task.Headers {
		headers.Set(k, v)
	}

	clientOpts := []netx.ClientOption{netx.WithHttpcService(svcCtx.Httpc)}
	if task.Timeout > 0 {
		clientOpts = append(clientOpts, netx.WithTimeout(time.Duration(task.Timeout)*time.Millisecond))
	}

	resp, err := netx.SendRequest(ctx, netx.NewRequest(task.URL, method,
		netx.WithHeaders(headers),
		netx.WithBody(task.Body),
	), clientOpts...)
	if err != nil {
		logx.WithContext(ctx).Errorf("invoke http failed: id=%s url=%s err=%v", task.ID, task.URL, err)
		result.Error = err.Error()
		result.StatusCode = http.StatusBadRequest
		result.CostMs = time.Since(start).Milliseconds()
		return result
	}

	result.StatusCode = int32(resp.StatusCode)
	result.Data = resp.Data
	result.CostMs = resp.CostMs
	result.CostFormatted = resp.CostFormatted
	result.Success = resp.Success
	if resp.Error != "" {
		result.Error = resp.Error
	}

	return result
}
