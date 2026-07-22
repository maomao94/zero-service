package logic

import (
	"context"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/common/ossx"
	"zero-service/model/gormmodel"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateOssLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateOssLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateOssLogic {
	return &CreateOssLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateOssLogic) CreateOss(in *file.CreateOssReq) (*file.CreateOssRes, error) {
	oss := &gormmodel.Oss{
		TenantId:   in.TenantId,
		Category:   int(in.Category),
		OssCode:    in.OssCode,
		Endpoint:   in.Endpoint,
		AccessKey:  in.AccessKey,
		SecretKey:  in.SecretKey,
		BucketName: in.BucketName,
		AppId:      in.AppId,
		Region:     in.Region,
		Remark:     in.Remark,
		Status:     OssStatusEnabled,
	}
	if err := l.svcCtx.DB.WithContext(l.ctx).Create(oss).Error; err != nil {
		return nil, err
	}
	ossx.CacheInvalidate(in.TenantId)
	return &file.CreateOssRes{Id: oss.Id}, nil
}
