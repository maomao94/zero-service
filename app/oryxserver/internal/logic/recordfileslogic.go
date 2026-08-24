package logic

import (
	"context"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type RecordFilesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRecordFilesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecordFilesLogic {
	return &RecordFilesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 列出录制文件
func (l *RecordFilesLogic) RecordFiles(in *oryxserver.RecordFilesReq) (*oryxserver.RecordFilesRes, error) {
	files, err := l.svcCtx.OryxClient.RecordFiles(l.ctx)
	if err != nil {
		l.Logger.Errorf("调用 Oryx API 失败: %v", err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "调用 Oryx API 失败")
	}
	list := make([]*oryxserver.RecordFile, 0, len(files))
	for _, f := range files {
		list = append(list, &oryxserver.RecordFile{
			Uuid:     f.UUID,
			Vhost:    f.Vhost,
			App:      f.App,
			Stream:   f.Stream,
			Progress: f.Progress,
			Update:   f.Update,
			Nn:       f.NN,
			Duration: f.Duration,
			Size:     f.Size,
		})
	}
	return &oryxserver.RecordFilesRes{Files: list}, nil
}
