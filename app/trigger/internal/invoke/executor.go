package invoke

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
	"zero-service/app/trigger/internal/svc"
	"zero-service/common/antsx"
	"zero-service/common/netx"

	"github.com/zeromicro/go-zero/core/logx"
)

func Run(ctx context.Context, svcCtx *svc.ServiceContext, tasks []*Task, maxConcurrency int32, debug bool) []*Result {
	if len(tasks) == 0 {
		return nil
	}

	antsxTasks := make([]antsx.Task[*Result], 0, len(tasks))
	for _, t := range tasks {
		task := t
		at := antsx.Task[*Result]{
			Name: task.ID,
			Fn: func(ctx context.Context) (*Result, error) {
				var invoker Invoker
				switch task.Protocol {
				case "http":
					invoker = &HTTPInvoker{}
				case "grpc":
					invoker = &GRPCInvoker{}
				default:
					return &Result{
						ID:         task.ID,
						Error:      fmt.Sprintf("unsupported protocol: %s", task.Protocol),
						StatusCode: http.StatusBadRequest,
					}, nil
				}
				return invoker.Execute(ctx, svcCtx, task), nil
			},
		}
		if task.Timeout > 0 {
			at.Timeout = time.Duration(task.Timeout) * time.Millisecond
		}
		antsxTasks = append(antsxTasks, at)
	}

	var settled []antsx.SettledResult[*Result]
	if maxConcurrency > 0 {
		reactor, err := antsx.NewReactor(int(maxConcurrency))
		if err != nil {
			results := make([]*Result, len(tasks))
			for i, t := range tasks {
				results[i] = &Result{ID: t.ID, Error: err.Error(), StatusCode: http.StatusInternalServerError}
			}
			return results
		}
		defer reactor.Release()
		settled = antsx.InvokeAllSettledWithReactor(ctx, reactor, antsxTasks...)
	} else {
		settled = antsx.InvokeAllSettled(ctx, antsxTasks...)
	}

	results := make([]*Result, len(settled))
	for i, sr := range settled {
		if sr.Err != nil {
			statusCode := int32(http.StatusInternalServerError)
			if errors.Is(sr.Err, context.DeadlineExceeded) || errors.Is(sr.Err, context.Canceled) {
				statusCode = http.StatusRequestTimeout
			}
			results[i] = &Result{
				ID:         tasks[i].ID,
				Error:      sr.Err.Error(),
				StatusCode: statusCode,
			}
		} else {
			results[i] = sr.Val
		}
		results[i].CostFormatted = netx.FormatCostMs(results[i].CostMs)
		if debug {
			r := results[i]
			if r.Error != "" {
				logx.WithContext(ctx).Infof("[invoke-debug] id=%s protocol=%s success=%v status=%d cost=%s error=%s",
					r.ID, tasks[i].Protocol, r.Success, r.StatusCode, r.CostFormatted, r.Error)
			} else if tasks[i].Protocol == "grpc" {
				logx.WithContext(ctx).Infof("[invoke-debug] id=%s protocol=grpc success=%v status=%d cost=%s data=%s",
					r.ID, r.Success, r.StatusCode, r.CostFormatted, RawProtoToJSON(r.Data))
			} else {
				logx.WithContext(ctx).Infof("[invoke-debug] id=%s protocol=http success=%v status=%d cost=%s data=%s",
					r.ID, r.Success, r.StatusCode, r.CostFormatted, string(r.Data))
			}
		}
	}
	return results
}
