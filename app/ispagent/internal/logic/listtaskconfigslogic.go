package logic

import (
	"context"
	"strings"

	"zero-service/app/ispagent/internal/svc"
	"zero-service/app/ispagent/ispagent"
	"zero-service/app/ispagent/model/gormmodel"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListTaskConfigsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListTaskConfigsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTaskConfigsLogic {
	return &ListTaskConfigsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ListTaskConfigs 分页查询任务配置，支持 task_code 模糊匹配和 substation_code 精确过滤。
func (l *ListTaskConfigsLogic) ListTaskConfigs(in *ispagent.ListTaskConfigsReq) (*ispagent.ListTaskConfigsRes, error) {
	if l.svcCtx.DB == nil {
		return &ispagent.ListTaskConfigsRes{}, nil
	}

	page := int(in.GetPage())
	if page < 1 {
		page = 1
	}
	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = 20
	}

	db := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.GormTaskConfig{})

	if code := in.GetSubstationCode(); code != "" {
		db = db.Where("substation_code = ?", code)
	}
	if like := in.GetTaskCode(); like != "" {
		db = db.Where("task_code LIKE ?", "%"+strings.TrimSpace(like)+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	var records []gormmodel.GormTaskConfig
	if err := db.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, err
	}

	items := make([]*ispagent.TaskConfigItem, 0, len(records))
	for i := range records {
		r := &records[i]
		item := &ispagent.TaskConfigItem{
			Id:                  r.Id,
			TaskCode:            r.TaskCode,
			TaskName:            r.TaskName,
			Priority:            int32(r.Priority),
			RruleStr:            r.RRuleStr,
			Status:              int32(r.Status),
			SubstationCode:      r.SubstationCode,
			PatrolType:          r.PatrolType,
			DeviceLevel:         int32(r.DeviceLevel),
			DeviceList:          r.DeviceList,
			IspEnable:           r.IsEnable,
			IspCreator:          r.IspCreator,
			IspCreateTime:       r.IspCreateTime,
			FixedStartTime:      r.FixedStartTime,
			CycleMonth:          r.CycleMonth,
			CycleWeek:           r.CycleWeek,
			CycleExecuteTime:    r.CycleExecuteTime,
			CycleStartTime:      r.CycleStartTime,
			CycleEndTime:        r.CycleEndTime,
			IntervalNumber:      r.IntervalNumber,
			IntervalType:        r.IntervalType,
			IntervalExecuteTime: r.IntervalExecuteTime,
			IntervalStartTime:   r.IntervalStartTime,
			IntervalEndTime:     r.IntervalEndTime,
			InvalidStartTime:    r.InvalidStartTime,
			InvalidEndTime:      r.InvalidEndTime,
			NextRun:             carbon.CreateFromStdTime(r.NextRun).ToDateTimeString(),
		}
		if r.LastRun.Valid {
			item.LastRun = carbon.CreateFromStdTime(r.LastRun.Time).ToDateTimeString()
		}
		items = append(items, item)
	}

	return &ispagent.ListTaskConfigsRes{Total: total, Items: items}, nil
}
