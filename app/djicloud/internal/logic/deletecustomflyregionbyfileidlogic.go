package logic

import (
	"context"
	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/app/djicloud/model/gormmodel"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteCustomFlyRegionByFileIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteCustomFlyRegionByFileIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteCustomFlyRegionByFileIdLogic {
	return &DeleteCustomFlyRegionByFileIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// DeleteCustomFlyRegionByFileId 按文件 ID 删除指定飞行区。
func (l *DeleteCustomFlyRegionByFileIdLogic) DeleteCustomFlyRegionByFileId(in *djicloud.DeleteCustomFlyRegionByFileIdReq) (*djicloud.DeleteCustomFlyRegionByFileIdRes, error) {
	fileId := in.GetFileId()

	var region gormmodel.DjiFlyRegion
	if err := l.svcCtx.DB.WithContext(l.ctx).Where("file_id = ?", fileId).First(&region).Error; err != nil {
		return nil, err
	}

	gatewaySn := region.GatewaySn

	if err := l.svcCtx.DB.WithContext(l.ctx).Delete(&region).Error; err != nil {
		return nil, err
	}

	tid, err := l.svcCtx.DjiClient.FlightAreasUpdate(l.ctx, gatewaySn)
	if err != nil {
		return &djicloud.DeleteCustomFlyRegionByFileIdRes{Code: -1, Message: err.Error(), Tid: tid}, nil
	}

	return &djicloud.DeleteCustomFlyRegionByFileIdRes{Code: 0, Message: "success", Tid: tid}, nil
}
