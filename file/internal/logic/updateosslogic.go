package logic

import (
	"context"

	"zero-service/file/file"
	"zero-service/file/internal/svc"

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
	// todo: add your logic here and delete this line

	return &file.UpdateOssRes{}, nil
}
