package logic

import (
	"context"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type SignUrlLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSignUrlLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SignUrlLogic {
	return &SignUrlLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SignUrlLogic) SignUrl(in *file.SignUrlReq) (*file.SignUrlRes, error) {
	type validateReq struct {
		TenantId string `validate:"required"`
		Filename string `validate:"required"`
	}
	var rule validateReq
	copier.Copy(&rule, in) // nolint:errcheck
	if err := l.svcCtx.Validate.StructCtx(l.ctx, rule); err != nil {
		return nil, err
	}
	ossTemplate, err := l.svcCtx.GetOssTemplate(l.ctx, in.TenantId, in.Code)
	if err != nil {
		return nil, err
	}
	signUrl, err := ossTemplate.SignUrl(l.ctx, in.TenantId, in.BucketName, in.Filename, calcExpires(in.Expires))
	if err != nil {
		return nil, err
	}
	return &file.SignUrlRes{
		Url: signUrl,
	}, nil
}
