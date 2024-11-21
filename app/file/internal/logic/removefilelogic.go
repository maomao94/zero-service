package logic

import (
	"context"
	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/model"
	"zero-service/ossx"

	"github.com/zeromicro/go-zero/core/logx"
)

type RemoveFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRemoveFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveFileLogic {
	return &RemoveFileLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RemoveFileLogic) RemoveFile(in *file.RemoveFileReq) (*file.RemoveFileRes, error) {
	ossTemplate, err := ossx.Template(in.TenantId, in.Code, l.svcCtx.Config.Oss.TenantMode, func(tenantId, code string) (oss *model.Oss, err error) {
		return l.svcCtx.OssModel.FindOneByTenantIdOssCode(l.ctx, in.TenantId, in.Code)
	})
	if err != nil {
		return nil, err
	}
	err = ossTemplate.RemoveFile(in.TenantId, in.BucketName, in.Filename)
	if err != nil {
		return nil, err
	}
	return &file.RemoveFileRes{}, nil
}