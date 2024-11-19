package logic

import (
	"context"

	"zero-service/file/file"
	"zero-service/file/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type OssListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewOssListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OssListLogic {
	return &OssListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *OssListLogic) OssList(in *file.OssListReq) (*file.OssListRes, error) {
	// todo: add your logic here and delete this line

	return &file.OssListRes{}, nil
}
