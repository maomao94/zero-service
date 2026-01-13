package logic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
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
	if len(dates) == 0 {
		return nil, fmt.Errorf("计划任务时间段内没有触发时间")
	}
	dates = validDates
	rule, _ := jsonx.Marshal(in.Rule)
	var insertPlan = &model.Plan{
		PlanId:           in.PlanId,
		PlanName:         in.PlanName,
		Type:             in.Type,
		GroupId:          in.GroupId,
		RecurrenceRule:   string(rule),
		StartTime:        rruleOption.Dtstart,
		EndTime:          rruleOption.Until,
		Status:           1,
		IsTerminated:     0,
		IsPaused:         0,
		TerminatedTime:   sql.NullTime{},
		TerminatedReason: "",
		PausedTime:       sql.NullTime{},
		PausedReason:     "",
		Description:      in.Description,
	}
	err = l.svcCtx.PlanModel.Trans(l.ctx, func(ctx context.Context, tx sqlx.Session) error {
		result, transErr := l.svcCtx.PlanModel.Insert(ctx, tx, insertPlan)
		if transErr != nil {
			return transErr
		}
		insertPlan.Id, _ = result.LastInsertId()
		for _, d := range dates {
			for _, item := range in.ExecItems {
				planItem := model.PlanExecItem{
					CreateUser:       in.CurrentUser.UserId,
					UpdateUser:       in.CurrentUser.UserId,
					PlanId:           in.PlanId,
					PlanPk:           insertPlan.Id,
					ItemId:           item.ItemId,
					ItemName:         item.ItemName,
					PointId:          item.PointId,
					ServiceAddr:      item.ServiceAddr,
					Payload:          item.Payload,
					RequestTimeout:   item.RequestTimeout,
					PlanTriggerTime:  d,
					NextTriggerTime:  d,
					LastTriggerTime:  sql.NullTime{},
					TriggerCount:     0,
					Status:           0,
					LastResult:       "",
					LastMsg:          "",
					IsTerminated:     0,
					TerminatedTime:   sql.NullTime{},
					TerminatedReason: "",
					IsPaused:         0,
					PausedTime:       sql.NullTime{},
					PausedReason:     "",
				}
				_, err = l.svcCtx.PlanExecItemModel.Insert(l.ctx, tx, &planItem)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &trigger.CreatePlanTaskRes{
		Id: insertPlan.Id,
	}, nil
}

func (l *CreatePlanTaskLogic) convertToRRuleOption(planRule *trigger.PbPlanRule, startTime, endTime *carbon.Carbon) (rrule.ROption, error) {
	// 设置默认的rrule选项
	opts := rrule.ROption{
		Freq:     rrule.Frequency(planRule.Freq),
		Dtstart:  startTime.StdTime(),
		Until:    endTime.StdTime(),
		Byhour:   []int{int(planRule.Hour)},
		Byminute: []int{int(planRule.Minute)},
		Bysecond: []int{0}, // 默认秒为0
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
