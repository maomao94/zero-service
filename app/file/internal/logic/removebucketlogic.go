package logic

import (
	"context"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
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
	ossTemplate, err := l.svcCtx.GetOssTemplate(l.ctx, in.TenantId, in.Code)
	if err != nil {
		return nil, err
	}
	exists, err := ossTemplate.BucketExists(l.ctx, in.TenantId, in.BucketName)
	if err != nil {
		return nil, err
	}
	if exists {
		if err = ossTemplate.RemoveBucket(l.ctx, in.TenantId, in.BucketName); err != nil {
			return nil, err
		}
	}
	return &file.RemoveBucketRes{}, nil
}
