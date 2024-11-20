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
	// todo: add your logic here and delete this line

	return &file.RemoveBucketRes{}, nil
}
