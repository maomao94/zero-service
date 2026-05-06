package logic

import (
	"context"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/model/gormmodel"

	"github.com/zeromicro/go-zero/core/logx"
)

type OssDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewOssDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OssDetailLogic {
	return &OssDetailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *OssDetailLogic) OssDetail(in *file.OssDetailReq) (*file.OssDetailRes, error) {
	var oss gormmodel.Oss
	if err := l.svcCtx.DB.WithContext(l.ctx).First(&oss, in.Id).Error; err != nil {
		return nil, err
	}
	return &file.OssDetailRes{Oss: toPbOss(&oss)}, nil
}
