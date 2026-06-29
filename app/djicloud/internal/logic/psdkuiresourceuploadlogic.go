package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type PsdkUIResourceUploadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPsdkUIResourceUploadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PsdkUIResourceUploadLogic {
	return &PsdkUIResourceUploadLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// PsdkUIResourceUpload PSDK UI 资源上传。
func (l *PsdkUIResourceUploadLogic) PsdkUIResourceUpload(in *djicloud.PsdkUIResourceUploadReq) (*djicloud.CommonRes, error) {
	data := &djisdk.PsdkUIResourceUploadData{
		Name:        in.GetName(),
		URL:         in.GetUrl(),
		Fingerprint: in.GetFingerprint(),
	}
	tid, err := l.svcCtx.DjiClient.PsdkUIResourceUpload(l.ctx, in.GetDeviceSn(), data)
	if err != nil {
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
