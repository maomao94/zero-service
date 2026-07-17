package logic

import (
	"context"
	"fmt"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/gormx"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListPlanBatchesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListPlanBatchesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPlanBatchesLogic {
	return &ListPlanBatchesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 分页获取计划批次列表
func (l *ListPlanBatchesLogic) ListPlanBatches(in *trigger.ListPlanBatchesReq) (*trigger.ListPlanBatchesRes, error) {
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	db := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.PlanBatch{})
	if len(in.PlanType) > 0 {
		db = db.Joins("LEFT JOIN plan AS p ON p.id = plan_batch.plan_pk AND p.is_deleted = 0").
			Where("p.type = ?", in.PlanType)
	}
	if in.PlanId != "" {
		db = db.Where("plan_batch.plan_id = ?", in.PlanId)
	}
	if in.BatchId != "" {
		db = db.Where("plan_batch.batch_id = ?", in.BatchId)
	}
	if len(in.Status) > 0 {
		statusInts := make([]int64, len(in.Status))
		for i, s := range in.Status {
			statusInts[i] = int64(s)
		}
		db = db.Where("plan_batch.status IN ?", statusInts)
	}

	var planBatches []gormmodel.PlanBatch
	page, err := gormx.QueryPage(db.Order("plan_batch.plan_trigger_time ASC, plan_batch.id DESC"), int(in.PageNum), int(in.PageSize), &planBatches)
	if err != nil {
		return nil, err
	}

	resp := &trigger.ListPlanBatchesRes{
		PlanBatches: make([]*trigger.PlanBatchPb, 0, len(planBatches)),
		Total:       page.Total,
	}

	rawDB := l.svcCtx.DB.WithContext(l.ctx).DB
	for _, planBatch := range planBatches {
		statusCounts, err := gormmodel.GetBatchStatusCounts(l.ctx, rawDB, planBatch.Id)
		if err != nil {
			return nil, err
		}
		totalExecItems, err := gormmodel.GetBatchTotalExecItems(l.ctx, rawDB, planBatch.Id)
		if err != nil {
			return nil, err
		}
		statusCountMap := make(map[string]int64)
		for _, sc := range statusCounts {
			statusCountMap[fmt.Sprintf("%d", sc.Status)] = sc.Count
		}

		pbPlanBatch := &trigger.PlanBatchPb{
			CreateTime:       carbon.CreateFromStdTime(planBatch.CreateTime).ToDateTimeString(),
			UpdateTime:       carbon.CreateFromStdTime(planBatch.UpdateTime).ToDateTimeString(),
			CreateUser:       planBatch.CreateUser.String,
			UpdateUser:       planBatch.UpdateUser.String,
			DeptCode:         planBatch.DeptCode.String,
			Id:               planBatch.Id,
			PlanPk:           planBatch.PlanPk,
			PlanId:           planBatch.PlanId,
			BatchId:          planBatch.BatchId,
			BatchName:        planBatch.BatchName.String,
			BatchNum:         planBatch.BatchNum.String,
			Status:           int32(planBatch.Status),
			ScanFlg:          int32(planBatch.ScanFlg),
			PlanTriggerTime:  carbon.CreateFromStdTime(planBatch.PlanTriggerTime.Time).ToDateTimeString(),
			TerminatedReason: planBatch.TerminatedReason.String,
			PausedReason:     planBatch.PausedReason.String,
			StatusCountMap:   statusCountMap,
			ExecCnt:          totalExecItems,
			Ext1:             planBatch.Ext1.String,
			Ext2:             planBatch.Ext2.String,
			Ext3:             planBatch.Ext3.String,
			Ext4:             planBatch.Ext4.String,
			Ext5:             planBatch.Ext5.String,
		}
		if planBatch.PausedTime.Valid {
			pbPlanBatch.PausedTime = carbon.CreateFromStdTime(planBatch.PausedTime.Time).ToDateTimeString()
		}
		if planBatch.FinishedTime.Valid {
			pbPlanBatch.FinishedTime = carbon.CreateFromStdTime(planBatch.FinishedTime.Time).ToDateTimeString()
		}
		resp.PlanBatches = append(resp.PlanBatches, pbPlanBatch)
	}
	return resp, nil
}
