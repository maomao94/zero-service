package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type FlightAreasUpdateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFlightAreasUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FlightAreasUpdateLogic {
	return &FlightAreasUpdateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// FlightAreasUpdate 触发自定义飞行区文件更新。
func (l *FlightAreasUpdateLogic) FlightAreasUpdate(in *djicloud.FlightAreasUpdateReq) (*djicloud.CommonRes, error) {
	file := in.GetFile()
	data := &djisdk.FlightAreasUpdateData{
		File: &djisdk.UnlockLicenseFile{
			URL:         file.GetUrl(),
			Fingerprint: file.GetFingerprint(),
		},
	}
	tid, err := l.svcCtx.DjiClient.FlightAreasUpdate(l.ctx, in.GetDeviceSn(), data)
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
