package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type UnlockLicenseSwitchLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUnlockLicenseSwitchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnlockLicenseSwitchLogic {
	return &UnlockLicenseSwitchLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// UnlockLicenseSwitch 启用或禁用设备的单个解禁证书。
func (l *UnlockLicenseSwitchLogic) UnlockLicenseSwitch(in *djicloud.UnlockLicenseSwitchReq) (*djicloud.CommonRes, error) {
	data := &djisdk.UnlockLicenseSwitchData{LicenseID: in.GetLicenseId(), Enable: in.GetEnable()}
	tid, err := l.svcCtx.DjiClient.UnlockLicenseSwitch(l.ctx, in.GetDeviceSn(), data)
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
