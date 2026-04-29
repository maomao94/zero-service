package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type OtaCreateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewOtaCreateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OtaCreateLogic {
	return &OtaCreateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// OtaCreate 创建固件升级任务。
func (l *OtaCreateLogic) OtaCreate(in *djicloud.OtaCreateReq) (*djicloud.CommonRes, error) {
	devices := make([]djisdk.OtaDevice, 0, len(in.Devices))
	for _, d := range in.Devices {
		devices = append(devices, djisdk.OtaDevice{
			SN:                  d.Sn,
			ProductVersion:      d.ProductVersion,
			FirmwareUpgradeType: int(d.FirmwareUpgradeType),
		})
	}
	data := &djisdk.OtaCreateData{Devices: devices}
	tid, err := l.svcCtx.DjiClient.OtaCreate(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[ota] ota create failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
