package logic

import (
	"context"
	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/model"

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
	_, err := l.svcCtx.OssModel.Update(l.ctx, nil, &model.Oss{
		Id:         in.Id,
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
		Status:     in.Status,
	})
	if err != nil {
		return nil, err
	}
	return &file.UpdateOssRes{}, nil
}
