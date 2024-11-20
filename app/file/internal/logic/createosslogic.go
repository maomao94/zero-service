package logic

import (
	"context"
	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/model"

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
	oss, err := l.svcCtx.OssModel.Insert(l.ctx, nil, &model.Oss{
		TenantId:   in.TenantId,
		Category:   in.Category,
		OssCode:    in.OssCode,
		Endpoint:   in.Endpoint,
		AccessKey:  in.AccessKey,
		SecretKey:  in.SecretKey,
		BucketName: in.BucketName,
		AppId:      in.AppId,
		Region:     in.Region,
		Remark:     in.Remark,
		Status:     2,
	})
	if err != nil {
		return nil, err
	}
	id, _ := oss.LastInsertId()
	return &file.CreateOssRes{Id: id}, nil
}
