package logic

import (
	"context"

	"zero-service/app/djigateway/djigateway"
	"zero-service/app/djigateway/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type MediaFastUploadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMediaFastUploadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MediaFastUploadLogic {
	return &MediaFastUploadLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// MediaFastUpload 快速上传指定媒体文件。
func (l *MediaFastUploadLogic) MediaFastUpload(in *djigateway.MediaFastUploadReq) (*djigateway.CommonRes, error) {
	data := &djisdk.MediaFastUploadData{FileID: in.FileId}
	tid, err := l.svcCtx.DjiClient.MediaFastUpload(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[media] fast upload failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
