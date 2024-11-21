package logic

import (
	"context"
	"time"
	"zero-service/model"
	"zero-service/ossx"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"

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
	signUrl, err := ossTemplate.SignUrl(in.TenantId, in.BucketName, in.Filename, expires)
	if err != nil {
		return nil, err
	}
	return &file.SignUrlRes{
		Url: signUrl,
	}, nil
}
