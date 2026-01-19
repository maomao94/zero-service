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
	rruleOption, err := NewCalcPlanTaskDateLogic(l.ctx, l.svcCtx).ConvertToRRuleOption(in.Rule, startTime, endTime)
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
		DeptCode:         sql.NullString{String: in.DeptCode, Valid: in.DeptCode != ""},
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
			dStr := carbon.NewCarbon(d).Format("Y-m-d H:i")
			batchName := fmt.Sprintf("%s@%s", in.PlanName, dStr)
			batch := model.PlanBatch{
				CreateUser:      sql.NullString{String: currentUserId, Valid: currentUserId != ""},
				UpdateUser:      sql.NullString{String: currentUserId, Valid: currentUserId != ""},
				DeptCode:        sql.NullString{String: in.DeptCode, Valid: in.DeptCode != ""},
				PlanPk:          insertPlan.Id,
				PlanId:          in.PlanId,
				BatchId:         batchId,
				BatchName:       sql.NullString{String: batchName, Valid: true},
				Status:          int64(model.PlanStatusEnabled),
				PlanTriggerTime: sql.NullTime{Time: d, Valid: true},
				CompletedTime:   sql.NullTime{},
				Ext1:            sql.NullString{String: in.Ext1, Valid: in.Ext1 != ""},
				Ext2:            sql.NullString{String: in.Ext2, Valid: in.Ext2 != ""},
				Ext3:            sql.NullString{String: in.Ext3, Valid: in.Ext3 != ""},
				Ext4:            sql.NullString{String: in.Ext4, Valid: in.Ext4 != ""},
				Ext5:            sql.NullString{String: in.Ext5, Valid: in.Ext5 != ""},
			}
			batchResult, err := l.svcCtx.PlanBatchModel.Insert(ctx, tx, &batch)
			if err != nil {
				return err
			}
			batchPk, _ := batchResult.LastInsertId()
			batchCnt++
			itemIndex := 0
			for _, item := range in.ExecItems {
				execId, _ := tool.SimpleUUID()
				nextTriggerTime := d
				switch item.IntervalType {
				case 1:
					nextTriggerTime = d.Add(time.Duration(itemIndex*int(item.IntervalTime)) * time.Millisecond)
					itemIndex++
				case 2:
					if item.IntervalTime > 0 {
						offset := l.svcCtx.UnstableExpiry.AroundDuration(time.Duration(item.IntervalTime) * time.Millisecond)
						nextTriggerTime = d.Add(offset)
					}
				}
				planItem := model.PlanExecItem{
					CreateUser:       sql.NullString{String: currentUserId, Valid: currentUserId != ""},
					UpdateUser:       sql.NullString{String: currentUserId, Valid: currentUserId != ""},
					DeptCode:         sql.NullString{String: in.DeptCode, Valid: in.DeptCode != ""},
					PlanPk:           insertPlan.Id,
					PlanId:           in.PlanId,
					BatchPk:          batchPk,
					BatchId:          batchId,
					ExecId:           execId,
					ItemId:           item.ItemId,
					ItemType:         sql.NullString{String: item.ItemType, Valid: item.ItemType != ""},
					ItemName:         sql.NullString{String: item.ItemName, Valid: item.ItemName != ""},
					PointId:          sql.NullString{String: item.PointId, Valid: item.PointId != ""},
					ServiceAddr:      item.ServiceAddr,
					Payload:          item.Payload,
					RequestTimeout:   item.RequestTimeout,
					PlanTriggerTime:  d,
					NextTriggerTime:  nextTriggerTime,
					LastTriggerTime:  sql.NullTime{},
					TriggerCount:     0,
					Status:           int64(model.StatusWaiting),
					LastResult:       sql.NullString{},
					LastMessage:      sql.NullString{},
					LastReason:       sql.NullString{},
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
