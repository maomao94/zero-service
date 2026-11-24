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
	db := l.svcCtx.DB.WithContext(l.ctx).DB
	var oss gormmodel.Oss
	if err := db.Where("id = ?", in.Id).First(&oss).Error; err != nil {
		return nil, err
	}
	// 只更新本次提交的列：整行 Save 在并发删除时会回退成 upsert，把已删除记录重新写回。
	result := db.Model(&gormmodel.Oss{}).
		Where("id = ?", oss.Id).
		Updates(map[string]any{
			"tenant_id":   in.TenantId,
			"category":    int(in.Category),
			"oss_code":    in.OssCode,
			"endpoint":    in.Endpoint,
			"access_key":  in.AccessKey,
			"secret_key":  in.SecretKey,
			"bucket_name": in.BucketName,
			"app_id":      in.AppId,
			"region":      in.Region,
			"remark":      in.Remark,
			"status":      int(in.Status),
		})
	if result.Error != nil {
		return nil, result.Error
	}
	ossx.CacheInvalidate(in.TenantId)
	return &file.UpdateOssRes{}, nil
}
