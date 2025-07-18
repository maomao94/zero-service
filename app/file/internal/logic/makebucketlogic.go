package logic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/common/ossx"
	"zero-service/model"
)

type MakeBucketLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMakeBucketLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MakeBucketLogic {
	return &MakeBucketLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *MakeBucketLogic) MakeBucket(in *file.MakeBucketReq) (*file.MakeBucketRes, error) {
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
	if !bool {
		err = ossTemplate.MakeBucket(l.ctx, in.TenantId, in.BucketName)
		if err != nil {
			return nil, err
		}
	}
	return &file.MakeBucketRes{}, nil
}
