package logic

import (
	"context"

	"zero-service/app/djicloud/djicloud"
	"zero-service/app/djicloud/internal/svc"
	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/logx"
)

type RemoteLogFileUploadUpdateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRemoteLogFileUploadUpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoteLogFileUploadUpdateLogic {
	return &RemoteLogFileUploadUpdateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// RemoteLogFileUploadUpdate 更新远程日志文件上传任务。
func (l *RemoteLogFileUploadUpdateLogic) RemoteLogFileUploadUpdate(in *djicloud.RemoteLogFileUploadReq) (*djicloud.CommonRes, error) {
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
	data := &djisdk.RemoteLogFileUploadUpdateData{Files: files}
	tid, err := l.svcCtx.DjiClient.RemoteLogFileUploadUpdate(l.ctx, in.DeviceSn, data)
	if err != nil {
		l.Errorf("[remote-log] file upload update failed: %v", err)
		return errRes(tid, err), nil
	}
	return okRes(tid), nil
}
