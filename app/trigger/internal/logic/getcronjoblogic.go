package logic

import (
	"context"
	"errors"

	"zero-service/app/trigger/internal/cronjob"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/crontask"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCronJobLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetCronJobLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCronJobLogic {
	return &GetCronJobLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取 Cron Job 详情
func (l *GetCronJobLogic) GetCronJob(in *trigger.GetCronJobReq) (*trigger.GetCronJobRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	task, err := l.svcCtx.CronJobStore.GetByID(l.ctx, in.JobId)
	if err != nil {
		if errors.Is(err, crontask.ErrNotFound) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST)
		}
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询 Cron Job 详情失败")
	}
	pbJob, err := cronjob.ToProto(task)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "解析 Cron Job 详情失败")
	}
	return &trigger.GetCronJobRes{CronJob: pbJob}, nil
}
