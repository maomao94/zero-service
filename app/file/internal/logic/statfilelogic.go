package logic

import (
	"context"

	"zero-service/app/file/file"
	"zero-service/app/file/internal/svc"

	"github.com/dromara/carbon/v2"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type StatFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStatFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StatFileLogic {
	return &StatFileLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *StatFileLogic) StatFile(in *file.StatFileReq) (*file.StatFileRes, error) {
	ossTemplate, err := l.svcCtx.GetOssTemplate(l.ctx, in.TenantId, in.Code)
	if err != nil {
		return nil, err
	}
	ossFile, err := ossTemplate.StatFile(l.ctx, in.TenantId, in.BucketName, in.Filename)
	if err != nil {
		return nil, err
	}
	var resOssFile file.OssFile
	copier.Copy(&resOssFile, ossFile) // nolint:errcheck
	resOssFile.PutTime = carbon.CreateFromStdTime(ossFile.PutTime).Format(carbon.DateTimeMicroFormat)
	if in.IsSign {
		signUrl, err := ossTemplate.SignUrl(l.ctx, in.TenantId, in.BucketName, in.Filename, calcExpires(in.Expires))
		if err != nil {
			return nil, err
		}
		resOssFile.SignUrl = signUrl
	}
	return &file.StatFileRes{
		OssFile: &resOssFile,
	}, nil
}
