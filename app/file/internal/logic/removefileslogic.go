package logic

import (
	"context"
	"fmt"
	"strings"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type RemoveFilesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRemoveFilesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveFilesLogic {
	return &RemoveFilesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RemoveFilesLogic) RemoveFiles(in *file.RemoveFilesReq) (*file.RemoveFileRes, error) {
	ossTemplate, err := l.svcCtx.GetOssTemplate(l.ctx, in.TenantId, in.Code)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "获取OSS模板失败")
	}
	results, err := ossTemplate.RemoveFiles(l.ctx, in.TenantId, in.BucketName, in.Filename)
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "批量删除OSS文件失败")
	}
	var errs []string
	for _, r := range results {
		if r.Err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", r.Filename, r.Err))
		}
	}
	if len(errs) > 0 {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_06_THIRD_PARTY, fmt.Sprintf("failed to remove files: %s", strings.Join(errs, "; ")))
	}
	return &file.RemoveFileRes{}, nil
}
