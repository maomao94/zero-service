package logic

import (
	"context"
	"encoding/json"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/model/gormmodel"
	"zero-service/app/trigger/trigger"
	"zero-service/common/gormx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/dromara/carbon/v2"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListPlansLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListPlansLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPlansLogic {
	return &ListPlansLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 分页获取计划列表
func (l *ListPlansLogic) ListPlans(in *trigger.ListPlansReq) (*trigger.ListPlansRes, error) {
	// 验证请求
	err := in.Validate()
	if err != nil {
		return nil, err
	}

	// 构建查询条件
	db := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.Plan{})
	if in.PlanId != "" {
		db = db.Where("plan_id = ?", in.PlanId)
	}
	if in.PlanName != "" {
		db = db.Where("plan_name LIKE ?", "%"+in.PlanName+"%")
	}
	if in.Type != "" {
		db = db.Where("type = ?", in.Type)
	}
	if len(in.Status) > 0 {
		statusInts := make([]int, len(in.Status))
		for i, status := range in.Status {
			statusInts[i] = int(status)
		}
		db = db.Where("status IN ?", statusInts)
	}

	var plans []gormmodel.Plan
	page, err := gormx.QueryPage(db.Order("id DESC"), int(in.PageNum), int(in.PageSize), &plans)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询计划列表失败")
	}

	// 构建响应
	resp := &trigger.ListPlansRes{
		Plans: make([]*trigger.PlanPb, 0, len(plans)),
		Total: page.Total,
	}

	// 转换计划列表
	rawDB := l.svcCtx.DB.WithContext(l.ctx).DB
	for i := range plans {
		// 解析规则
		var pbRule trigger.PlanRulePb
		if err := json.Unmarshal([]byte(plans[i].RecurrenceRule), &pbRule); err != nil {
			continue
		}

		pbPlan := &trigger.PlanPb{
			CreateTime:       carbon.CreateFromStdTime(plans[i].CreateTime).ToDateTimeString(),
			UpdateTime:       carbon.CreateFromStdTime(plans[i].UpdateTime).ToDateTimeString(),
			CreateUser:       plans[i].CreateUser.String,
			UpdateUser:       plans[i].UpdateUser.String,
			DeptCode:         plans[i].DeptCode.String,
			Id:               plans[i].Id,
			PlanId:           plans[i].PlanId,
			PlanName:         plans[i].PlanName.String,
			Type:             plans[i].Type.String,
			GroupId:          plans[i].GroupId.String,
			Description:      plans[i].Description.String,
			StartTime:        carbon.CreateFromStdTime(plans[i].StartTime).ToDateTimeString(),
			EndTime:          carbon.CreateFromStdTime(plans[i].EndTime).ToDateTimeString(),
			Rule:             &pbRule,
			Status:           int32(plans[i].Status),
			ScanFlg:          int32(plans[i].ScanFlg),
			TerminatedReason: plans[i].TerminatedReason.String,
			PausedReason:     plans[i].PausedReason.String,
			Ext1:             plans[i].Ext1.String,
			Ext2:             plans[i].Ext2.String,
			Ext3:             plans[i].Ext3.String,
			Ext4:             plans[i].Ext4.String,
			Ext5:             plans[i].Ext5.String,
		}

		// 设置暂停时间和原因
		if plans[i].PausedTime.Valid {
			pbPlan.PausedTime = carbon.CreateFromStdTime(plans[i].PausedTime.Time).ToDateTimeString()
		}

		progress, err := gormmodel.CalculatePlanProgress(l.ctx, rawDB, plans[i].Id)
		if err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "计算计划进度失败")
		}
		pbPlan.Progress = progress

		resp.Plans = append(resp.Plans, pbPlan)
	}

	return resp, nil
}
