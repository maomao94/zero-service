package logic

import (
	"context"

	"zero-service/file/file"
	"zero-service/file/internal/svc"

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
	// todo: add your logic here and delete this line

	return &file.CreateOssRes{}, nil
}
