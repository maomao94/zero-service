package logic

import (
	"context"
	"errors"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_02_RECORD_NOT_EXIST, "未找到飞行区配置")
		}
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询飞行区配置失败")
	}

	gatewaySn := region.GatewaySn

	if err := l.svcCtx.DB.WithContext(l.ctx).Delete(&region).Error; err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "删除飞行区配置失败")
	}

	tid, err := l.svcCtx.DjiClient.FlightAreasUpdate(l.ctx, gatewaySn)
	if err != nil {
		message, reasonCode, err := commandError(err)
		if err != nil {
			return nil, err
		}
		return &djicloud.DeleteCustomFlyRegionByFileIdRes{Code: -1, Message: message, Tid: tid, ReasonCode: reasonCode}, nil
	}

	return &djicloud.DeleteCustomFlyRegionByFileIdRes{Code: 0, Message: "success", Tid: tid}, nil
}
