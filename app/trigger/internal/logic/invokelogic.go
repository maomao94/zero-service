package logic

import (
	"context"

	"zero-service/app/trigger/internal/invoke"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/zeromicro/go-zero/core/logx"
)

type InvokeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewInvokeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InvokeLogic {
	return &InvokeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *InvokeLogic) Invoke(in *trigger.InvokeReq) (*trigger.InvokeRes, error) {
	tasks := make([]*invoke.Task, 0, len(in.Tasks))
	for _, t := range in.Tasks {
		tasks = append(tasks, &invoke.Task{
			ID:         t.Id,
			Protocol:   t.Protocol,
			Timeout:    t.Timeout,
			URL:        t.Url,
			HTTPMethod: t.HttpMethod,
			Headers:    t.Headers,
			Body:       t.Body,
			GrpcServer: t.GrpcServer,
			Method:     t.Method,
			Payload:    t.Payload,
		})
	}

	results := invoke.Run(l.ctx, l.svcCtx, tasks, in.MaxConcurrency, in.Debug)

	pbResults := make([]*trigger.InvokeTaskResultPb, 0, len(results))
	for _, r := range results {
		pbResults = append(pbResults, &trigger.InvokeTaskResultPb{
			Id:            r.ID,
			Success:       r.Success,
			StatusCode:    r.StatusCode,
			Error:         r.Error,
			Data:          r.Data,
			CostMs:        r.CostMs,
			CostFormatted: r.CostFormatted,
		})
	}

	return &trigger.InvokeRes{Results: pbResults}, nil
}
