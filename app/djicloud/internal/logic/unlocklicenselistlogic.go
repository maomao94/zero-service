package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type UnlockLicenseListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUnlockLicenseListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnlockLicenseListLogic {
	return &UnlockLicenseListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// UnlockLicenseList 获取设备解禁证书列表。
func (l *UnlockLicenseListLogic) UnlockLicenseList(in *djicloud.UnlockLicenseListReq) (*djicloud.CommonRes, error) {
	data := &djisdk.UnlockLicenseListData{DeviceModelDomain: int(in.GetDeviceModelDomain())}
	tid, err := l.svcCtx.DjiClient.UnlockLicenseList(l.ctx, in.GetDeviceSn(), data)
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
