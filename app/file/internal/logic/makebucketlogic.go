package logic

import (
	"context"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
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
	ossTemplate, err := l.svcCtx.GetOssTemplate(l.ctx, in.TenantId, in.Code)
	if err != nil {
		return nil, err
	}
	exists, err := ossTemplate.BucketExists(l.ctx, in.TenantId, in.BucketName)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err = ossTemplate.MakeBucket(l.ctx, in.TenantId, in.BucketName); err != nil {
			return nil, err
		}
	}
	return &file.MakeBucketRes{}, nil
}
