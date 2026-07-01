package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/gormx"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListFlyRegionSyncStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListFlyRegionSyncStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListFlyRegionSyncStatusLogic {
	return &ListFlyRegionSyncStatusLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ListFlyRegionSyncStatus 分页查询飞行区同步状态。
func (l *ListFlyRegionSyncStatusLogic) ListFlyRegionSyncStatus(in *djicloud.ListFlyRegionSyncStatusReq) (*djicloud.ListFlyRegionSyncStatusRes, error) {
	db := l.svcCtx.DB.WithContext(l.ctx).Model(&gormmodel.DjiFlyRegionSyncStatus{})
	if in.GetGatewaySn() != "" {
		db = db.Where("gateway_sn = ?", in.GetGatewaySn())
	}
	if in.GetSyncStatus() != "" {
		db = db.Where("sync_status = ?", in.GetSyncStatus())
	}

	var statuses []gormmodel.DjiFlyRegionSyncStatus
	pageResult, err := gormx.QueryPage(db.Order("id DESC"), int(in.GetPage()), int(in.GetPageSize()), &statuses)
	if err != nil {
		return nil, err
	}

	list := make([]*djicloud.FlyRegionSyncStatusInfo, 0, len(statuses))
	for i := range statuses {
		s := statuses[i]
		list = append(list, &djicloud.FlyRegionSyncStatusInfo{
			Id:          s.Id,
			GatewaySn:   s.GatewaySn,
			FlyRegionId: s.FlyRegionID,
			SyncStatus:  s.SyncStatus,
			SyncReason:  int32(s.SyncReason),
			CreateTime:  s.CreateTime.UnixMilli(),
			UpdateTime:  s.UpdateTime.UnixMilli(),
		})
	}

	return &djicloud.ListFlyRegionSyncStatusRes{
		Total: int64(pageResult.Total),
		List:  list,
	}, nil
}
