package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/app/djicloud/model/gormmodel"
	"zero-service/common/carbonx"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type GetFlightTaskProgressLastLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetFlightTaskProgressLastLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFlightTaskProgressLastLogic {
	return &GetFlightTaskProgressLastLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetFlightTaskProgressLast 查询机巢当前航线任务状态。
func (l *GetFlightTaskProgressLastLogic) GetFlightTaskProgressLast(in *djicloud.GetFlightTaskProgressLastReq) (*djicloud.GetFlightTaskProgressLastRes, error) {
	var progress gormmodel.DjiDockDeviceFlightTaskState
	err := l.svcCtx.DB.WithContext(l.ctx).
		Where("gateway_sn = ?", in.DeviceSn).
		First(&progress).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &djicloud.GetFlightTaskProgressLastRes{}, nil
		}
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_02_DB, err, "查询最近航线任务进度失败")
	}
	return &djicloud.GetFlightTaskProgressLastRes{
		HasProgress:  true,
		ReportedAt:   carbonx.FormatDateTimeMicro(progress.ReportedAt),
		ProgressJson: progress.RawJSON,
	}, nil
}
