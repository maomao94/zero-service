package logic

import (
	"context"
	"fmt"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/model"

	"github.com/doug-martin/goqu/v9"
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
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	query := l.svcCtx.Database.From(goqu.I("plan_batch").As("pb"))
	if len(in.PlanType) > 0 {
		query = query.LeftJoin(goqu.I("plan").As("p"), goqu.On(goqu.I("pb.plan_pk").Eq(goqu.I("p.id")))).
			Where(goqu.I("p.plan_type").Eq(in.PlanType))
	}
	if in.PlanId != "" {
		query = query.Where(goqu.I("pb.plan_id").Eq(in.PlanId))
	}
	if in.BatchId != "" {
		query = query.Where(goqu.I("pb.batch_id").Eq(in.BatchId))
	}
	if len(in.Status) > 0 {
		statusInterface := make([]interface{}, len(in.Status))
		for i, status := range in.Status {
			statusInterface[i] = int64(status)
		}
		query = query.Where(goqu.I("pb.status").In(statusInterface...))
	}
	query = query.Where(goqu.I("pb.del_state").Eq(0))

	countQuery := query.Select(goqu.COUNT("pb.id"))
	countSQL, countArgs, err := countQuery.ToSQL()
	if err != nil {
		return nil, err
	}
	var total int64
	err = l.svcCtx.SqlConn.QueryRowCtx(l.ctx, &total, countSQL, countArgs...)
	if err != nil {
		return nil, err
	}

	// 构建分页查询
	dataQuery := query.Select(goqu.I("pb.*")).
		Order(goqu.I("pb.plan_trigger_time").Asc(), goqu.I("pb.id").Desc()).
		Limit(uint(in.PageSize)).
		Offset(uint((in.PageNum - 1) * in.PageSize))

	dataSQL, dataArgs, err := dataQuery.ToSQL()
	if err != nil {
		return nil, err
	}
	var planBatches []*model.PlanBatch
	err = l.svcCtx.SqlConn.QueryRowPartialCtx(l.ctx, &planBatches, dataSQL, dataArgs...)
	if err != nil {
		return nil, err
	}

	// 构建响应
	resp := &trigger.ListPlanBatchesRes{
		PlanBatches: make([]*trigger.PbPlanBatch, 0, len(planBatches)),
		Total:       total,
	}

	// 转换计划批次列表
	for _, planBatch := range planBatches {
		// 获取批次执行项状态统计
		statusCounts, err := l.svcCtx.PlanExecItemModel.GetBatchStatusCounts(l.ctx, planBatch.Id)
		if err != nil {
			return nil, err
		}

		// 获取批次总执行项数
		totalExecItems, err := l.svcCtx.PlanExecItemModel.GetBatchTotalExecItems(l.ctx, planBatch.Id)
		if err != nil {
			return nil, err
		}
		statusCountMap := make(map[string]int64)

		// 统计各状态数量
		for _, sc := range statusCounts {
			statusStr := fmt.Sprintf("%d", sc.Status)
			statusCountMap[statusStr] = sc.Count
		}

		pbPlanBatch := &trigger.PbPlanBatch{
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
