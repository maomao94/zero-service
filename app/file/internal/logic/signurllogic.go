package logic

import (
	"context"
	"github.com/jinzhu/copier"
	"time"
	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"
	"zero-service/common/ossx"
	"zero-service/model"

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
	copier.Copy(&rule, in)
	if err := l.svcCtx.Validate.StructCtx(l.ctx, rule); err != nil {
		return nil, err
	}
	ossTemplate, err := ossx.Template(in.TenantId, in.Code, l.svcCtx.Config.Oss.TenantMode, func(tenantId, code string) (oss *model.Oss, err error) {
		return l.svcCtx.OssModel.FindOneByTenantIdOssCode(l.ctx, in.TenantId, in.Code)
	})
	if err != nil {
		return nil, err
	}
	//_, err = ossTemplate.StatFile(in.TenantId, in.BucketName, in.Filename)
	//if err != nil {
	//	return nil, err
	//}
	expires := 60 * time.Minute // 1 小时
	if in.Expires > 0 {
		expires = time.Duration(in.Expires) * time.Minute
	}
	signUrl, err := ossTemplate.SignUrl(l.ctx, in.TenantId, in.BucketName, in.Filename, expires)
	if err != nil {
		return nil, err
	}
	return &file.SignUrlRes{
		Url: signUrl,
	}, nil
}
