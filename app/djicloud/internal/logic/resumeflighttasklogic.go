package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"

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
func (l *ResumeFlightTaskLogic) ResumeFlightTask(in *djicloud.DeviceSnReq) (*djicloud.CommonRes, error) {
	tid, err := l.svcCtx.DjiClient.ResumeFlightTask(l.ctx, in.DeviceSn)
	if err != nil {
		l.Errorf("[flight-task] resume flight task failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
