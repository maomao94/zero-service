package logic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/common/ossx"
	"zero-service/model"
)

type RemoveBucketLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRemoveBucketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveBucketLogic {
	return &RemoveBucketLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RemoveBucketLogic) RemoveBucket(in *file.RemoveBucketReq) (*file.RemoveBucketRes, error) {
	ossTemplate, err := ossx.Template(in.TenantId, in.Code, l.svcCtx.Config.Oss.TenantMode, func(tenantId, code string) (oss *model.Oss, err error) {
		return l.svcCtx.OssModel.FindOneByTenantIdOssCode(l.ctx, in.TenantId, in.Code)
	})
	if err != nil {
		return nil, err
	}
	bool, err := ossTemplate.BucketExists(l.ctx, in.TenantId, in.BucketName)
	if err != nil {
		return nil, err
	}
	if bool {
		err = ossTemplate.RemoveBucket(l.ctx, in.TenantId, in.BucketName)
		if err != nil {
			return nil, err
		}
	}
	return &file.RemoveBucketRes{}, nil
}
