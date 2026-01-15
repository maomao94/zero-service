package logic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"zero-service/common/tool"
	"zero-service/model"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"

	"github.com/dromara/carbon/v2"
	"github.com/teambition/rrule-go"
	"github.com/zeromicro/go-zero/core/jsonx"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type CreatePlanTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreatePlanTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreatePlanTaskLogic {
	return &CreatePlanTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 创建计划任务
func (l *CreatePlanTaskLogic) CreatePlanTask(in *trigger.CreatePlanTaskReq) (*trigger.CreatePlanTaskRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}
	for _, item := range in.ExecItems {
		matsh := GrpcServerRegexp.MatchString(item.ServiceAddr)
		if !matsh {
			return nil, errors.New("grpcServer is invalid")
		}
	}
	querPlan, err := l.svcCtx.PlanModel.FindOneByPlanId(l.ctx, in.PlanId)
	if err != nil {
		if err != sqlx.ErrNotFound {
			return nil, err
		}
	}
	if querPlan != nil {
		return nil, fmt.Errorf("计划任务已存在")
	}
	if len(in.StartTime) == 0 {
		in.StartTime = fmt.Sprintf("%d-1-1 00:00:00", time.Now().Year())
	}
	startTime := carbon.Parse(in.StartTime)
	if startTime.Error != nil {
		return nil, startTime.Error
	}
	if len(in.EndTime) == 0 {
		in.EndTime = fmt.Sprintf("%d-12-31 23:59:59", startTime.Year())
	}
	endTime := carbon.Parse(in.EndTime)
	if endTime.Error != nil {
		return nil, endTime.Error
	}
	if endTime.Lt(startTime) {
		return nil, errors.New("结束时间必须晚于开始时间")
	}
	if endTime.Gt(startTime.AddYears(3)) {
		return nil, errors.New("计划时间跨度不能超过3年")
	}
	rruleOption, err := l.convertToRRuleOption(in.Rule, startTime, endTime)
	if err != nil {
		return nil, err
	}
	r, err := rrule.NewRRule(rruleOption)
	if err != nil {
		return nil, err
	}
	// 获取所有触发时间
	dates := r.All()
	// 过滤掉小于当前时间的触发时间
	now := time.Now()
	var validDates []time.Time = make([]time.Time, 0)
	for _, d := range dates {
		if !d.Before(now) {
			validDates = append(validDates, d)
		}
	}
	dates = validDates
	if len(dates) == 0 {
		return nil, fmt.Errorf("计划任务时间段内没有触发时间")
	}
	if len(dates)*len(in.ExecItems) > 5000 {
		return nil, fmt.Errorf("计划任务时间段内调度项过多")
	}
	rule, _ := jsonx.Marshal(in.Rule)
	currentUserId := tool.GetCurrentUserId(in.CurrentUser)

	var insertPlan = &model.Plan{
		CreateUser:       sql.NullString{String: currentUserId, Valid: currentUserId != ""},
		UpdateUser:       sql.NullString{String: currentUserId, Valid: currentUserId != ""},
		PlanId:           in.PlanId,
		PlanName:         sql.NullString{String: in.PlanName, Valid: in.PlanName != ""},
		Type:             sql.NullString{String: in.Type, Valid: in.Type != ""},
		GroupId:          sql.NullString{String: in.GroupId, Valid: in.GroupId != ""},
		RecurrenceRule:   string(rule),
		StartTime:        rruleOption.Dtstart,
		EndTime:          rruleOption.Until,
		Status:           int64(model.PlanStatusEnabled),
		TerminatedTime:   sql.NullTime{},
		TerminatedReason: sql.NullString{},
		PausedTime:       sql.NullTime{},
		PausedReason:     sql.NullString{},
		Description:      sql.NullString{String: in.Description, Valid: in.Description != ""},
		Ext1:             sql.NullString{String: in.Ext1, Valid: in.Ext1 != ""},
		Ext2:             sql.NullString{String: in.Ext2, Valid: in.Ext2 != ""},
		Ext3:             sql.NullString{String: in.Ext3, Valid: in.Ext3 != ""},
		Ext4:             sql.NullString{String: in.Ext4, Valid: in.Ext4 != ""},
		Ext5:             sql.NullString{String: in.Ext5, Valid: in.Ext5 != ""},
	}
	var batchCnt int64 = 0
	var execCnt int64 = 0
	err = l.svcCtx.PlanModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		result, transErr := l.svcCtx.PlanModel.Insert(ctx, tx, insertPlan)
		if transErr != nil {
			return transErr
		}
		insertPlan.Id, _ = result.LastInsertId()
		for _, d := range dates {
			batchId, _ := tool.SimpleUUID()
			dStr := carbon.NewCarbon(d).ToDateTimeString()
			batchName := fmt.Sprintf("%s-%s", in.PlanName, dStr)
			batch := model.PlanBatch{
				CreateUser:    sql.NullString{String: currentUserId, Valid: currentUserId != ""},
				UpdateUser:    sql.NullString{String: currentUserId, Valid: currentUserId != ""},
				PlanPk:        insertPlan.Id,
				PlanId:        in.PlanId,
				BatchId:       batchId,
				BatchName:     sql.NullString{String: batchName, Valid: true},
				Status:        int64(model.PlanStatusEnabled),
				CompletedTime: sql.NullTime{},
				Ext1:          sql.NullString{String: in.Ext1, Valid: in.Ext1 != ""},
				Ext2:          sql.NullString{String: in.Ext2, Valid: in.Ext2 != ""},
				Ext3:          sql.NullString{String: in.Ext3, Valid: in.Ext3 != ""},
				Ext4:          sql.NullString{String: in.Ext4, Valid: in.Ext4 != ""},
				Ext5:          sql.NullString{String: in.Ext5, Valid: in.Ext5 != ""},
			}
			batchResult, err := l.svcCtx.PlanBatchModel.Insert(ctx, tx, &batch)
			if err != nil {
				return err
			}
			batchPk, _ := batchResult.LastInsertId()
			batchCnt++
			for _, item := range in.ExecItems {
				planItem := model.PlanExecItem{
					CreateUser:       sql.NullString{String: currentUserId, Valid: currentUserId != ""},
					UpdateUser:       sql.NullString{String: currentUserId, Valid: currentUserId != ""},
					PlanPk:           insertPlan.Id,
					PlanId:           in.PlanId,
					BatchPk:          batchPk,
					BatchId:          batchId,
					ItemId:           item.ItemId,
					ItemName:         sql.NullString{String: item.ItemName, Valid: item.ItemName != ""},
					PointId:          sql.NullString{String: item.PointId, Valid: item.PointId != ""},
					ServiceAddr:      item.ServiceAddr,
					Payload:          item.Payload,
					RequestTimeout:   item.RequestTimeout,
					PlanTriggerTime:  d,
					NextTriggerTime:  d,
					LastTriggerTime:  sql.NullTime{},
					TriggerCount:     0,
					Status:           int64(model.StatusWaiting),
					LastResult:       sql.NullString{},
					LastMsg:          sql.NullString{},
					TerminatedTime:   sql.NullTime{},
					TerminatedReason: sql.NullString{},
					PausedTime:       sql.NullTime{},
					PausedReason:     sql.NullString{},
					CompletedTime:    sql.NullTime{},
					Ext1:             sql.NullString{String: item.Ext1, Valid: item.Ext1 != ""},
					Ext2:             sql.NullString{String: item.Ext2, Valid: item.Ext2 != ""},
					Ext3:             sql.NullString{String: item.Ext3, Valid: item.Ext3 != ""},
					Ext4:             sql.NullString{String: item.Ext4, Valid: item.Ext4 != ""},
					Ext5:             sql.NullString{String: item.Ext5, Valid: item.Ext5 != ""},
				}
				_, err = l.svcCtx.PlanExecItemModel.Insert(l.ctx, tx, &planItem)
				if err != nil {
					return err
				}
				execCnt++
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &trigger.CreatePlanTaskRes{
		Id:       insertPlan.Id,
		PlanId:   insertPlan.PlanId,
		BatchCnt: batchCnt,
		ExecCnt:  execCnt,
	}, nil
}

func (l *CreatePlanTaskLogic) convertToRRuleOption(planRule *trigger.PbPlanRule, startTime, endTime *carbon.Carbon) (rrule.ROption, error) {
	// 设置默认的rrule选项
	opts := rrule.ROption{
		Freq:     rrule.Frequency(planRule.Freq),
		Dtstart:  startTime.StdTime(),
		Until:    endTime.StdTime(),
		Bysecond: []int{0}, // 默认秒为0
	}

	// 设置小时
	if len(planRule.Hours) > 0 {
		byhour := make([]int, len(planRule.Hours))
		for i, h := range planRule.Hours {
			byhour[i] = int(h)
		}
		opts.Byhour = byhour
	}

	// 设置分钟
	if len(planRule.Minutes) > 0 {
		byminute := make([]int, len(planRule.Minutes))
		for i, m := range planRule.Minutes {
			byminute[i] = int(m)
		}
		opts.Byminute = byminute
	}

	// 设置月份
	if len(planRule.Month) > 0 {
		bymonth := make([]int, len(planRule.Month))
		for i, m := range planRule.Month {
			bymonth[i] = int(m)
		}
		opts.Bymonth = bymonth
	}

	// 设置月中的天数
	if len(planRule.Day) > 0 {
		bymonthday := make([]int, len(planRule.Day))
		for i, d := range planRule.Day {
			bymonthday[i] = int(d)
		}
		opts.Bymonthday = bymonthday
	}

	// 设置星期几
	if len(planRule.Week) > 0 {
		byweekday := make([]rrule.Weekday, len(planRule.Week))
		for i, w := range planRule.Week {
			switch w {
			case 1:
				byweekday[i] = rrule.MO
			case 2:
				byweekday[i] = rrule.TU
			case 3:
				byweekday[i] = rrule.WE
			case 4:
				byweekday[i] = rrule.TH
			case 5:
				byweekday[i] = rrule.FR
			case 6:
				byweekday[i] = rrule.SA
			case 7:
				byweekday[i] = rrule.SU
			default:
				return rrule.ROption{}, fmt.Errorf("invalid week day: %d", w)
			}
		}
		opts.Byweekday = byweekday
	}

	return opts, nil
}
