package logic

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/common/ossx"
	"zero-service/model"
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
	ossTemplate, err := ossx.Template(in.TenantId, in.Code, l.svcCtx.Config.Oss.TenantMode, func(tenantId, code string) (oss *model.Oss, err error) {
		return l.svcCtx.OssModel.FindOneByTenantIdOssCode(l.ctx, in.TenantId, in.Code)
	})
	if err != nil {
		return nil, err
	}
	results, err := ossTemplate.RemoveFiles(l.ctx, in.TenantId, in.BucketName, in.Filename)
	if err != nil {
		return nil, err
	}
	for _, r := range results {
		if r.Err != nil {
			return nil, fmt.Errorf("failed to remove file %s: %w", r.Filename, r.Err)
		}
	}
	return &file.RemoveFileRes{}, nil
}
