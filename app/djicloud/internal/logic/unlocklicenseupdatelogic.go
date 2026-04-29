package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type UnlockLicenseUpdateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUnlockLicenseUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnlockLicenseUpdateLogic {
	return &UnlockLicenseUpdateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// UnlockLicenseUpdate 更新设备的解禁证书。
func (l *UnlockLicenseUpdateLogic) UnlockLicenseUpdate(in *djicloud.UnlockLicenseUpdateReq) (*djicloud.CommonRes, error) {
	data := &djisdk.UnlockLicenseUpdateData{}
	if in.GetFile() != nil {
		data.File = &djisdk.UnlockLicenseFile{URL: in.GetFile().GetUrl(), Fingerprint: in.GetFile().GetFingerprint()}
	}
	tid, err := l.svcCtx.DjiClient.UnlockLicenseUpdate(l.ctx, in.GetDeviceSn(), data)
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
