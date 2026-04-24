package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ResumeFlightTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResumeFlightTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResumeFlightTaskLogic {
	return &ResumeFlightTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ResumeFlightTask 恢复已暂停的飞行任务。
func (l *ResumeFlightTaskLogic) ResumeFlightTask(in *djigateway.DeviceSnReq) (*djigateway.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.ResumeFlightTask(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[flight-task] resume flight task failed: %v", err)
		return &djigateway.CommonRes{Code: -1, Message: err.Error(), Tid: tid}, nil
	}
	return &djigateway.CommonRes{Code: 0, Message: "success", Tid: tid}, nil
}
