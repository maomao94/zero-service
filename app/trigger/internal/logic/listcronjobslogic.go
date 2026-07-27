package logic

import (
	"context"

	"zero-service/app/trigger/internal/cronjob"
	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/gormx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListCronJobsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListCronJobsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListCronJobsLogic {
	return &ListCronJobsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 分页获取 Cron Job 列表
func (l *ListCronJobsLogic) ListCronJobs(in *trigger.ListCronJobsReq) (*trigger.ListCronJobsRes, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}
	query := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.CronJob{})
	if in.TaskCode != "" {
		query = query.Where("task_code LIKE ?", "%"+in.TaskCode+"%")
	}
	if in.TaskName != "" {
		query = query.Where("task_name LIKE ?", "%"+in.TaskName+"%")
	}
	statuses := make([]int, len(in.Status))
	for i, status := range in.Status {
		statuses[i] = int(status)
	}
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	if in.DeptCode != "" {
		query = query.Where("dept_code = ?", in.DeptCode)
	}
	if in.Type != "" {
		query = query.Where("type = ?", in.Type)
	}
	if in.GroupId != "" {
		query = query.Where("group_id = ?", in.GroupId)
	}

	var records []gormmodel.CronJob
	page, err := gormx.QueryPage(
		query.Order("create_time DESC, id DESC"),
		in.PageNum,
		in.PageSize,
		&records,
	)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询 Cron Job 列表失败")
	}
	response := &trigger.ListCronJobsRes{
		CronJobs: make([]*trigger.CronJobPb, 0, len(records)),
		Total:    page.Total,
	}
	for i := range records {
		task, err := cronjob.ToTaskConfig(&records[i])
		if err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "解析 Cron Job 列表失败")
		}
		job, err := cronjob.ToProto(task)
		if err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "解析 Cron Job 列表失败")
		}
		response.CronJobs = append(response.CronJobs, job)
	}
	return response, nil
}
