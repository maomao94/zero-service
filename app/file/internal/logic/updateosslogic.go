package logic

import (
	"context"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/common/ossx"
	"zero-service/model/gormmodel"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateOssLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateOssLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateOssLogic {
	return &UpdateOssLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateOssLogic) UpdateOss(in *file.UpdateOssReq) (*file.UpdateOssRes, error) {
	var oss gormmodel.Oss
	if err := l.svcCtx.DB.WithContext(l.ctx).First(&oss, in.Id).Error; err != nil {
		return nil, err
	}
	oss.TenantId = in.TenantId
	oss.Category = in.Category
	oss.OssCode = in.OssCode
	oss.Endpoint = in.Endpoint
	oss.AccessKey = in.AccessKey
	oss.SecretKey = in.SecretKey
	oss.BucketName = in.BucketName
	oss.AppId = in.AppId
	oss.Region = in.Region
	oss.Remark = in.Remark
	oss.Status = in.Status
	if err := l.svcCtx.DB.WithContext(l.ctx).Save(&oss).Error; err != nil {
		return nil, err
	}
	ossx.CacheInvalidate(in.TenantId)
	return &file.UpdateOssRes{}, nil
}
