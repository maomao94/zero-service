package logic

import (
	"context"
	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteOssLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteOssLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteOssLogic {
	return &DeleteOssLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteOssLogic) DeleteOss(in *file.DeleteOssReq) (*file.DeleteOssRes, error) {
	oss, err := l.svcCtx.OssModel.FindOne(l.ctx, in.Id)
	if err != nil {
		return nil, err
	}
	err = l.svcCtx.OssModel.DeleteSoft(l.ctx, nil, oss.Id)
	if err != nil {
		return nil, err
	}
	return &file.DeleteOssRes{Id: oss.Id}, nil
}
