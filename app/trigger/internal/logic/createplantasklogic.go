package logic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/model"

	"github.com/dromara/carbon/v2"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/teambition/rrule-go"
	"google.golang.org/protobuf/encoding/protojson"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
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
	db := l.svcCtx.DB.WithContext(l.ctx).DB
	var plan gormmodel.Plan
	err = db.Where("plan_id = ?", in.PlanId).First(&plan).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询计划失败")
		}
	} else {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_ALREADY_EXIST)
	}
	calculated, err := NewCalcPlanTaskDateLogic(l.ctx, l.svcCtx).CalcPlanTaskDate(&trigger.CalcPlanTaskDateReq{
		StartTime:    in.StartTime,
		EndTime:      in.EndTime,
		Rule:         in.Rule,
		ExcludeDates: in.ExcludeDates,
	})
	if err != nil {
		return nil, err
	}
	rruleStr := calculated.GetRruleStr()
	set, err := rrule.StrToRRuleSet(rruleStr)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, err, "解析计划规则失败")
	}
	rule := set.GetRRule()
	if rule == nil {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "计划规则缺少 RRULE")
	}
	dates := make([]time.Time, 0, len(calculated.PlanDates))
	for _, value := range calculated.PlanDates {
		date := carbon.ParseByLayout(value, carbon.DateTimeLayout)
		if date.Error != nil || date.IsInvalid() {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, date.Error, "解析计划日期失败")
		}
		dates = append(dates, date.StdTime())
	}
	if !in.SkipTimeFilter {
		now := time.Now()
		validDates := make([]time.Time, 0, len(dates))
		for _, date := range dates {
			if !date.Before(now) {
				validDates = append(validDates, date)
			}
		}
		dates = validDates
		if len(dates) == 0 {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "计划任务时间段内没有触发时间")
		}
	}
	if len(dates) > 5000/len(in.ExecItems) {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "计划任务时间段内调度项过多")
	}
	ruleJSON, err := protojson.Marshal(in.Rule)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, err, "序列化计划规则失败")
	}
	currentUserId := tool.GetCurrentUserId(l.ctx, nil)

	var insertPlan = gormmodel.Plan{
		CreateUser:       sql.NullString{String: currentUserId, Valid: currentUserId != ""},
		UpdateUser:       sql.NullString{String: currentUserId, Valid: currentUserId != ""},
		DeptCode:         sql.NullString{String: in.DeptCode, Valid: in.DeptCode != ""},
		PlanId:           in.PlanId,
		PlanName:         sql.NullString{String: in.PlanName, Valid: in.PlanName != ""},
		Type:             sql.NullString{String: in.Type, Valid: in.Type != ""},
		GroupId:          sql.NullString{String: in.GroupId, Valid: in.GroupId != ""},
		RecurrenceRule:   string(ruleJSON),
		RRuleStr:         rruleStr,
		StartTime:        rule.Options.Dtstart,
		EndTime:          rule.Options.Until,
		Status:           model.PlanStatusEnabled,
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
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&insertPlan).Error; err != nil {
			return err
		}
		for _, d := range dates {
			batchId, idErr := tool.SimpleUUID()
			if idErr != nil {
				return idErr
			}
			dStr := carbon.NewCarbon(d).Format("Y-m-d H:i")
			batchName := fmt.Sprintf("%s@%s", in.PlanName, dStr)
			batchNum, nextIdErr := l.svcCtx.IdUtil.NextId("P", l.svcCtx.Config.Name)
			if nextIdErr != nil {
				return nextIdErr
			}
			if len(in.BatchNumPrefix) >= 0 {
				batchNum = fmt.Sprintf("%s%s", in.BatchNumPrefix, strutil.After(batchNum, "P"))
			}
			batch := gormmodel.PlanBatch{
				CreateUser:      sql.NullString{String: currentUserId, Valid: currentUserId != ""},
				UpdateUser:      sql.NullString{String: currentUserId, Valid: currentUserId != ""},
				DeptCode:        sql.NullString{String: in.DeptCode, Valid: in.DeptCode != ""},
				PlanPk:          insertPlan.Id,
				PlanId:          in.PlanId,
				BatchId:         batchId,
				BatchName:       sql.NullString{String: batchName, Valid: true},
				BatchNum:        sql.NullString{String: batchNum, Valid: true},
				Status:          model.PlanStatusEnabled,
				PlanTriggerTime: sql.NullTime{Time: d, Valid: true},
				FinishedTime:    sql.NullTime{},
				Ext1:            sql.NullString{String: in.Ext1, Valid: in.Ext1 != ""},
				Ext2:            sql.NullString{String: in.Ext2, Valid: in.Ext2 != ""},
				Ext3:            sql.NullString{String: in.Ext3, Valid: in.Ext3 != ""},
				Ext4:            sql.NullString{String: in.Ext4, Valid: in.Ext4 != ""},
				Ext5:            sql.NullString{String: in.Ext5, Valid: in.Ext5 != ""},
			}
			if err := tx.Create(&batch).Error; err != nil {
				return err
			}
			batchCnt++
			itemIndex := 0
			for _, item := range in.ExecItems {
				execId, idErr := tool.SimpleUUID()
				if idErr != nil {
					return idErr
				}
				nextTriggerTime := d
				switch in.IntervalType {
				case 1:
					nextTriggerTime = d.Add(time.Duration(itemIndex*int(in.IntervalTime)) * time.Millisecond)
				case 2:
					if in.IntervalTime > 0 {
						offset := l.svcCtx.UnstableExpiry.AroundDuration(time.Duration(in.IntervalTime) * time.Millisecond)
						nextTriggerTime = d.Add(offset)
					}
				}
				planItem := gormmodel.PlanExecItem{
					CreateUser:       sql.NullString{String: currentUserId, Valid: currentUserId != ""},
					UpdateUser:       sql.NullString{String: currentUserId, Valid: currentUserId != ""},
					DeptCode:         sql.NullString{String: in.DeptCode, Valid: in.DeptCode != ""},
					PlanPk:           insertPlan.Id,
					PlanId:           in.PlanId,
					BatchPk:          batch.Id,
					BatchId:          batchId,
					ExecId:           execId,
					ItemId:           item.ItemId,
					ItemType:         sql.NullString{String: item.ItemType, Valid: item.ItemType != ""},
					ItemName:         sql.NullString{String: item.ItemName, Valid: item.ItemName != ""},
					ItemRowId:        int64(itemIndex),
					PointId:          sql.NullString{String: item.PointId, Valid: item.PointId != ""},
					Payload:          item.Payload,
					RequestTimeout:   item.RequestTimeout,
					PlanTriggerTime:  d,
					NextTriggerTime:  nextTriggerTime,
					LastTriggerTime:  sql.NullTime{},
					TriggerCount:     0,
					Status:           model.StatusWaiting,
					LastResult:       sql.NullString{},
					LastMessage:      sql.NullString{},
					LastReason:       sql.NullString{},
					TerminatedReason: sql.NullString{},
					PausedTime:       sql.NullTime{},
					PausedReason:     sql.NullString{},
					Ext1:             sql.NullString{String: item.Ext1, Valid: item.Ext1 != ""},
					Ext2:             sql.NullString{String: item.Ext2, Valid: item.Ext2 != ""},
					Ext3:             sql.NullString{String: item.Ext3, Valid: item.Ext3 != ""},
					Ext4:             sql.NullString{String: item.Ext4, Valid: item.Ext4 != ""},
					Ext5:             sql.NullString{String: item.Ext5, Valid: item.Ext5 != ""},
				}
				itemIndex++
				if err := tx.Create(&planItem).Error; err != nil {
					return err
				}
				execCnt++
			}
		}
		return nil
	})
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "创建计划事务失败")
	}
	return &trigger.CreatePlanTaskRes{
		Id:       insertPlan.Id,
		PlanId:   insertPlan.PlanId,
		BatchCnt: batchCnt,
		ExecCnt:  execCnt,
	}, nil
}
