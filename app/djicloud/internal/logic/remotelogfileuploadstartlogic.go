package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type RemoteLogFileUploadStartLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRemoteLogFileUploadStartLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoteLogFileUploadStartLogic {
	return &RemoteLogFileUploadStartLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// RemoteLogFileUploadStart 开始上传远程日志文件。
func (l *RemoteLogFileUploadStartLogic) RemoteLogFileUploadStart(in *djicloud.RemoteLogFileUploadReq) (*djicloud.CommonRes, error) {
	files := make([]djisdk.RemoteLogFile, 0, len(in.Files))
	for _, f := range in.Files {
		files = append(files, djisdk.RemoteLogFile{
			DeviceSN: f.DeviceSn,
			Module:   f.Module,
			Key:      f.Key,
			Name:     f.Name,
			URL:      f.Url,
			Size:     f.Size,
		})
	}
	data := &djisdk.RemoteLogFileUploadStartData{Files: files}
	tid, err := l.svcCtx.DjiClient.RemoteLogFileUploadStart(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[remote-log] file upload start failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
